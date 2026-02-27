import { apiGet } from "./client"

/*
  订阅相关类型
*/
export interface Subscription {
  id: string
  user_id: string
  plan_id: string
  plan_name: string
  status: string
  started_at: string
  expires_at: string
  traffic_used: number
  traffic_limit: number
}

/*
  subscriptionApi 订阅 API 服务
  功能：封装用户订阅信息查询和管理员订阅列表
  对齐后端路由：/subscriptions/*、/plans/my/subscription
*/
export const subscriptionApi = {
  /* 获取当前用户订阅 */
  current: () => apiGet<Subscription>("/subscriptions/current"),

  /* 获取用户的套餐订阅（旧接口兼容） */
  mySubscription: () => apiGet<Subscription>("/plans/my/subscription"),

  /* 管理员：获取所有订阅列表 */
  list: () => apiGet<Subscription[]>("/subscriptions"),
}
