import { apiGet, apiPost } from "./client"

/*
  支付配置类型
*/
export interface PaymentConfig {
  id: string
  name: string
  provider: string
  enabled: boolean
  config: Record<string, unknown>
  created_at: string
  updated_at: string
}

/*
  paymentApi 支付管理 API 服务
  功能：封装支付配置的查询、更新和切换，以及管理员手动充值
  对齐后端路由：/admin/payment/*
*/
export const paymentApi = {
  /* 获取所有支付配置 */
  listConfigs: () => apiGet<PaymentConfig[]>("/admin/payment/configs"),

  /* 获取单个支付配置 */
  getConfig: (id: string) => apiGet<PaymentConfig>(`/admin/payment/config/${id}`),

  /* 更新支付配置 */
  updateConfig: (id: string, data: Record<string, unknown>) =>
    apiPost<PaymentConfig>(`/admin/payment/config/${id}/update`, data),

  /* 切换支付配置启用状态 */
  toggleConfig: (id: string) => apiPost(`/admin/payment/config/${id}/toggle`),

  /* 管理员手动充值 */
  manualRecharge: (data: { user_id: string; amount: number; reason: string }) =>
    apiPost("/admin/payment/manual-recharge", data),
}
