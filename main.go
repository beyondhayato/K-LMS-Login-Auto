package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath" // これを使うように修正しました
	"strings"
	"time"

	"github.com/joho/godotenv"

	"klms-go/internal/browser"
	"klms-go/internal/ics"
	"klms-go/internal/notify"
	"klms-go/internal/ocr"
	"klms-go/internal/storage"
)

// === 📂 ディレクトリとファイルパスの設定 ===
// ディレクトリは定数(const)でOK
const (
	LogDir  = "logs"
	DataDir = "data"
)

// ファイルパスは計算が必要なので変数(var)にします
var (
	LogFile     = filepath.Join(LogDir, "run-log.txt")
	LastRunFile = filepath.Join(DataDir, "last-run.txt")
	LastOcrFile = filepath.Join(DataDir, "last-ocr.txt")
	ScheduleFile = "schedule.ics" // これは添付用の一時ファイルなのでルートでOK
)

func main() {
	// === 0. フォルダ作成 (なければ作る) ===
	if err := os.MkdirAll(LogDir, 0755); err != nil {
		log.Fatalf("ログフォルダ作成エラー: %v", err)
	}
	if err := os.MkdirAll(DataDir, 0755); err != nil {
		log.Fatalf("データフォルダ作成エラー: %v", err)
	}

	// === 1. ログ設定 ===
	f, err := os.OpenFile(LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("ログファイルオープンエラー: %v", err)
	} else {
		defer f.Close()
		mw := io.MultiWriter(os.Stdout, f)
		log.SetOutput(mw)
	}

	log.Println("------------------------------------------------")
	log.Println("🚀 K-LMS監視を開始します: ", time.Now().Format("2006-01-02 15:04:05"))

	// === 2. 環境変数 ===
	if err := godotenv.Load(); err != nil {
		reportError("環境変数(.env)の読み込みに失敗しました")
		return
	}

	// === 3. 前回ハッシュ読み込み ===
	oldHash := ""
	if data, err := ioutil.ReadFile(LastRunFile); err == nil {
		oldHash = string(data)
	}

	// === 4. ブラウザ操作 ===
	result, err := browser.CheckKLMSTask(oldHash)
	if err != nil {
		reportError(fmt.Sprintf("ブラウザ操作エラー: %v", err))
		return
	}

	// === 5. 結果処理 ===
	if result.HasDiff {
		log.Println("📸 画像変化検知。OCRで詳細を確認します...")

		ocrText, assignments, err := ocr.ExtractAssignmentInfo(result.ScreenshotPath)
		if err != nil {
			log.Printf("⚠️ OCRエラー: %v", err)
			return
		}

		// 前回テキストの読み込み
		lastOcrText := ""
		if data, err := ioutil.ReadFile(LastOcrFile); err == nil {
			lastOcrText = string(data)
		}

		// テキスト内容の比較
		if normalizeText(ocrText) == normalizeText(lastOcrText) {
			log.Println("🧘 課題内容（テキスト）に変更はありませんでした。")
			ioutil.WriteFile(LastRunFile, []byte(result.Hash), 0644)
			return
		}

		// === ここから変更通知フロー ===
		log.Println("🔔 新しい課題を検出しました！")
		now := time.Now().Format("2006-01-02 15:04")

		// --- 重複防止フィルタリング ---
		history, _ := storage.LoadHistory()
		var newAssignments []ocr.Assignment

		for _, task := range assignments {
			if history.IsNew(task.Course, task.Title, task.Deadline) {
				newAssignments = append(newAssignments, task)
				history.Add(task.Course, task.Title, task.Deadline)
			}
		}

		// --- 添付ファイル準備 ---
		var attachments []string
		attachments = append(attachments, result.ScreenshotPath)

		// 新規課題がある場合のみICS作成
		if len(newAssignments) > 0 {
			log.Printf("📅 新規課題が %d 件あります。.icsを作成します...", len(newAssignments))
			
			icsContent := ics.GenerateICS(newAssignments)
			
			if err := ioutil.WriteFile(ScheduleFile, []byte(icsContent), 0644); err == nil {
				attachments = append(attachments, ScheduleFile)
			}
			history.Save()
		} else {
			log.Println("🧘 既出の課題ばかりなので、カレンダーファイルは作成しません。")
		}

		// === ① LINE通知 ===
		log.Println("💬 LINE送信中...")
		lineMsg := fmt.Sprintf("📚 K-LMS課題通知\n\n%s\n\n📅 %s\n(詳細はメールを確認してください)", ocrText, now)
		if err := notify.SendLINE(lineMsg); err != nil {
			log.Printf("⚠️ LINE送信エラー: %v", err)
		}

		// === ② Gmail通知 ===
		log.Println("📧 Gmail送信中...")
		mailBody := fmt.Sprintf("課題を検出しました。\n\n%s\n\n📅 検知時刻: %s", ocrText, now)
		
		if len(newAssignments) > 0 {
			mailBody += "\n\n✨ 新しい課題が含まれていたため、カレンダー登録用ファイルを添付しました。"
		} else {
			mailBody += "\n\n(※新しい課題はないため、カレンダーファイルは添付していません)"
		}

		if err := notify.SendGmail("【K-LMS】課題通知", mailBody, attachments); err != nil {
			log.Printf("⚠️ Gmail送信エラー: %v", err)
		} else {
			log.Println("✅ Gmail送信完了")
		}

		// 完了処理
		ioutil.WriteFile(LastRunFile, []byte(result.Hash), 0644)
		ioutil.WriteFile(LastOcrFile, []byte(ocrText), 0644)
		log.Println("🎉 全工程完了")

	} else {
		log.Println("✅ 変化なし")
	}
}

func normalizeText(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "　", "")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

func reportError(errMsg string) {
	log.Printf("❌ 致命的なエラー: %s", errMsg)
	notify.SendGmail("【K-LMSエラー】監視システム停止", errMsg, nil)
}