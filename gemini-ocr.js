// gemini-ocr.js
const { GoogleGenerativeAI } = require("@google/generative-ai");
const fs = require('fs');

async function extractAssignmentInfo(imagePath) {
  const genAI = new GoogleGenerativeAI(process.env.GEMINI_API_KEY);
  const model = genAI.getGenerativeModel({ model: "gemini-2.5-flash" });

  // 画像を読み込み
  const imageData = fs.readFileSync(imagePath);
  const base64Image = imageData.toString('base64');

  const prompt = `
この画像はKLMS（慶應義塾大学の学習管理システム）のダッシュボードのスクリーンショットです。

以下の情報を抽出してください：
1. 授業名
2. 課題のタイトル
3. 提出期限（日時）

複数の課題がある場合は、すべて抽出してください。
情報が見つからない場合は「なし」と記載してください。

出力形式：
【授業名】
【課題】
【期限】

簡潔に、箇条書きで出力してください。
`;

  const result = await model.generateContent([
    prompt,
    {
      inlineData: {
        mimeType: "image/png",
        data: base64Image
      }
    }
  ]);

  const response = await result.response;
  return response.text();
}

module.exports = { extractAssignmentInfo };
