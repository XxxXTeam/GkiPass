import { apiGet, apiPost } from "./client"
import type { SystemSettings } from "@/lib/types"

/*
  settingsApi 系统设置 API 服务
  功能：封装系统设置的读取和更新，对齐后端 /admin/settings/* 路由
*/
export const settingsApi = {
  /* 获取通用设置 */
  get: () => apiGet<SystemSettings>("/admin/settings/general"),

  /* 更新通用设置 */
  update: (data: Record<string, unknown>) => apiPost<SystemSettings>("/admin/settings/general/update", data),

  /* 获取安全设置 */
  getSecurity: () => apiGet("/admin/settings/security"),

  /* 更新安全设置 */
  updateSecurity: (data: Record<string, unknown>) => apiPost("/admin/settings/security/update", data),

  /* 获取通知设置 */
  getNotification: () => apiGet("/admin/settings/notification"),

  /* 更新通知设置 */
  updateNotification: (data: Record<string, unknown>) => apiPost("/admin/settings/notification/update", data),

  /* 获取验证码设置 */
  getCaptcha: () => apiGet("/admin/settings/captcha"),

  /* 更新验证码设置 */
  updateCaptcha: (data: Record<string, unknown>) => apiPost("/admin/settings/captcha/update", data),
}
