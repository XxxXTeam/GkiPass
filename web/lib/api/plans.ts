import { apiGet, apiPost } from "./client"
import type { Plan } from "@/lib/types"

/*
  planApi 套餐 API 服务
  功能：封装套餐管理相关的所有 HTTP 请求，包括增删改查和状态切换
*/
export const planApi = {
  list: () => apiGet<Plan[]>("/plans"),
  get: (id: string) => apiGet<Plan>(`/plans/${id}`),
  create: (data: Partial<Plan>) => apiPost<Plan>("/plans/create", data),
  update: (id: string, data: Partial<Plan>) => apiPost<Plan>(`/plans/${id}/update`, data),
  delete: (id: string) => apiPost(`/plans/${id}/delete`),
  toggleStatus: (id: string) => apiPost(`/plans/${id}/toggle`),

  /* 用户订阅套餐 */
  subscribe: (id: string, months?: number) => apiPost(`/plans/${id}/subscribe`, { months: months || 1 }),

  /* 获取当前用户的套餐订阅 */
  mySubscription: () => apiGet("/plans/my/subscription"),
}
