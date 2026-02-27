import { apiGet, apiPost } from "./client"
import type { Node, NodeGroup } from "@/lib/types"

/*
  nodeApi 节点 API 服务
  功能：封装节点相关的所有 HTTP 请求
*/
export const nodeApi = {
  list: () => apiGet<Node[]>("/nodes/list"),
  get: (id: string) => apiGet<Node>(`/nodes/${id}`),
  create: (data: Partial<Node>) => apiPost<Node>("/nodes/create", data),
  update: (id: string, data: Partial<Node>) => apiPost<Node>(`/nodes/${id}/update`, data),
  delete: (id: string) => apiPost(`/nodes/${id}/delete`),

  /* 节点状态 */
  status: (id: string) => apiGet(`/nodes/${id}/status`),

  /* 证书管理 */
  generateCert: (id: string) => apiPost(`/nodes/${id}/cert/generate`),
  downloadCert: (id: string) => apiGet(`/nodes/${id}/cert/download`),
  renewCert: (id: string) => apiPost(`/nodes/${id}/cert/renew`),
  certInfo: (id: string) => apiGet(`/nodes/${id}/cert/info`),

  /* 用户可用节点（根据套餐过滤） */
  available: () => apiGet<Node[]>("/nodes/available"),

  /* 节点心跳 */
  heartbeat: (id: string) => apiPost(`/nodes/${id}/heartbeat`),

  /* Connection Key 管理 */
  generateCK: (id: string) => apiPost(`/nodes/${id}/generate-ck`),
  listCKs: (id: string) => apiGet(`/nodes/${id}/connection-keys`),
  revokeCK: (ckId: string) => apiPost(`/nodes/connection-keys/${ckId}/revoke`),
}

/*
  nodeGroupApi 节点组 API 服务
  对齐后端路由：/node-groups/*、/node-groups/:id/config
*/
export const nodeGroupApi = {
  list: () => apiGet<NodeGroup[]>("/node-groups/list"),
  get: (id: string) => apiGet<NodeGroup>(`/node-groups/${id}`),
  create: (data: Partial<NodeGroup>) => apiPost<NodeGroup>("/node-groups/create", data),
  update: (id: string, data: Partial<NodeGroup>) => apiPost<NodeGroup>(`/node-groups/${id}/update`, data),
  delete: (id: string) => apiPost(`/node-groups/${id}/delete`),

  /* 获取节点组配置 */
  getConfig: (id: string) => apiGet(`/node-groups/${id}/config`),

  /* 更新节点组配置 */
  updateConfig: (id: string, data: Record<string, unknown>) =>
    apiPost(`/node-groups/${id}/config/update`, data),

  /* 重置节点组配置为默认值 */
  resetConfig: (id: string) => apiPost(`/node-groups/${id}/config/reset`),

  /* 获取节点组内的节点列表 */
  listNodes: (id: string) => apiGet<Node[]>(`/node-groups/${id}/nodes`),
}
