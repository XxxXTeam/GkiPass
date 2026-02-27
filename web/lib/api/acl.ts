import { apiGet, apiPost } from "./client"
import type { Policy } from "@/lib/types"

/*
  policyApi 协议转发策略 API 服务
  功能：管理隧道协议限制策略（禁止/允许特定协议转发）
  对齐后端路由：/policies/*，后端 PolicyConfig.Protocols 字段控制协议白名单
*/
export const policyApi = {
  /* 获取策略列表（可按类型过滤） */
  list: (type?: string) =>
    apiGet<{ policies: Policy[]; total: number }>("/policies", type ? { params: { type } } : undefined),

  /* 获取策略详情 */
  get: (id: string) => apiGet<Policy>(`/policies/${id}`),

  /* 创建策略 */
  create: (data: {
    name: string
    type: "protocol"
    priority?: number
    enabled: boolean
    config: { protocols: string[] }
    node_ids?: string[]
    description?: string
  }) => apiPost<Policy>("/policies/create", data),

  /* 更新策略 */
  update: (id: string, data: Partial<{
    name: string
    priority: number
    enabled: boolean
    config: { protocols: string[] }
    node_ids: string[]
    description: string
  }>) => apiPost<Policy>(`/policies/${id}/update`, data),

  /* 删除策略 */
  delete: (id: string) => apiPost(`/policies/${id}/delete`),

  /* 部署策略到节点 */
  deploy: (id: string) => apiPost(`/policies/${id}/deploy`),
}
