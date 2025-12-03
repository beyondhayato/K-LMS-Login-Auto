package ocr

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// 設定ファイル
const (
	DailyLimitFile = "data/daily-gemini-count.json"
	CourseListFile = "data/courses.json" // ★追加: 科目リストのパス
	MaxGeminiPerDay = 50
)

type DailyData struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type Assignment struct {
	Course   string `json:"course"`
	Title    string `json:"title"`
	Deadline string `json:"deadline"`
}

func ExtractAssignmentInfo(imagePath string) (string, []Assignment, error) {
	if !canRunGeminiToday() {
		return "実行制限到達のためOCRスキップ", nil, nil
	}

	ctx := context.Background()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", nil, fmt.Errorf("GEMINI_API_KEYが設定されていません")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", nil, fmt.Errorf("Geminiクライアント作成エラー: %v", err)
	}
	defer client.Close()

	imgData, err := ioutil.ReadFile(imagePath)
	if err != nil {
		return "", nil, fmt.Errorf("画像読み込みエラー: %v", err)
	}

	// ★追加: 科目リストを読み込む
	courseListJSON := "[]"
	if data, err := ioutil.ReadFile(CourseListFile); err == nil {
		courseListJSON = string(data)
	}

	model := client.GenerativeModel("gemini-2.5-flash")

	// ★修正: プロンプトに科目リスト(Known Courses)を含める
	currentYear := time.Now().Year()
	prompt := genai.Text(fmt.Sprintf(`
この画像はK-LMSのダッシュボードです。
以下の「登録済み科目リスト」を参照し、検出された授業名がリスト内のものと一致、あるいは類似している場合は、**必ずリスト内の正式名称（教員名含む）**に修正して出力してください。

【登録済み科目リスト】
%s

抽出ルール:
1. course: 授業名。可能な限り上記のリストにある名称を使用すること。リストにない場合は画像内の表記に従うが、教員名がわかる場合は "授業名 (教員名)" の形式にすること。
2. title: 課題名
3. deadline: 期限 (現在は%d年です。"YYYY-MM-DD HH:mm" 形式に変換すること)

出力は**JSON配列形式のみ**で行ってください。

出力例:
[
  {"course": "造形・デザイン論 (荒木 文果)", "title": "小テスト (7)", "deadline": "2025-12-07 23:59"},
  {"course": "統計学基礎 (藪 友良)", "title": "課題1", "deadline": "2026-01-13 23:59"}
]
`, courseListJSON, currentYear))

	resp, err := model.GenerateContent(ctx, prompt, genai.ImageData("png", imgData))
	if err != nil {
		return "", nil, fmt.Errorf("Gemini生成エラー: %v", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "読み取り結果なし", nil, nil
	}

	var rawJSON string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			rawJSON += string(txt)
		}
	}

	rawJSON = strings.TrimSpace(rawJSON)
	rawJSON = strings.TrimPrefix(rawJSON, "```json")
	rawJSON = strings.TrimPrefix(rawJSON, "```")
	rawJSON = strings.TrimSuffix(rawJSON, "```")

	incrementGeminiCount()

	var assignments []Assignment
	if err := json.Unmarshal([]byte(rawJSON), &assignments); err != nil {
		log.Printf("JSONパース失敗: %v \n生データ: %s", err, rawJSON)
		return rawJSON, nil, nil
	}

	var notifyText string
	if len(assignments) == 0 {
		notifyText = "課題は見つかりませんでした"
	} else {
		for _, a := range assignments {
			dateStr := formatDeadline(a.Deadline)
			// 通知フォーマット
			notifyText += fmt.Sprintf("【コース詳細】%s\n【課題】%s\n【期限】%s\n---\n", a.Course, a.Title, dateStr)
		}
	}
	
	return notifyText, assignments, nil
}

func formatDeadline(isoDate string) string {
	t, err := time.Parse("2006-01-02 15:04", isoDate)
	if err != nil {
		return isoDate
	}
	now := time.Now()
	if t.Year() == now.Year() {
		return t.Format("1月2日 15:04")
	}
	return t.Format("2006年1月2日 15:04")
}

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
	os.MkdirAll("data", 0755)
	ioutil.WriteFile(DailyLimitFile, file, 0644)
}

func canRunGeminiToday() bool {
	data := loadDailyCount()
	today := time.Now().Format("2006-01-02")

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