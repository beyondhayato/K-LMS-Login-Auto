🚀 K-LMS Auto Checker (Standalone)
Go言語 × Playwright × Gemini × LINE/Gmail 慶應義塾大学の K-LMS（Canvas）をバックグラウンドで高速監視し、課題が追加された瞬間に LINEとメールで通知 する自動化ツールです。

✨ 特徴
完全自立型: GAS（Google Apps Script）や中間サーバーは不要。PC単体で完結します。

爆速: 起動からチェック終了までわずか数秒。

ステルス: 作業の邪魔をしない完全バックグラウンド実行。

W通知: LINEで「速報」を、Gmailで「詳細（スクショ付き）」を受け取れます。

📦 配布内容
フォルダ内には以下のファイルのみが含まれています。

K-LMS-Mac (または K-LMS.exe) ... アプリ本体

.env ... 設定ファイル

README.md ... この説明書

⚙️ 設定ファイルの準備 (.env)
同梱されている .env ファイルをテキストエディタで開き、以下の情報を入力して保存してください。

Ini, TOML

# --- 1. 大学ログイン情報 ---
KEIO_USER=your_id@keio.jp
KEIO_PASS=your_password

# --- 2. AI設定 (OCR用) ---
# Google AI Studioで無料取得: https://aistudio.google.com/app/apikey
GEMINI_API_KEY=取得したキーを貼り付け

# --- 3. LINE通知設定 (Messaging API) ---
# LINE Developersで取得したトークンとUser ID
LINE_TOKEN=
LINE_USER_ID=

# --- 4. Gmail通知設定 (画像送付用) ---
# 自分のGmailアドレス
SMTP_USER=your_email@gmail.com
# Gmailの「アプリパスワード」(16桁) ※普段のログインパスワードではありません
SMTP_PASS=xxxx xxxx xxxx xxxx
💡 Gmailアプリパスワードの取得方法

Googleアカウント管理 → セキュリティ → 2段階認証をオン

セキュリティ画面の検索窓で「アプリパスワード」と検索

適当な名前（K-LMSなど）をつけて作成 → 生成された16桁を使用

🍎 Macでのセットアップ手順
1. 準備
ターミナルを開き、解凍したフォルダへ移動します。 （cd と入力した後に、フォルダをターミナルにドラッグ＆ドロップしてEnter）

2. 権限の付与（必須）
Macのセキュリティにより、最初は実行できません。以下のコマンドを実行してください。

Bash

chmod +x K-LMS-Mac
3. 初回起動と許可
Finderで K-LMS-Mac を 右クリック（Control+クリック） して「開く」を選択します。

「開発元が未確認ですが開きますか？」と出たら 「開く」 を押します。

初回のみ、ブラウザのダウンロード等が自動で行われます。

ターミナルが閉じて、LINE/Gmailに通知が来れば成功です！

4. 自動実行の設定 (launchd)
1分おきに自動で動かす設定です。

以下の内容で ~/Library/LaunchAgents/com.klms.autocheck.plist というファイルを作成します。 ※ Path/To/... の部分は、実際のフォルダの場所（絶対パス）に書き換えてください。

XML

<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.klms.autocheck</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/あなたのユーザー名/Desktop/K-LMS-Distribution/K-LMS-Mac</string>
    </array>
    <key>StartInterval</key>
    <integer>60</integer>
    <key>WorkingDirectory</key>
    <string>/Users/あなたのユーザー名/Desktop/K-LMS-Distribution</string>
    
    <key>StandardOutPath</key>
    <string>/tmp/klms.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/klms_err.log</string>
</dict>
</plist>
設定を反映して開始します。

Bash

launchctl load ~/Library/LaunchAgents/com.klms.autocheck.plist
launchctl start com.klms.autocheck
🪟 Windowsでのセットアップ手順
.env を設定します。

K-LMS.exe をダブルクリックして動作確認します（初回は数秒かかります）。

タスクスケジューラ に登録します。

トリガー: 毎日 (詳細設定で「1分間隔」「継続時間: 9時間」など)

操作: プログラムの開始 (K-LMS.exeを選択)

開始オプション (重要): K-LMS.exe があるフォルダのパスを入力

条件: 「タスクを実行するためにスリープを解除する」にチェック

⚠️ 注意事項
.envの管理: パスワードが含まれるため、このファイルは絶対に他人に渡さないでください。

API制限: Gemini APIなどの無料枠を超えないよう、内部で回数制限がかかっています。