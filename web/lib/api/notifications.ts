import { apiGet, apiPost } from "./client"

/*
  通知相关类型
*/
export interface Notification {
  id: string
  title: string
  content: string
  type: "info" | "warning" | "error" | "success"
  read: boolean
  created_at: string
}

/*
  notificationApi 通知 API 服务
  功能：封装通知的查询、已读标记和删除
  对齐后端路由：/notifications/*
*/
export const notificationApi = {
  /* 获取通知列表 */
  list: () => apiGet<Notification[]>("/notifications"),

  /* 标记单条为已读 */
  markAsRead: (id: string) => apiPost(`/notifications/${id}/read`),

  /* 全部标记为已读 */
  markAllAsRead: () => apiPost("/notifications/read-all"),

  /* 删除通知 */
  delete: (id: string) => apiPost(`/notifications/${id}/delete`),
}
