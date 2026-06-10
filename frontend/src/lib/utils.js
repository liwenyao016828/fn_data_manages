import { clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs) {
  return twMerge(clsx(inputs))
}

export function getTypeLabel(type) {
  const labels = {
    mysql: 'MySQL',
    mariadb: 'MariaDB',
    postgresql: 'PostgreSQL',
    redis: 'Redis',
    sqlite: 'SQLite',
  }
  return labels[type] || type
}

export function getTypeColor(type) {
  const colors = {
    mysql: '#3b82f6',
    mariadb: '#06b6d4',
    postgresql: '#6366f1',
    redis: '#f59e0b',
    sqlite: '#10b981',
  }
  return colors[type] || 'var(--text-tertiary)'
}

export function getTypeSoftColor(type) {
  const colors = {
    mysql: 'rgba(59, 130, 246, 0.1)',
    mariadb: 'rgba(6, 182, 212, 0.1)',
    postgresql: 'rgba(99, 102, 241, 0.1)',
    redis: 'rgba(245, 158, 11, 0.1)',
    sqlite: 'rgba(16, 185, 129, 0.1)',
  }
  return colors[type] || 'var(--muted)'
}

export function isSqlType(type) {
  return ['mysql', 'mariadb', 'postgresql', 'sqlite'].includes(type)
}

export function getTypeBadgeClass(type) {
  const classes = {
    mysql: 'bg-blue-500/5 text-blue-600 border-blue-500/20',
    mariadb: 'bg-cyan-500/5 text-cyan-600 border-cyan-500/20',
    postgresql: 'bg-indigo-500/5 text-indigo-600 border-indigo-500/20',
    redis: 'bg-amber-500/5 text-amber-600 border-amber-500/20',
    sqlite: 'bg-emerald-500/5 text-emerald-600 border-emerald-500/20',
  }
  return classes[type] || 'bg-gray-500/5 text-gray-600 border-gray-500/20'
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
