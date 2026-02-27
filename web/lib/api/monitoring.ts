import { apiGet, apiPost } from "./client"
import type { MonitoringStats, NodeMetrics } from "@/lib/types"

/*
  monitoringApi 监控 API 服务
  功能：封装系统监控数据查询，包括节点指标、系统概览和历史数据
  对齐后端路由：/monitoring/*
*/
export const monitoringApi = {
  /* 获取整体监控概览 */
  overview: () => apiGet<MonitoringStats>("/monitoring/overview"),

  /* 获取监控汇总数据 */
  summary: () => apiGet("/monitoring/summary"),

  /* 获取单个节点的监控状态 */
  nodeStatus: (nodeId: string) => apiGet<NodeMetrics>(`/monitoring/nodes/${nodeId}/status`),

  /* 获取单个节点的监控数据 */
  nodeData: (nodeId: string) => apiGet<NodeMetrics>(`/monitoring/nodes/${nodeId}/data`),

  /* 获取节点历史性能数据 */
  nodeHistory: (nodeId: string, params?: { range?: string }) =>
    apiGet<NodeMetrics[]>(`/monitoring/nodes/${nodeId}/history`, { params }),

  /* 获取节点告警信息 */
  nodeAlerts: (nodeId: string) => apiGet(`/monitoring/nodes/${nodeId}/alerts`),

  /* 获取节点监控配置 */
  nodeConfig: (nodeId: string) => apiGet(`/monitoring/nodes/${nodeId}/config`),

  /* 更新节点监控配置（管理员） */
  updateNodeConfig: (nodeId: string, data: Record<string, unknown>) =>
    apiPost(`/monitoring/nodes/${nodeId}/config/update`, data),

  /* 获取当前用户的监控权限 */
  myPermissions: () => apiGet("/monitoring/my-permissions"),

  /* 管理员：获取监控权限列表 */
  listPermissions: () => apiGet("/monitoring/permissions"),

  /* 管理员：创建监控权限 */
  createPermission: (data: { user_id: string; node_id: string; permission_type: string }) =>
    apiPost("/monitoring/permissions", data),
}
