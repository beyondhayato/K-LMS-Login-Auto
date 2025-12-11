package ocr

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// 設定ファイルパス
const (
	DailyLimitFile    = "data/daily-gemini-count.json" // Gemini APIの1日あたりの使用回数を記録
	CourseListFile    = "data/courses.json"            // 登録済み科目リストのパス
	OcrCacheFile      = "data/ocr-cache.json"          // 画像ハッシュとOCR結果のキャッシュ
	MaxGeminiPerDay   = 20                              // Gemini APIの1日あたりの使用制限（環境変数MAX_GEMINI_PER_DAYで上書き可能）
)

// DailyData は1日あたりのGemini API使用回数を記録します
type DailyData struct {
	Date  string `json:"date"`  // 日付（YYYY-MM-DD形式）
	Count int    `json:"count"` // 使用回数
}

// Assignment は課題情報を保持します
type Assignment struct {
	Course   string `json:"course"`   // 授業名（教員名含む）
	Title    string `json:"title"`    // 課題名
	Deadline string `json:"deadline"` // 期限（YYYY-MM-DD HH:mm形式）
}

// OCRキャッシュ用の構造体
type OcrCacheEntry struct {
	ImageHash   string       `json:"image_hash"`
	OcrText     string       `json:"ocr_text"`
	Assignments []Assignment `json:"assignments"`
	Timestamp   string       `json:"timestamp"`
}

type OcrCache struct {
	Entries []OcrCacheEntry `json:"entries"`
}

// 画像のハッシュを計算
func calculateImageHash(imagePath string) (string, error) {
	imgData, err := ioutil.ReadFile(imagePath)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(imgData)
	return hex.EncodeToString(hash[:]), nil
}

// OCRキャッシュを読み込む
func loadOcrCache() *OcrCache {
	cache := &OcrCache{Entries: []OcrCacheEntry{}}
	if data, err := ioutil.ReadFile(OcrCacheFile); err == nil {
		json.Unmarshal(data, cache)
	}
	return cache
}

// OCRキャッシュを保存
func saveOcrCache(cache *OcrCache) {
	data, _ := json.MarshalIndent(cache, "", "  ")
	os.MkdirAll("data", 0755)
	ioutil.WriteFile(OcrCacheFile, data, 0644)
}

// キャッシュからOCR結果を取得
func getCachedOcrResult(imageHash string) (string, []Assignment, bool) {
	cache := loadOcrCache()
	for _, entry := range cache.Entries {
		if entry.ImageHash == imageHash {
			log.Printf("💾 キャッシュからOCR結果を取得しました（API使用回数節約）")
			return entry.OcrText, entry.Assignments, true
		}
	}
	return "", nil, false
}

// OCR結果をキャッシュに保存
func saveOcrResult(imageHash string, ocrText string, assignments []Assignment) {
	cache := loadOcrCache()
	
	// 既存のエントリを削除（同じハッシュがある場合）
	newEntries := []OcrCacheEntry{}
	for _, entry := range cache.Entries {
		if entry.ImageHash != imageHash {
			newEntries = append(newEntries, entry)
		}
	}
	
	// 新しいエントリを追加
	newEntries = append(newEntries, OcrCacheEntry{
		ImageHash:   imageHash,
		OcrText:     ocrText,
		Assignments: assignments,
		Timestamp:   time.Now().Format("2006-01-02 15:04:05"),
	})
	
	// キャッシュサイズを制限（最新50件まで保持）
	if len(newEntries) > 50 {
		newEntries = newEntries[len(newEntries)-50:]
	}
	
	cache.Entries = newEntries
	saveOcrCache(cache)
}

func ExtractAssignmentInfo(imagePath string) (string, []Assignment, error) {
	// 画像ハッシュを計算
	imageHash, err := calculateImageHash(imagePath)
	if err != nil {
		return "", nil, fmt.Errorf("画像ハッシュ計算エラー: %v", err)
	}
	
	// キャッシュをチェック
	if ocrText, assignments, found := getCachedOcrResult(imageHash); found {
		return ocrText, assignments, nil
	}
	
	// キャッシュにない場合、Gemini APIを使用
	if !canRunGeminiToday() {
		log.Printf("⚠️ Gemini APIの1日あたりの使用制限（%d回）に達しました。OCRをスキップします。", MaxGeminiPerDay)
		return "実行制限到達のためOCRスキップ", nil, nil
	}
	
	log.Printf("🔍 新しい画像を検出しました。Gemini APIでOCRを実行します...")

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
	log.Printf("✅ Gemini APIでOCR完了")

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
	
	// OCR結果をキャッシュに保存
	saveOcrResult(imageHash, notifyText, assignments)
	
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
		log.Printf("📊 Gemini API使用回数: 0/%d (本日リセット)", MaxGeminiPerDay)
		return true
	}
	remaining := MaxGeminiPerDay - data.Count
	log.Printf("📊 Gemini API使用回数: %d/%d (残り: %d回)", data.Count, MaxGeminiPerDay, remaining)
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