require('dotenv').config();
const cron = require('node-cron');
const { exec } = require('child_process');

console.log('🧪 cronテストモード起動');
console.log('📅 1分ごとに実行されます');
console.log('🛑 停止する場合は Ctrl+C を押してください');
console.log('');

// テスト用: 1分ごとに実行
cron.schedule('* * * * *', () => {
  const now = new Date().toLocaleString('ja-JP', { timeZone: 'Asia/Tokyo' });
  console.log('');
  console.log('🚀 テスト実行:', now);
  
  exec('node token.js', (error, stdout, stderr) => {
    if (error) {
      console.error('❌ エラー:', error.message);
      return;
    }
    console.log(stdout);
    console.log('✅ テスト実行完了');
    console.log('');
  });
}, {
  timezone: "Asia/Tokyo"
});
