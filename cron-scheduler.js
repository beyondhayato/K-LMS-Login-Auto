require('dotenv').config();
const cron = require('node-cron');
const { exec } = require('child_process');
const fs = require('fs');

console.log('');
console.log('═══════════════════════════════════════');
console.log('⏰ KLMS自動実行スケジューラー起動');
console.log('═══════════════════════════════════════');
console.log('');

const lastRunFile = 'last-run.txt';

// === 実行関数 ===
function runScript(description) {
  const now = new Date();
  console.log('');
  console.log('🚀 実行開始');
  console.log('📅 実行時刻:', now.toLocaleString('ja-JP', { timeZone: 'Asia/Tokyo' }));
  console.log('📝 実行内容:', description);
  console.log('');
  
  exec('node token.js', (error, stdout, stderr) => {
    if (error) {
      console.error('❌ エラー:', error.message);
      return;
    }
    console.log(stdout);
    
    // 実行時刻を記録
    fs.writeFileSync(lastRunFile, now.toISOString());
    console.log('✅ 実行完了');
    console.log('');
  });
}

// === 最終実行時刻をチェック ===
function shouldRun() {
  if (!fs.existsSync(lastRunFile)) return true;
  
  const lastRun = new Date(fs.readFileSync(lastRunFile, 'utf8'));
  const now = new Date();
  const hoursSinceLastRun = (now - lastRun) / (1000 * 60 * 60);
  
  return hoursSinceLastRun >= 6; // 6時間以上経過していれば実行
}

// === 起動時に即実行 ===
if (shouldRun()) {
  console.log('🚀 起動時に即実行します（前回実行から6時間以上経過）');
  runScript('起動時実行');
} else {
  const lastRun = new Date(fs.readFileSync(lastRunFile, 'utf8'));
  const hoursSinceLastRun = ((new Date() - lastRun) / (1000 * 60 * 60)).toFixed(1);
  console.log(`⏭️ 前回実行から${hoursSinceLastRun}時間しか経過していないためスキップ`);
  console.log('');
}

// === 定期実行スケジュール ===
const schedules = [
  { time: '0 9 * * 1-5', description: '朝9時の定期実行' },
  { time: '0 12 * * 1-5', description: '昼12時の定期実行' },
  { time: '0 18 * * 1-5', description: '夕方18時の定期実行' }
];

schedules.forEach((schedule, index) => {
  cron.schedule(schedule.time, () => {
    if (shouldRun()) {
      runScript(schedule.description);
    } else {
      console.log(`⏭️ ${schedule.description}: 前回実行から6時間未満のためスキップ`);
    }
  }, {
    timezone: "Asia/Tokyo"
  });
  
  console.log(`📌 スケジュール${index + 1}: ${schedule.time} (${schedule.description})`);
});

console.log('');
console.log('✅ 全スケジュール登録完了');
console.log('💡 このターミナルを開いたままにしてください');
console.log('🛑 停止する場合は Ctrl+C を押してください');
console.log('');
console.log('⏰ 次回実行を待機中...');
console.log('');

process.stdin.resume();
