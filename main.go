package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"klms-go/internal/browser"
	"klms-go/internal/notify"
	"klms-go/internal/ocr"
)

// ログファイルのパス
const logFile = "run-log.txt"

func main() {
	// === 1. ログ設定 (画面が出ないのでファイルに書き込む) ===
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		// ログファイルすら開けない場合はどうしようもないので終了
		return
	}
	defer f.Close()
	// ログの出力先を「ファイル」と「画面（あれば）」の両方にする
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)

	// 実行開始ログ
	log.Println("------------------------------------------------")
	log.Println("🚀 K-LMS監視を開始します: ", time.Now().Format("2006-01-02 15:04:05"))

	// === 2. 環境変数読み込み ===
	if err := godotenv.Load(); err != nil {
		reportError("環境変数(.env)の読み込みに失敗しました")
		return
	}

	// === 3. 前回ハッシュ読み込み ===
	oldHash := ""
	if data, err := ioutil.ReadFile("last-run.txt"); err == nil {
		oldHash = string(data)
	}

	// === 4. ブラウザ操作 (ここでエラーが起きやすい) ===
	result, err := browser.CheckKLMSTask(oldHash)
	if err != nil {
		reportError(fmt.Sprintf("ブラウザ操作エラー: %v", err))
		return
	}

	// === 5. 結果処理 ===
	if result.HasDiff {
		log.Println("📸 スクリーンショットを保存しました")

		// OCR実行
		log.Println("🔍 Geminiで課題情報を抽出中...")
		ocrText, err := ocr.ExtractAssignmentInfo(result.ScreenshotPath)
		if err != nil {
			// OCRのエラーは致命的ではないので通知せずログだけ残す（あるいは通知してもOK）
			log.Printf("⚠️ OCRエラー: %v", err)
			ocrText = "OCR読み取り失敗（画像を確認してください）"
		}

		// 現在時刻
		now := time.Now().Format("2006-01-02 15:04")

		// === ① LINE通知 ===
		log.Println("💬 LINE送信中...")
		lineMsg := fmt.Sprintf("📚 K-LMS課題通知\n\n%s\n\n📅 %s", ocrText, now)
		if err := notify.SendLINE(lineMsg); err != nil {
			log.Printf("⚠️ LINE送信エラー: %v", err)
			// LINEがダメでもメールは送りたいので続行
		} else {
			log.Println("✅ LINE送信完了")
		}

		// === ② Gmail通知 (画像添付) ===
		log.Println("📧 Gmail送信中...")
		mailBody := fmt.Sprintf("以下の課題を検出しました。\n\n%s\n\n📅 検知時刻: %s", ocrText, now)
		if err := notify.SendGmail("【K-LMS】新しい課題通知", mailBody, result.ScreenshotPath); err != nil {
			log.Printf("⚠️ Gmail送信エラー: %v", err)
		} else {
			log.Println("✅ Gmail送信完了")
		}

		// 完了処理
		ioutil.WriteFile("last-run.txt", []byte(result.Hash), 0644)
		log.Println("🎉 全工程完了")

	} else {
		log.Println("✅ 変化なし")
	}
}

// reportError はエラーをログに書き込み、Gmailで通知します
func reportError(errMsg string) {
	// 1. ログに書く
	log.Printf("❌ 致命的なエラー: %s", errMsg)

	// 2. メールを送る
	// ※画像パスは空文字""にして画像なしで送る
	err := notify.SendGmail(
		"【K-LMSエラー】監視システムが停止しました",
		fmt.Sprintf("K-LMS監視システムで以下のエラーが発生しました。\n\n内容:\n%s\n\nログファイル(run-log.txt)を確認してください。", errMsg),
		"", 
	)

	if err != nil {
		log.Printf("⚠️ エラー通知メールの送信にも失敗しました: %v", err)
	} else {
		log.Println("📧 エラー通知メールを送信しました")
	}
}