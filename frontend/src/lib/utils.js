import { clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs) {
  return twMerge(clsx(inputs))
}

export function formatLogTime(timeStr) {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  if (isNaN(date.getTime())) return timeStr
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const yesterday = new Date(today.getTime() - 24 * 60 * 60 * 1000)
  const logDate = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  let datePart = ''
  if (logDate.getTime() === today.getTime()) {
    datePart = '今天'
  } else if (logDate.getTime() === yesterday.getTime()) {
    datePart = '昨天'
  } else {
    datePart = `${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
  }
  const timePart = `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`
  return `${datePart} ${timePart}`
}
