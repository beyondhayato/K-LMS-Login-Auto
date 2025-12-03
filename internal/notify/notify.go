package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"gopkg.in/gomail.v2"
	
	"klms-go/internal/storage"
)

const (
	MaxLinePerDay  = 10
	MaxGmailPerDay = 50
)

// SendLINE はテキストメッセージをLINEに送ります
func SendLINE(message string) error {
	usage := storage.LoadUsage()
	if usage.LineCount >= MaxLinePerDay {
		return fmt.Errorf("本日のLINE送信上限(%d回)に達したためスキップします", MaxLinePerDay)
	}

	token := os.Getenv("LINE_TOKEN")
	userID := os.Getenv("LINE_USER_ID")

	if token == "" || userID == "" {
		return fmt.Errorf("LINE設定が足りません")
	}

	payload := map[string]interface{}{
		"to": userID,
		"messages": []map[string]string{
			{"type": "text", "text": message},
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

	storage.IncrementLine()
	return nil
}

// SendGmail は画像と.icsファイルを添付してGmailを送ります
func SendGmail(subject string, body string, attachments []string) error {
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")

	if smtpUser == "" || smtpPass == "" {
		return fmt.Errorf("Gmail設定が足りません")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", smtpUser)
	m.SetHeader("To", smtpUser)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	// リストにあるファイルを全て添付
	for _, filePath := range attachments {
		if filePath != "" {
			m.Attach(filePath)
		}
	}

	d := gomail.NewDialer("smtp.gmail.com", 587, smtpUser, smtpPass)

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}