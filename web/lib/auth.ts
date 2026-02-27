/*
  auth 认证工具
  功能：客户端 Token 管理和认证状态检查
  说明：静态导出模式，仅使用 localStorage 存储
*/

const TOKEN_KEY = "gkipass_token"
const ROLE_KEY = "gkipass_role"

export function getToken(): string | null {
  if (typeof window === "undefined") return null
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  if (typeof window === "undefined") return
  localStorage.setItem(TOKEN_KEY, token)
}

export function setRole(role: string): void {
  if (typeof window === "undefined") return
  localStorage.setItem(ROLE_KEY, role)
}

export function getRole(): string | null {
  if (typeof window === "undefined") return null
  return localStorage.getItem(ROLE_KEY)
}

export function removeToken(): void {
  if (typeof window === "undefined") return
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(ROLE_KEY)
}

export function isAuthenticated(): boolean {
  return !!getToken()
}
