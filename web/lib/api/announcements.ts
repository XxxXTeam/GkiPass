import { apiGet, apiPost } from "./client"

/*
  公告相关类型
*/
export interface Announcement {
  id: string
  title: string
  content: string
  type: "info" | "warning" | "maintenance" | "update"
  priority: number
  is_active: boolean
  created_at: string
  updated_at: string
}

/*
  announcementApi 公告 API 服务
  功能：封装公告的查询（公开）和管理（管理员CRUD）
  对齐后端路由：/announcements（公开）+ /admin/announcements（管理员）
*/
export const announcementApi = {
  /* 公开：获取活跃公告列表 */
  listActive: () => apiGet<Announcement[]>("/announcements"),

  /* 公开：获取公告详情 */
  get: (id: string) => apiGet<Announcement>(`/announcements/${id}`),

  /* 管理员：获取全部公告列表（含已禁用） */
  listAll: () => apiGet<Announcement[]>("/admin/announcements"),

  /* 管理员：创建公告 */
  create: (data: Partial<Announcement>) => apiPost<Announcement>("/admin/announcements/create", data),

  /* 管理员：更新公告 */
  update: (id: string, data: Partial<Announcement>) => apiPost<Announcement>(`/admin/announcements/${id}/update`, data),

  /* 管理员：删除公告 */
  delete: (id: string) => apiPost(`/admin/announcements/${id}/delete`),
}
