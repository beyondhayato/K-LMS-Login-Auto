package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

const historyFile = "data/sent_history.json"

type History struct {
	SentIDs []string `json:"sent_ids"`
}

// GenerateID は文字情報からIDを作ります（型への依存を排除）
func GenerateID(course, title, deadline string) string {
	data := fmt.Sprintf("%s|%s|%s", course, title, deadline)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func LoadHistory() (*History, error) {
	h := &History{SentIDs: []string{}}
	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		return h, nil
	}
	data, err := ioutil.ReadFile(historyFile)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, h); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *History) Save() error {
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	os.MkdirAll("data", 0755)
	return ioutil.WriteFile(historyFile, data, 0644)
}

// IsNew も文字列を受け取るように変更
func (h *History) IsNew(course, title, deadline string) bool {
	id := GenerateID(course, title, deadline)
	for _, sentID := range h.SentIDs {
		if sentID == id {
			return false
		}
	}
	return true
}

// Add も文字列を受け取るように変更
func (h *History) Add(course, title, deadline string) {
	if h.IsNew(course, title, deadline) {
		h.SentIDs = append(h.SentIDs, GenerateID(course, title, deadline))
	}
}