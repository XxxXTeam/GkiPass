import { apiGet, apiPost } from "./client"

/*
  流量统计相关类型
*/
export interface TrafficStat {
  id: string
  node_id: string
  node_name: string
  tunnel_id: string
  tunnel_name: string
  bytes_in: number
  bytes_out: number
  connections: number
  period: string
  created_at: string
}

export interface TrafficSummary {
  total_in: number
  total_out: number
  total_connections: number
  period: string
}

/*
  trafficApi 流量统计 API 服务
  功能：封装流量数据查询和上报
  对齐后端路由：/traffic/*
*/
export const trafficApi = {
  /* 获取流量统计列表 */
  list: (params?: { node_id?: string; tunnel_id?: string; period?: string }) =>
    apiGet<TrafficStat[]>("/traffic/stats", { params }),

  /* 获取流量汇总 */
  summary: (params?: { period?: string }) =>
    apiGet<TrafficSummary>("/traffic/summary", { params }),

  /* 节点上报流量（供节点调用） */
  report: (data: Record<string, unknown>) =>
    apiPost("/traffic/report", data),
}
