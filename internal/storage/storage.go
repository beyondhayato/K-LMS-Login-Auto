package storage

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"
)

// ファイルの保存場所
const (
	DataDir   = "data"
	UsageFile = "data/daily_usage.json"
)

// まとめたデータ構造
type DailyUsage struct {
	Date               string `json:"date"`
	GeminiCount        int    `json:"gemini_count"`
	LineCount          int    `json:"line_count"`
	GmailCount         int    `json:"gmail_count"`
	GmailLimitNotified bool   `json:"gmail_limit_notified"`
}

// データを読み込む（なければ作る）
func LoadUsage() DailyUsage {
	// 今日の日付
	today := time.Now().Format("2006-01-02")
	
	defaultData := DailyUsage{
		Date: today, GeminiCount: 0, LineCount: 0, GmailCount: 0, GmailLimitNotified: false,
	}

	// フォルダがない場合は作成（念のため）
	if _, err := os.Stat(DataDir); os.IsNotExist(err) {
		os.Mkdir(DataDir, 0755)
	}

	file, err := ioutil.ReadFile(UsageFile)
	if err != nil {
		return defaultData
	}

	var data DailyUsage
	json.Unmarshal(file, &data)

	// 日付が変わっていたらリセット
	if data.Date != today {
		data = defaultData
		SaveUsage(data)
	}

	return data
}

// データを保存する
func SaveUsage(data DailyUsage) {
	file, _ := json.MarshalIndent(data, "", "  ")
	ioutil.WriteFile(UsageFile, file, 0644)
}

// --- 以下、各機能向けの便利関数 ---

// Geminiのカウントを増やす
func IncrementGemini() {
	data := LoadUsage()
	data.GeminiCount++
	SaveUsage(data)
}

// LINEのカウントを増やす
func IncrementLine() {
	data := LoadUsage()
	data.LineCount++
	SaveUsage(data)
}

// Gmailのカウントを増やす
func IncrementGmail() {
	data := LoadUsage()
	data.GmailCount++
	SaveUsage(data)
}

// Gmailの終了通知フラグを立てる
func MarkGmailLimitNotified() {
	data := LoadUsage()
	data.GmailLimitNotified = true
	SaveUsage(data)
}