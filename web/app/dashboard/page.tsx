"use client"

import { useEffect, useState, useCallback } from "react"
import {
  Network,
  Server,
  Users,
  ArrowUpRight,
  ArrowDownRight,
  Activity,
  Zap,
  Globe,
  RefreshCw,
  Info,
  AlertTriangle,
} from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { dashboardApi } from "@/lib/api/dashboard"
import { announcementApi, type Announcement } from "@/lib/api/announcements"
import { tunnelApi } from "@/lib/api/tunnels"
import type { DashboardStats, Tunnel } from "@/lib/types"
import { formatBytes } from "@/lib/utils"
import dynamic from "next/dynamic"
import Link from "next/link"

const TrafficChart = dynamic(
  () => import("@/components/charts/traffic-chart").then((mod) => mod.TrafficChart),
  { ssr: false, loading: () => <div className="h-[260px] animate-pulse rounded bg-muted" /> }
)

/* 模拟流量数据（后端实现历史 API 后替换） */
function generateMockTrafficData(hours = 24) {
  const now = new Date()
  return Array.from({ length: hours }, (_, i) => {
    const time = new Date(now.getTime() - (hours - 1 - i) * 3600000)
    return {
      time: `${time.getHours().toString().padStart(2, "0")}:00`,
      inbound: Math.floor(Math.random() * 500 * 1024 * 1024),
      outbound: Math.floor(Math.random() * 300 * 1024 * 1024),
    }
  })
}

const REFRESH_INTERVAL = 30000

