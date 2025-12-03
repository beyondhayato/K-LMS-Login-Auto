package browser

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/playwright-community/playwright-go"
)

// ファイルパス定義（フォルダ分け）
const (
	CookieFile     = "data/state.json"
	DebugTextFile  = "logs/debug_last_text.txt"
	ScreenshotFile = "data/screenshot.png"
)

type CheckResult struct {
	Hash           string
	ScreenshotPath string
	HasDiff        bool
}

func CheckKLMSTask(oldHash string) (*CheckResult, error) {
	// フォルダが存在しないとエラーになる可能性があるので、念のため作成しておく
	_ = os.MkdirAll("data", 0755)
	_ = os.MkdirAll("logs", 0755)

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("Playwright起動エラー: %v", err)
	}
	
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true), // デバッグ中はfalse推奨
	})
	if err != nil {
		return nil, fmt.Errorf("ブラウザ起動エラー: %v", err)
	}
	defer browser.Close()

	// Cookie読み込み先を変更
	contextOptions := playwright.BrowserNewContextOptions{}
	if _, err := os.Stat(CookieFile); err == nil {
		contextOptions.StorageStatePath = playwright.String(CookieFile)
	}
	
	context, err := browser.NewContext(contextOptions)
	if err != nil {
		return nil, fmt.Errorf("コンテキスト作成エラー: %v", err)
	}

	page, err := context.NewPage()
	if err != nil {
		return nil, fmt.Errorf("ページ作成エラー: %v", err)
	}

	log.Println("🌐 アクセス中: https://lms.keio.jp")
	if _, err := page.Goto("https://lms.keio.jp", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return nil, fmt.Errorf("ページ遷移エラー: %v", err)
	}

	// === ログイン処理 ===
	keioLink, _ := page.QuerySelector("a:has-text(\"keio.jp\")")
	if keioLink != nil {
		log.Println("🔗 keio.jpリンクをクリック")
		keioLink.Click(playwright.ElementHandleClickOptions{Force: playwright.Bool(true)})

		page.WaitForSelector("input[type=\"text\"]")
		page.Fill("input[type=\"text\"]", os.Getenv("KEIO_USER"))
		// 【最強のEnter連打】ここは絶対に変えません
		page.Press("input[type=\"text\"]", "Enter")

		page.WaitForSelector("input[type=\"password\"]")
		page.Fill("input[type=\"password\"]", os.Getenv("KEIO_PASS"))
		// 【最強のEnter連打】ここも絶対に変えません
		page.Press("input[type=\"password\"]", "Enter")
		
		// ログイン後の待機
		page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
			State: playwright.LoadStateDomcontentloaded,
		})
		context.StorageState(CookieFile) // 保存
	}

	// === ダッシュボード待機（ここを修正） ===
	// 以前: #dashboard を待っていた
	// 修正: #planner-today-btn (本日ボタン) を待つことで「中身」の読み込み完了を保証
	log.Println("⏳ ダッシュボード(本日ボタン)待機中...")
	
	if _, err := page.WaitForSelector("#planner-today-btn", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(90000), // 90秒
	}); err != nil {
		// タイムアウト時の詳細ログ
		log.Printf("⚠️ 待機タイムアウト。現在のURL: %s", page.URL())
		return nil, fmt.Errorf("ダッシュボード到達タイムアウト: %v", err)
	}

	// 念のためネットワークが落ち着くまで待機
	page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})

	// === ハッシュ化 ===
	targetSelector := "#dashboard"
	// カレンダー表示かリスト表示かを判定して対象を変えるロジック（そのまま維持）
	if listItems, _ := page.QuerySelector(".planner-day"); listItems != nil {
		targetSelector = "#dashboard-planner"
	}

	log.Printf("🎯 監視対象: %s", targetSelector)
	bodyText, err := page.InnerText(targetSelector)
	if err != nil {
		return nil, fmt.Errorf("テキスト取得エラー: %v", err)
	}

	// デバッグ用ログをlogsフォルダへ
	ioutil.WriteFile(DebugTextFile, []byte(bodyText), 0644)

	hashBytes := sha256.Sum256([]byte(bodyText))
	newHash := hex.EncodeToString(hashBytes[:])
	log.Printf("🔍 新ハッシュ: %s", newHash[:10])

	if newHash == oldHash {
		log.Println("🟦 変更なし")
		return &CheckResult{Hash: newHash, HasDiff: false}, nil
	}

	// スクショ保存先をdataフォルダへ
	log.Println("🟥 変更検知！スクショを撮ります")
	if _, err := page.Screenshot(playwright.PageScreenshotOptions{
		Path:     playwright.String(ScreenshotFile),
		FullPage: playwright.Bool(true),
	}); err != nil {
		return nil, fmt.Errorf("スクショ失敗: %v", err)
	}

	return &CheckResult{
		Hash:           newHash,
		ScreenshotPath: ScreenshotFile,
		HasDiff:        true,
	}, nil
}