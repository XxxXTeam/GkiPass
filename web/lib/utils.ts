import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/*
  formatBytes 字节格式化工具
  功能：将字节数转换为人类可读的格式（如 1.5 GB）
*/
export function formatBytes(bytes: number, fallback = "0 B"): string {
  if (bytes === 0) return fallback
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB", "TB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i]
}

/*
  formatRelativeTime 相对时间格式化
  功能：将 ISO 时间字符串转换为"刚刚/x分钟前/x小时前/x天前"格式
*/
export function formatRelativeTime(dateStr: string): string {
  if (!dateStr) return "-"
  const date = new Date(dateStr)
  const now = new Date()
  const diff = Math.floor((now.getTime() - date.getTime()) / 1000)

  if (diff < 60) return "刚刚"
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前`
  if (diff < 2592000) return `${Math.floor(diff / 86400)} 天前`
  return date.toLocaleDateString("zh-CN")
}

/*
  formatDateTime 日期时间格式化
  功能：将 ISO 时间字符串转换为 "YYYY-MM-DD HH:mm" 格式
*/
export function formatDateTime(dateStr: string): string {
  if (!dateStr) return "-"
  const date = new Date(dateStr)
  return date.toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  })
}
