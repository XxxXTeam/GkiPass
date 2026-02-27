import { apiGet } from "./client"
import type { DashboardStats } from "@/lib/types"

/*
  dashboardApi 仪表盘 API 服务
  功能：封装仪表盘统计数据和概览信息请求
*/
export const dashboardApi = {
  /* 用户概览统计 */
  stats: () => apiGet<DashboardStats>("/statistics/overview"),

  /* 管理员概览统计（含更详细的系统级数据） */
  adminOverview: () => apiGet("/admin/statistics/overview"),
}
