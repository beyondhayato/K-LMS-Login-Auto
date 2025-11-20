package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"gopkg.in/gomail.v2"
)

// SendLINE はテキストメッセージをLINEに送ります
func SendLINE(message string) error {
	token := os.Getenv("LINE_TOKEN")
	userID := os.Getenv("LINE_USER_ID")

	if token == "" || userID == "" {
		return fmt.Errorf("LINE設定が足りません")
	}

	// LINE Messaging API の形式
	payload := map[string]interface{}{
		"to": userID,
		"messages": []map[string]string{
			{
				"type": "text",
				"text": message,
			},
		},
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.line.me/v2/bot/message/push", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("LINE送信失敗: %s", resp.Status)
	}
	return nil
}

// SendGmail は画像を添付してGmailを送ります
func SendGmail(subject string, body string, imagePath string) error {
	smtpUser := os.Getenv("SMTP_USER") // あなたのGmailアドレス
	smtpPass := os.Getenv("SMTP_PASS") // Gmailアプリパスワード

	if smtpUser == "" || smtpPass == "" {
		return fmt.Errorf("Gmail設定が足りません")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", smtpUser)
	m.SetHeader("To", smtpUser) // 自分宛てに送る
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	// 画像があれば添付する
	if imagePath != "" {
		m.Attach(imagePath)
	}

	// Gmailのサーバー設定
	d := gomail.NewDialer("smtp.gmail.com", 587, smtpUser, smtpPass)

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}