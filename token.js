	require('dotenv').config();
const puppeteer = require('puppeteer');
const fs = require('fs');
const { google } = require('googleapis');
const path = require('path');
const readline = require('readline');
const { extractAssignmentInfo } = require("./gemini-ocr");

// === OAuth認証でGoogleドライブにアップロード ===
// === Googleドライブにアップロード（OAuth認証付き）===
async function uploadToGoogleDriveOAuth(screenshotPath) {
  try {
    console.log('📤 Googleドライブにアップロード中...');
    
    const credentials = JSON.parse(fs.readFileSync('oauth-credentials.json'));
    const { client_secret, client_id, redirect_uris } = credentials.installed || credentials.web;
    const oAuth2Client = new google.auth.OAuth2(client_id, client_secret, redirect_uris[0]);
    
    const TOKEN_PATH = 'token.json';
    if (fs.existsSync(TOKEN_PATH)) {
      const token = fs.readFileSync(TOKEN_PATH);
      oAuth2Client.setCredentials(JSON.parse(token));
    } else {
      await getAccessToken(oAuth2Client, TOKEN_PATH);
    }
    
    const drive = google.drive({ version: 'v3', auth: oAuth2Client });
    
    const fileMetadata = {
      name: path.basename(screenshotPath),
      parents: [process.env.GOOGLE_DRIVE_FOLDER_ID]
    };
    
    const media = {
      mimeType: 'image/png',
      body: fs.createReadStream(screenshotPath)
    };
    
    const response = await drive.files.create({
      requestBody: fileMetadata,
      media: media,
      fields: 'id, name, webViewLink'
    });
    
    console.log('✅ アップロード完了:', response.data.name);
    console.log('🔗 リンク:', response.data.webViewLink);

    // 🔹 ここでアップロード結果と認証情報を両方返す
    return { file: response.data, auth: oAuth2Client };

  } catch (error) {
    console.error('❌ アップロードエラー:', error.message);
    throw error;
  }
}


async function getAccessToken(oAuth2Client, tokenPath) {
  const authUrl = oAuth2Client.generateAuthUrl({
    access_type: 'offline',
    scope: ['https://www.googleapis.com/auth/drive.file']
  });
  
  console.log('🔐 以下のURLにアクセスして認証してください:');
  console.log(authUrl);
  
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout
  });
  
  return new Promise((resolve) => {
    rl.question('認証コードを入力してください: ', async (code) => {
      rl.close();
      const { tokens } = await oAuth2Client.getToken(code);
      oAuth2Client.setCredentials(tokens);
      fs.writeFileSync(tokenPath, JSON.stringify(tokens));
      console.log('✅ トークンを保存しました');
      resolve();
    });
  });
}

(async () => {
  const browser = await puppeteer.launch({
    headless: true,
    defaultViewport: null,
    args: ['--start-maximized']
  });
  const page = await browser.newPage();

  try {
    const loginUrl = 'https://lms.keio.jp/';
    console.log(`🌐 アクセス中: ${loginUrl}`);
    await page.goto(loginUrl, { waitUntil: 'networkidle2', timeout: 60000 });

    // === Step 0: keio.jpボタンのタップ ===
    try {
      await page.waitForFunction(
        () => {
          const links = Array.from(document.querySelectorAll('a'));
          return links.some(link => link.textContent.includes('keio.jp'));
        },
        { timeout: 10000 }
      );
      await page.evaluate(() => {
        const links = Array.from(document.querySelectorAll('a'));
        const targetLink = links.find(link => link.textContent.includes('keio.jp'));
        if (targetLink) {
          targetLink.click();
        }
      });
      console.log('keio.jpリンクをクリックしました');
      await page.waitForNavigation({ waitUntil: 'networkidle2' });  
    } catch (error) {
      console.error('エラー:', error.message);
    }

    // === Step 1: ユーザー名入力・次へボタンのタップ ===
    await page.waitForSelector('input[type="text"]', { timeout: 60000 });
    console.log('🧾 Keioログイン（ユーザー名入力）画面を検出');
    await page.type('input[type="text"]', process.env.KEIO_USER, { delay: 60 });
    await page.click('input.button.button-primary');

    // === Step 2: パスワード入力・確認ボタンのタップ ===
    await page.waitForSelector('input[type="password"]', { timeout: 60000 });
    console.log('🔑 パスワード入力画面を検出');
    await page.type('input[type="password"]', process.env.KEIO_PASS, { delay: 60 });
    await page.click('input.button.button-primary');

    // === Step 3: ダッシュボードのタップ ===
    await page.waitForSelector('a#global_nav_dashboard_link', {
      visible: true,
      timeout: 10000
    });
    await page.click('a#global_nav_dashboard_link');
    console.log('🧑‍💻ダッシュボードをクリックしました');

    // === Step 3.5:待機画面 ===
    await new Promise(resolve => setTimeout(resolve, 5000));

    // === Step 6: スクショ保存 ===
    await page.screenshot({ path: 'klms_after_login.png', fullPage: true });
    console.log('📸 スクリーンショット保存完了');

    // Step 6.5: Gemini APIで文字起こし
    console.log('🤖 Gemini APIで文字起こし中...');
    const extractedText = await extractAssignmentInfo('klms_after_login.png');
    console.log('✅ 文字起こし完了:\n', extractedText);

// === Step 7: Googleドライブにアップロード（OCRテキストを説明欄に保存）===
const { file: uploadedFile, auth: oAuth2Client } = await uploadToGoogleDriveOAuth('klms_after_login.png');

// 🔹 ファイルの説明欄にGemini OCR結果を追加
const drive = google.drive({ version: 'v3', auth: oAuth2Client });
await drive.files.update({
  fileId: uploadedFile.id,
  requestBody: { description: extractedText }
});

console.log('🧠 Gemini文字起こし結果をDriveファイルの説明欄に保存しました');

 } finally {
  }
})();
