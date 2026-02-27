import { apiGet, apiPost } from "./client"

/*
  证书相关类型
*/
export interface Certificate {
  id: string
  type: "ca" | "leaf"
  subject: string
  issuer: string
  serial_number: string
  not_before: string
  not_after: string
  is_revoked: boolean
  created_at: string
}

/*
  certificateApi 证书管理 API 服务
  功能：封装 CA 和叶子证书的生成、查询、吊销和下载
  对齐后端路由：/certificates/*
*/
export const certificateApi = {
  /* 获取证书列表 */
  list: () => apiGet<Certificate[]>("/certificates"),

  /* 获取证书详情 */
  get: (id: string) => apiGet<Certificate>(`/certificates/${id}`),

  /* 生成 CA 证书 */
  generateCA: (data?: Record<string, unknown>) => apiPost("/certificates/ca", data),

  /* 生成叶子证书 */
  generateLeaf: (data: Record<string, unknown>) => apiPost("/certificates/leaf", data),

  /* 吊销证书 */
  revoke: (id: string) => apiPost(`/certificates/${id}/revoke`),

  /* 下载证书 */
  downloadUrl: (id: string) => `/certificates/${id}/download`,
}
