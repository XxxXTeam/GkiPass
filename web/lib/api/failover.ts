import { apiGet } from "./client"
import type { FailoverEvent, ActiveFailover, GroupFailoverSummary } from "@/lib/types"

/*
  failoverApi 容灾事件 API 服务
  功能：封装出口容灾事件查询接口，供仪表盘展示容灾切换状态
  对齐后端路由：/failover/*
*/

export const failoverApi = {
  /* 获取当前所有活跃容灾状态 */
  getActive: () =>
    apiGet<{ active_failovers: ActiveFailover[]; count: number }>("/failover/active"),

  /* 获取隧道容灾历史 */
  getTunnelHistory: (tunnelId: string, limit = 20) =>
    apiGet<{ tunnel_id: string; events: FailoverEvent[]; count: number }>(
      `/failover/tunnels/${tunnelId}/history`,
      { params: { limit } }
    ),

  /* 获取出口组容灾摘要 */
  getGroupSummary: (groupId: string) =>
    apiGet<GroupFailoverSummary>(`/failover/groups/${groupId}/summary`),
}
