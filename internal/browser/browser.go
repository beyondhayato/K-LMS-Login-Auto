package browser

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/playwright-community/playwright-go"
)

// ファイルパス定義（フォルダ分け）
const (
	CookieFile     = "data/state.json"
	DebugTextFile  = "logs/debug_last_text.txt"
	ScreenshotFile = "data/screenshot.png"
)

// 設定定数
const (
	MaxRetries           = 3              // 最大リトライ回数
	RetryDelay           = 5 * time.Second // リトライ間隔
	DefaultTimeout       = 120000          // デフォルトタイムアウト（120秒）
	NetworkIdleTimeout   = 30000           // ネットワークアイドル待機タイムアウト（30秒）
)

type CheckResult struct {
	Hash           string
	ScreenshotPath string
	HasDiff        bool
}

// CheckKLMSTask はK-LMSをチェックします（リトライ機能付き）
func CheckKLMSTask(oldHash string) (*CheckResult, error) {
	var lastErr error
	
	// リトライループ
	for attempt := 1; attempt <= MaxRetries; attempt++ {
		if attempt > 1 {
			log.Printf("🔄 リトライ %d/%d 回目（%v 待機後）", attempt, MaxRetries, RetryDelay)
			time.Sleep(RetryDelay)
		}
		
		result, err := checkKLMSTaskOnce(oldHash, attempt)
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		log.Printf("⚠️ 試行 %d/%d 失敗: %v", attempt, MaxRetries, err)
		
		// 最後の試行でない場合は続行
		if attempt < MaxRetries {
			continue
		}
	}
	
	// すべてのリトライが失敗した場合でも、可能な限り処理を続行
	log.Printf("❌ すべてのリトライが失敗しました。最後のエラー: %v", lastErr)
	return nil, fmt.Errorf("最大リトライ回数（%d回）に達しました: %v", MaxRetries, lastErr)
}

// checkKLMSTaskOnce は1回のチェックを実行します
func checkKLMSTaskOnce(oldHash string, attempt int) (*CheckResult, error) {
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
		Timeout:   playwright.Float(60000), // 60秒タイムアウト
	}); err != nil {
		return nil, fmt.Errorf("ページ遷移エラー: %v", err)
	}

	// === ログイン処理 ===
	keioLink, _ := page.QuerySelector("a:has-text(\"keio.jp\")")
	if keioLink != nil {
		log.Println("🔗 keio.jpリンクをクリック")
		keioLink.Click(playwright.ElementHandleClickOptions{Force: playwright.Bool(true)})

		// タイムアウトを設定して待機
		if _, err := page.WaitForSelector("input[type=\"text\"]", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(30000), // 30秒
		}); err != nil {
			return nil, fmt.Errorf("ログインフォーム待機タイムアウト: %v", err)
		}
		
		page.Fill("input[type=\"text\"]", os.Getenv("KEIO_USER"))
		// 【最強のEnter連打】ここは絶対に変えません
		page.Press("input[type=\"text\"]", "Enter")

		if _, err := page.WaitForSelector("input[type=\"password\"]", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(30000), // 30秒
		}); err != nil {
			return nil, fmt.Errorf("パスワード入力欄待機タイムアウト: %v", err)
		}
		
		page.Fill("input[type=\"password\"]", os.Getenv("KEIO_PASS"))
		// 【最強のEnter連打】ここも絶対に変えません
		page.Press("input[type=\"password\"]", "Enter")
		
		// ログイン後の待機
		if err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
			State:   playwright.LoadStateDomcontentloaded,
			Timeout: playwright.Float(60000), // 60秒
		}); err != nil {
			return nil, fmt.Errorf("ログイン後のページ読み込みタイムアウト: %v", err)
		}
		context.StorageState(CookieFile) // 保存
	}

	// === ダッシュボード待機（複数のセレクタを試す） ===
	log.Println("⏳ ダッシュボード待機中...")
	
	// 複数のセレクタを順番に試す
	selectors := []string{
		"#planner-today-btn",  // 本日ボタン（最優先）
		"#dashboard",          // ダッシュボード要素
		"#dashboard-planner",  // プランナー表示
		".planner-day",        // プランナー日付要素
	}
	
	var dashboardFound bool
	var lastSelectorErr error
	
	for _, selector := range selectors {
		log.Printf("🔍 セレクタを試行中: %s", selector)
		if _, err := page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(DefaultTimeout), // 120秒
		}); err == nil {
			log.Printf("✅ セレクタが見つかりました: %s", selector)
			dashboardFound = true
			break
		} else {
			log.Printf("⚠️ セレクタが見つかりませんでした: %s (エラー: %v)", selector, err)
			lastSelectorErr = err
		}
	}
	
	if !dashboardFound {
		// タイムアウト時の詳細ログ
		currentURL := page.URL()
		log.Printf("⚠️ すべてのセレクタでタイムアウト。現在のURL: %s", currentURL)
		
		// ページのスクリーンショットを保存（デバッグ用）
		debugScreenshot := fmt.Sprintf("logs/timeout-debug-%d.png", time.Now().Unix())
		if _, err := page.Screenshot(playwright.PageScreenshotOptions{
			Path: playwright.String(debugScreenshot),
			FullPage: playwright.Bool(true),
		}); err == nil {
			log.Printf("📸 デバッグ用スクリーンショットを保存: %s", debugScreenshot)
		}
		
		// ページのHTMLを一部保存（デバッグ用）
		if html, err := page.Content(); err == nil {
			htmlDebugFile := fmt.Sprintf("logs/timeout-debug-%d.html", time.Now().Unix())
			ioutil.WriteFile(htmlDebugFile, []byte(html), 0644)
			log.Printf("📄 デバッグ用HTMLを保存: %s", htmlDebugFile)
		}
		
		return nil, fmt.Errorf("ダッシュボード到達タイムアウト（試行 %d回目）: %v", attempt, lastSelectorErr)
	}

	// ネットワークが落ち着くまで待機（タイムアウトを設定）
	log.Println("⏳ ネットワークアイドル待機中...")
	if err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State:   playwright.LoadStateNetworkidle,
		Timeout: playwright.Float(NetworkIdleTimeout), // 30秒
	}); err != nil {
		log.Printf("⚠️ ネットワークアイドル待機タイムアウト（続行します）: %v", err)
		// ネットワークアイドル待機のタイムアウトは致命的ではないので続行
	}

	// === ハッシュ化 ===
	targetSelector := "#dashboard"
	// カレンダー表示かリスト表示かを判定して対象を変えるロジック（そのまま維持）
	if listItems, _ := page.QuerySelector(".planner-day"); listItems != nil {
		targetSelector = "#dashboard-planner"
	}

	log.Printf("🎯 監視対象: %s", targetSelector)
	
	// セレクタが存在するか確認
	if _, err := page.QuerySelector(targetSelector); err != nil {
		log.Printf("⚠️ 監視対象セレクタが見つかりません: %s", targetSelector)
		// 代替セレクタを試す
		if listItems, _ := page.QuerySelector(".planner-day"); listItems != nil {
			targetSelector = "#dashboard-planner"
			log.Printf("🔄 代替セレクタを使用: %s", targetSelector)
		} else if dashboard, _ := page.QuerySelector("#dashboard"); dashboard != nil {
			targetSelector = "#dashboard"
			log.Printf("🔄 代替セレクタを使用: %s", targetSelector)
		} else {
			return nil, fmt.Errorf("監視対象セレクタが見つかりません: %s", targetSelector)
		}
	}
	
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