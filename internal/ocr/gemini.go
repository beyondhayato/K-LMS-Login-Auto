package ocr

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// 1日の制限回数設定
const (
	DailyLimitFile = "daily-gemini-count.json"
	MaxGeminiPerDay = 50
)

// 日次カウントデータの構造
type DailyData struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// ExtractAssignmentInfo は画像をGeminiに投げて文字起こしします
func ExtractAssignmentInfo(imagePath string) (string, error) {
	// 1. まず1日の制限チェック
	if !canRunGeminiToday() {
		log.Println("🚫 今日のGemini実行上限（50回）に達したため、OCRをスキップします")
		return "実行制限到達のためOCRスキップ", nil
	}

	ctx := context.Background()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEYが設定されていません")
	}

	// 2. Geminiクライアント作成
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", fmt.Errorf("Geminiクライアント作成エラー: %v", err)
	}
	defer client.Close()

	// 3. 画像読み込み
	imgData, err := ioutil.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("画像読み込みエラー: %v", err)
	}

	// 4. モデル設定 (Node.js版と同じモデル名)
	model := client.GenerativeModel("gemini-2.5-flash") // 

	// 5. プロンプト作成
	prompt := genai.Text(`
この画像は慶應義塾大学のK-LMS（Canvas）ダッシュボードのスクリーンショットです。

以下の情報を抽出してください：
1. 授業名
2. 課題タイトル
3. 提出期限（日付・時間）

複数課題がある場合はすべて出力してください。
情報が見つからない場合は「なし」と記載してください。

出力形式（厳守）：
【授業名】
【課題】
【期限】

余計な説明文は書かず、上記フォーマットのみ出力してください。
`)

	// 6. 送信 (画像 + テキスト)
	resp, err := model.GenerateContent(ctx, prompt, genai.ImageData("png", imgData))
	if err != nil {
		return "", fmt.Errorf("Gemini生成エラー: %v", err)
	}

	// 7. 結果取得
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "読み取り結果なし", nil
	}

	// テキスト部分を取り出す
	var resultText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			resultText += string(txt)
		}
	}

	// 8. 成功したのでカウントアップ
	incrementGeminiCount()
	
	return resultText, nil
}

// === 以下、回数制限管理ロジック ===

func loadDailyCount() DailyData {
	data := DailyData{Date: time.Now().Format("2006-01-02"), Count: 0}
	file, err := ioutil.ReadFile(DailyLimitFile)
	if err == nil {
		json.Unmarshal(file, &data)
	}
	return data
}

func saveDailyCount(data DailyData) {
	file, _ := json.MarshalIndent(data, "", "  ")
	ioutil.WriteFile(DailyLimitFile, file, 0644)
}

func canRunGeminiToday() bool {
	data := loadDailyCount()
	today := time.Now().Format("2006-01-02")

	// 日付が変わっていたらリセット
	if data.Date != today {
		saveDailyCount(DailyData{Date: today, Count: 0})
		return true
	}
	return data.Count < MaxGeminiPerDay
}

func incrementGeminiCount() {
	data := loadDailyCount()
	today := time.Now().Format("2006-01-02")

	if data.Date != today {
		data = DailyData{Date: today, Count: 1}
	} else {
		data.Count++
	}
	saveDailyCount(data)
}