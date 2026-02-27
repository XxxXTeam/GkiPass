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