export default function DashboardPage() {
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [announcements, setAnnouncements] = useState<Announcement[]>([])
  const [recentTunnels, setRecentTunnels] = useState<Tunnel[]>([])

  const fetchStats = useCallback(async (silent = false) => {
    if (!silent) setRefreshing(true)
    try {
      const res = await dashboardApi.stats()
      if (res.success && res.data) setStats(res.data)
    } catch {
      if (!stats) {
        setStats({
          total_tunnels: 0, active_tunnels: 0, total_nodes: 0, online_nodes: 0,
          total_users: 0, active_users: 0, traffic_in_today: 0, traffic_out_today: 0,
          total_connections: 0,
        })
      }
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [stats])

  useEffect(() => {
    fetchStats()
    announcementApi.listActive().then((res) => {
      if (res.success && res.data) setAnnouncements(res.data.slice(0, 3))
    }).catch(() => {})
    tunnelApi.list().then((res) => {
      if (res.success && res.data) setRecentTunnels(res.data.slice(0, 5))
    }).catch(() => {})
    const timer = setInterval(() => fetchStats(true), REFRESH_INTERVAL)
    return () => clearInterval(timer)
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const statCards = stats
    ? [
        {
          title: "隧道总数",
          value: stats.total_tunnels,
          sub: `${stats.active_tunnels} 活跃`,
          icon: Network,
          color: "text-blue-500",
          bg: "bg-blue-500/10",
        },
        {
          title: "节点总数",
          value: stats.total_nodes,
          sub: `${stats.online_nodes} 在线`,
          icon: Server,
          color: "text-green-500",
          bg: "bg-green-500/10",
        },
        {
          title: "用户总数",
          value: stats.total_users,
          sub: `${stats.active_users} 活跃`,
          icon: Users,
          color: "text-violet-500",
          bg: "bg-violet-500/10",
        },
        {
          title: "当前连接",
          value: stats.total_connections,
          sub: "实时",
          icon: Zap,
          color: "text-amber-500",
          bg: "bg-amber-500/10",
        },
      ]
    : []

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">仪表盘</h1>
          <p className="text-muted-foreground">GkiPass 隧道管理控制面板概览</p>
        </div>
        <Button variant="outline" size="sm" onClick={() => fetchStats()} disabled={refreshing}>
          <RefreshCw className={`mr-2 h-4 w-4 ${refreshing ? "animate-spin" : ""}`} />
          {refreshing ? "刷新中..." : "刷新数据"}
        </Button>
      </div>

      {/* 活跃公告 */}
      {announcements.length > 0 && (
        <div className="space-y-2">
          {announcements.map((a) => {
            const isWarning = a.type === "warning" || a.type === "maintenance"
            return (
              <div
                key={a.id}
                className={`flex items-start gap-3 rounded-lg border p-3 ${
                  isWarning ? "border-amber-500/30 bg-amber-500/5" : "border-blue-500/30 bg-blue-500/5"
                }`}
              >
                {isWarning ? (
                  <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-500" />
                ) : (
                  <Info className="mt-0.5 h-4 w-4 shrink-0 text-blue-500" />
                )}
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium">{a.title}</p>
                  {a.content && (
                    <p className="text-xs text-muted-foreground mt-0.5 line-clamp-1">{a.content}</p>
                  )}
                </div>
                <Badge variant="outline" className="shrink-0 text-[10px]">
                  {a.type === "maintenance" ? "维护" : a.type === "warning" ? "警告" : a.type === "update" ? "更新" : "公告"}
                </Badge>
              </div>
            )
          })}
        </div>
      )}

      {/* 统计卡片 */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {loading
          ? Array.from({ length: 4 }).map((_, i) => (
              <Card key={i}>
                <CardContent className="p-6">
                  <div className="h-20 animate-pulse rounded bg-muted" />
                </CardContent>
              </Card>
            ))
          : statCards.map((card) => (
              <Card key={card.title}>
                <CardContent className="p-6">
                  <div className="flex items-center justify-between">
                    <div className="space-y-1">
                      <p className="text-sm text-muted-foreground">
                        {card.title}
                      </p>
                      <p className="text-3xl font-bold">{card.value}</p>
                      <Badge variant="secondary" className="text-xs">
                        {card.sub}
                      </Badge>
                    </div>
                    <div className={`rounded-lg p-3 ${card.bg}`}>
                      <card.icon className={`h-5 w-5 ${card.color}`} />
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
      </div>

      {/* 流量概览 */}
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base">
              <ArrowDownRight className="h-4 w-4 text-green-500" />
              今日入站流量
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-bold">
              {stats ? formatBytes(stats.traffic_in_today) : "--"}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base">
              <ArrowUpRight className="h-4 w-4 text-blue-500" />
              今日出站流量
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-bold">
              {stats ? formatBytes(stats.traffic_out_today) : "--"}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* 流量趋势图表 */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base">24小时流量趋势</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-4 mb-3 text-xs text-muted-foreground">
            <span className="flex items-center gap-1">
              <span className="inline-block h-2 w-2 rounded-full bg-green-500" /> 入站
            </span>
            <span className="flex items-center gap-1">
              <span className="inline-block h-2 w-2 rounded-full bg-blue-500" /> 出站
            </span>
          </div>
          <TrafficChart data={generateMockTrafficData(24)} />
        </CardContent>
      </Card>

      {/* 最近隧道 */}
      {recentTunnels.length > 0 && (
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">最近隧道</CardTitle>
              <Link href="/dashboard/tunnels" className="text-xs text-muted-foreground hover:text-foreground">查看全部 →</Link>
            </div>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {recentTunnels.map((t) => (
                <div key={t.id} className="flex items-center justify-between rounded-lg border px-3 py-2">
                  <div className="flex items-center gap-3">
                    <div className={`h-2 w-2 rounded-full ${t.enabled ? "bg-green-500" : "bg-muted-foreground"}`} />
                    <div>
                      <p className="text-sm font-medium">{t.name}</p>
                      <p className="text-xs text-muted-foreground">
                        {t.protocol.toUpperCase()} :{t.listen_port} → {t.target_address}:{t.target_port}
                      </p>
                    </div>
                  </div>
                  <Badge variant={t.enabled ? "default" : "secondary"} className="text-[10px]">
                    {t.enabled ? "运行中" : "已停止"}
                  </Badge>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* 快捷操作 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">快捷操作</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            {[
              { title: "创建隧道", icon: Network, href: "/dashboard/tunnels?action=create", color: "text-blue-500" },
              { title: "添加节点", icon: Server, href: "/dashboard/nodes?action=create", color: "text-green-500" },
              { title: "系统监控", icon: Activity, href: "/dashboard/monitoring", color: "text-orange-500" },
              { title: "订阅与钱包", icon: Globe, href: "/dashboard/subscription", color: "text-violet-500" },
            ].map((action) => (
              <Link
                key={action.title}
                href={action.href}
                className="flex items-center gap-3 rounded-lg border p-3 transition-colors hover:bg-accent"
              >
                <action.icon className={`h-5 w-5 ${action.color}`} />
                <span className="text-sm font-medium">{action.title}</span>
              </Link>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
