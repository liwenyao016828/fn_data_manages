export const writeLog = (message, level = 'info', source = 'USER') => {
  fetch('/api/system/logs/write', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ level, source, message })
  }).catch(() => {})
}
