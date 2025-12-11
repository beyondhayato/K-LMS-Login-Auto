package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config はアプリケーションの設定を保持します
type Config struct {
	// ログイン情報
	KeioUser string
	KeioPass string

	// API設定
	GeminiAPIKey string
	MaxGeminiPerDay int

	// LINE通知設定
	LineToken  string
	LineUserID string

	// Gmail通知設定
	SMTPUser string
	SMTPPass string

	// その他
	CourseListFile string
}

// LoadConfig は環境変数から設定を読み込みます
func LoadConfig() (*Config, error) {
	cfg := &Config{
		KeioUser:       os.Getenv("KEIO_USER"),
		KeioPass:       os.Getenv("KEIO_PASS"),
		GeminiAPIKey:   os.Getenv("GEMINI_API_KEY"),
		LineToken:      os.Getenv("LINE_TOKEN"),
		LineUserID:     os.Getenv("LINE_USER_ID"),
		SMTPUser:       os.Getenv("SMTP_USER"),
		SMTPPass:       os.Getenv("SMTP_PASS"),
		CourseListFile: "data/courses.json",
		MaxGeminiPerDay: 20, // デフォルト値
	}

	// 環境変数からMaxGeminiPerDayを読み込む（オプション）
	if maxStr := os.Getenv("MAX_GEMINI_PER_DAY"); maxStr != "" {
		if max, err := strconv.Atoi(maxStr); err == nil && max > 0 {
			cfg.MaxGeminiPerDay = max
		}
	}

	// 必須項目のバリデーション
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate は設定の必須項目をチェックします
func (c *Config) Validate() error {
	if c.KeioUser == "" {
		return fmt.Errorf("KEIO_USERが設定されていません")
	}
	if c.KeioPass == "" {
		return fmt.Errorf("KEIO_PASSが設定されていません")
	}
	if c.GeminiAPIKey == "" {
		return fmt.Errorf("GEMINI_API_KEYが設定されていません")
	}
	// LINEとGmailはオプションなのでチェックしない
	
	return nil
}

