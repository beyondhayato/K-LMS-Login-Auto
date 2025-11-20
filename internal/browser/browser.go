package browser

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/playwright-community/playwright-go"
)

// CheckResult はブラウザ操作の結果をまとめる「報告書」のフォーマットです
type CheckResult struct {
	Hash           string // ページのハッシュ値（指紋）
	ScreenshotPath string // スクショを保存したパス（撮った場合）
	HasDiff        bool   // 差分があったかどうか
}

// CheckKLMSTask はK-LMSにアクセスしてハッシュを確認し、差分があればスクショを撮ります
func CheckKLMSTask(oldHash string) (*CheckResult, error) {
	// 1. Playwright（ブラウザ操縦士）を起動
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("Playwright起動エラー: %v", err)
	}
	
	// 2. ブラウザ（Chrome）を起動
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true), // trueなら画面なしで実行
	})
	if err != nil {
		return nil, fmt.Errorf("ブラウザ起動エラー: %v", err)
	}
	defer browser.Close()

	// 3. Cookie（state.json）を読み込んでコンテキスト作成
	contextOptions := playwright.BrowserNewContextOptions{}
	// state.jsonが存在するか確認
	if _, err := os.Stat("state.json"); err == nil {
		contextOptions.StorageStatePath = playwright.String("state.json")
	}
	
	context, err := browser.NewContext(contextOptions)
	if err != nil {
		return nil, fmt.Errorf("コンテキスト作成エラー: %v", err)
	}

	// 4. ページを開く
	page, err := context.NewPage()
	if err != nil {
		return nil, fmt.Errorf("ページ作成エラー: %v", err)
	}

	// K-LMSへアクセス
	log.Println("🌐 アクセス中: https://lms.keio.jp/")
	if _, err := page.Goto("https://lms.keio.jp/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return nil, fmt.Errorf("ページ遷移エラー: %v", err)
	}

	// === ログイン処理 (keio.jpが表示された場合のみ実行) ===
	keioLink, _ := page.QuerySelector("a:has-text(\"keio.jp\")")
	if keioLink != nil {
		log.Println("🔗 keio.jpリンクをクリック")
		keioLink.Click()

		// ID入力
		page.WaitForSelector("input[type=\"text\"]")
		page.Fill("input[type=\"text\"]", os.Getenv("KEIO_USER"))
		page.Click("button[type=\"submit\"]")

		// PASS入力
		page.WaitForSelector("input[type=\"password\"]")
		page.Fill("input[type=\"password\"]", os.Getenv("KEIO_PASS"))
		page.Click("button[type=\"submit\"]")
		
		// ログイン成功したらCookieを保存
		// ページ遷移を少し待つ
		page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
			State: playwright.LoadStateNetworkidle,
		})
		context.StorageState("state.json")
	}

	// === ダッシュボード待機 ===
	log.Println("⏳ ダッシュボード待機中...")
	if _, err := page.WaitForSelector("#global_nav_dashboard_link", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(90000), // 90秒待つ
	}); err != nil {
		return nil, fmt.Errorf("ダッシュボード到達タイムアウト: %v", err)
	}

	// 安定するまで待機
	page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})

	// === ハッシュ化 (指紋採取) ===
	bodyText, err := page.InnerText("body")
	if err != nil {
		return nil, fmt.Errorf("テキスト取得エラー: %v", err)
	}

	// SHA-256でハッシュ化
	hashBytes := sha256.Sum256([]byte(bodyText))
	newHash := hex.EncodeToString(hashBytes[:])

	log.Printf("🔍 新ハッシュ: %s", newHash[:10])

	// 差分チェック
	if newHash == oldHash {
		log.Println("🟦 変更なし")
		return &CheckResult{Hash: newHash, HasDiff: false}, nil
	}

	// === 変更ありの場合: スクショ撮影 ===
	log.Println("🟥 変更検知！スクショを撮ります")
	screenshotPath := "klms_after_login.png"
	if _, err := page.Screenshot(playwright.PageScreenshotOptions{
		Path:     playwright.String(screenshotPath),
		FullPage: playwright.Bool(true),
	}); err != nil {
		return nil, fmt.Errorf("スクショ失敗: %v", err)
	}

	return &CheckResult{
		Hash:           newHash,
		ScreenshotPath: screenshotPath,
		HasDiff:        true,
	}, nil
}