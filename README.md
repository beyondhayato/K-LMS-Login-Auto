🚀 K-LMS Auto Checker (Standalone)
Go言語 × Playwright × Gemini × LINE/Gmail 慶應義塾大学の K-LMS（Canvas）をバックグラウンドで高速監視し、課題が追加された瞬間に LINEとメールで通知 する自動化ツールです。

✨ 特徴
完全自立型: GAS（Google Apps Script）や中間サーバーは不要。PC単体で完結します。

爆速: 起動からチェック終了までわずか数秒。

ステルス: 作業の邪魔をしない完全バックグラウンド実行。

W通知: LINEで「速報」を、Gmailで「詳細（スクショ付き）」を受け取れます。

📅 課題を Googleカレンダーに自動取り込み

K-LMS 上の「課題の締切情報」をもとに、プログラム実行時に `schedule.ics` というカレンダーファイルを自動生成します。

- 課題のタイトル・締切日時などを内部で JSON 形式にまとめ、それを元に iCalendar 形式 (`.ics`) のファイルを生成
- 生成された `schedule.ics` は Gmail 通知メールに添付されます
- スマホでメールを開き、`schedule.ics` をダウンロードして開くと、そのまま Googleカレンダーに予定として追加できます

※ `schedule.ics` は毎回プログラムから自動生成されるファイルであり、リポジトリには含めていません（`.gitignore` 対象）。

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

責任者：慶應義塾大学商学部2年 宮久保隼(haya.miy02@keio.jp)