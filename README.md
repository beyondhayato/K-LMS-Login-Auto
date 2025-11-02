# K-LMS Automation (Puppeteer × Google Drive × LINE × Gemini)

**概要**  
このプロジェクトは、Puppeteerを使って慶應義塾大学K-LMSに自動ログイン → ダッシュボードをスクリーンショット → Gemini APIで文字起こし → Google Driveにアップロード → GASでDriveフォルダを監視し、新規ファイルを検知したらGmailとLINEへ通知を行う自動化システムです。

---

## 🚀 主な機能
1. **自動ログイン（Puppeteer）**  
   Keio SSOを自動操作してK-LMSにログイン
2. **スクリーンショット取得**  
   ダッシュボードを撮影し `klms_after_login.png` として保存
3. **Geminiによる文字起こし**  
   画像内の課題名・期限などを抽出
4. **Google Driveへのアップロード（OAuth2）**
   Drive APIを通じてクラウドにアップロード
5. **GAS連携通知**
   DriveトリガーでGmail（添付＋文字起こし）・LINE（文字のみ）へ通知

---

## 🛠 ファイル構成
| ファイル名 | 内容 |
|-------------|------|
| `token.js` | メインスクリプト（Puppeteer＋Gemini＋Drive） |
| `gemini-ocr.js` | Gemini APIでスクショを文字起こし |
| `cron-scheduler.js` | 定期実行用スケジューラ |
| `package.json` | npm設定・依存ライブラリ |
| `.env.example` | 環境変数テンプレート |
| `.gitignore` | 機密ファイル除外設定 |
| `README.md` | このドキュメント |

---

## ⚙️ 環境変数の設定
1. `.env.example` をコピーして `.env` にリネーム  
2. 以下の値を入力  
   ```bash
   KEIO_USER=your_keio_email@keio.jp
   KEIO_PASS=your_keio_password
   GOOGLE_EMAIL=your_google_email
   GOOGLE_PASSWORD=your_google_password
   GOOGLE_DRIVE_FOLDER_ID=your_drive_folder_id
   GEMINI_API_KEY=your_gemini_api_key
.env は 絶対にGitHubにアップロードしない こと。

⚡ セットアップ手順
依存ライブラリをインストール

bash
コードをコピーする
npm install
OAuth認証の設定

Google Cloud ConsoleでDrive APIを有効化

oauth-credentials.json を取得し、ルートに配置

初回実行時に認証URLが出る → 自分のGoogleアカウントで認可

自動ログイン＆アップロードを実行

bash
コードをコピーする
npm run token
成功すると klms_after_login.png が生成され、Driveにアップされます。

🧠 仕組みの概要
text
コードをコピーする
Puppeteer → K-LMS自動ログイン
   ↓
Gemini API → 画像をOCR解析
   ↓
Google Drive API → スクリーンショット＋テキストをアップロード
   ↓
GASトリガー → Gmail/LINEに自動通知
🔒 セキュリティ
.env, token.json, oauth-credentials.json は機密情報を含むため絶対にGitHubへpushしない

.gitignore に以下を追加済み：

pgsql
コードをコピーする
.env
token.json
oauth-credentials.json
credentials.json
node_modules/
LINEトークンはGASのスクリプトプロパティに保存して利用します。

🧩 よくあるエラー
エラー	原因・対処
MODULE_NOT_FOUND	npm install が未実行。依存関係を入れる
waitForSelector timeout	SSOリダイレクトが遅延。waitUntil: 'networkidle2' を追加
model not found (Gemini)	APIバージョンに対応したモデル名を使う
oAuth2Client is not defined	認証処理のスコープ外。uploadToGoogleDriveOAuth() 前に oAuth2Client を明示的に定義

💬 補足
GAS側では checkNewFiles() をトリガーにして Drive フォルダを定期監視

新規ファイルがあれば文字起こし結果と画像をメール＆LINEで通知

Node側はローカル or サーバー上のcronスケジューラで定期実行可能

👨‍💻 作者
開発者: 慶應義塾大学商学部2年 宮久保隼 
(Contact：haya.miy02@keio.jp) © 2025