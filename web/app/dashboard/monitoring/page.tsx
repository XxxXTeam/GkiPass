"use client"

import { useEffect, useState } from "react"
import {
  Activity, Cpu, HardDrive, Wifi, ArrowDownRight, RefreshCw,
} from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { nodeApi } from "@/lib/api/nodes"
import type { Node } from "@/lib/types"
import { formatBytes } from "@/lib/utils"

function UsageBar({ value, max = 100, color }: { value: number; max?: number; color: string }) {
  const pct = Math.min((value / max) * 100, 100)
  return (
    <div className="flex items-center gap-2">
      <div className="h-2 flex-1 rounded-full bg-muted overflow-hidden">
        <div className={`h-full rounded-full ${color}`} style={{ width: `${pct}%` }} />
      </div>
      <span className="text-xs tabular-nums w-10 text-right">{pct.toFixed(0)}%</span>
    </div>
  )
}

export default function MonitoringPage() {
  const [nodes, setNodes] = useState<Node[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)

  const fetchNodes = async (isRefresh = false) => {
    if (isRefresh) setRefreshing(true)
    try {
      const res = await nodeApi.list()
      if (res.success && res.data) setNodes(res.data)
    } catch { /* 忽略 */ } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }

  useEffect(() => { fetchNodes() }, [])

  /* 自动刷新 30s */
  useEffect(() => {
    const timer = setInterval(() => fetchNodes(), 30000)
    return () => clearInterval(timer)
  }, [])

  const onlineNodes = nodes.filter((n) => n.status === "online")
  const totalConns = nodes.reduce((s, n) => s + (n.connection_count || 0), 0)
  const avgCpu = onlineNodes.length > 0
    ? onlineNodes.reduce((s, n) => s + (n.cpu_usage || 0), 0) / onlineNodes.length
    : 0
  const avgMem = onlineNodes.length > 0
    ? onlineNodes.reduce((s, n) => s + (n.memory_usage || 0), 0) / onlineNodes.length
    : 0

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">系统监控</h1>
          <p className="text-muted-foreground">实时监控节点资源使用和连接状态</p>
        </div>
        <Button variant="outline" size="sm" onClick={() => fetchNodes(true)} disabled={refreshing}>
          <RefreshCw className={`mr-2 h-4 w-4 ${refreshing ? "animate-spin" : ""}`} />
          刷新
        </Button>
      </div>

      {/* 概览卡片 */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <div className="rounded-lg bg-green-500/10 p-2.5">
              <Wifi className="h-5 w-5 text-green-500" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">在线节点</p>
              <p className="text-2xl font-bold">{onlineNodes.length}<span className="text-sm text-muted-foreground font-normal">/{nodes.length}</span></p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <div className="rounded-lg bg-blue-500/10 p-2.5">
              <Activity className="h-5 w-5 text-blue-500" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">当前连接</p>
              <p className="text-2xl font-bold">{totalConns}</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <div className="rounded-lg bg-amber-500/10 p-2.5">
              <Cpu className="h-5 w-5 text-amber-500" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">平均 CPU</p>
              <p className="text-2xl font-bold">{avgCpu.toFixed(1)}%</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <div className="rounded-lg bg-violet-500/10 p-2.5">
              <HardDrive className="h-5 w-5 text-violet-500" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">平均内存</p>
              <p className="text-2xl font-bold">{avgMem.toFixed(1)}%</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 节点详情表 */}
      <Card>
        <CardHeader><CardTitle className="text-base">节点资源详情</CardTitle></CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-3">
              {Array.from({ length: 4 }).map((_, i) => (
                <div key={i} className="h-12 animate-pulse rounded bg-muted" />
              ))}
            </div>
          ) : nodes.length === 0 ? (
            <p className="py-8 text-center text-muted-foreground">暂无节点数据</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>节点</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead className="w-40">CPU</TableHead>
                  <TableHead className="w-40">内存</TableHead>
                  <TableHead>连接数</TableHead>
                  <TableHead>带宽</TableHead>
                  <TableHead>最后上报</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {nodes.map((node) => (
                  <TableRow key={node.id}>
                    <TableCell>
                      <div>
                        <p className="font-medium">{node.name}</p>
                        <p className="text-xs text-muted-foreground font-mono">{node.ip}</p>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={node.status === "online" ? "default" : "secondary"}>
                        {node.status === "online" ? "在线" : node.status === "maintenance" ? "维护" : "离线"}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <UsageBar value={node.cpu_usage || 0} color={
                        (node.cpu_usage || 0) > 80 ? "bg-red-500" : (node.cpu_usage || 0) > 50 ? "bg-amber-500" : "bg-green-500"
                      } />
                    </TableCell>
                    <TableCell>
                      <UsageBar value={node.memory_usage || 0} color={
                        (node.memory_usage || 0) > 80 ? "bg-red-500" : (node.memory_usage || 0) > 50 ? "bg-amber-500" : "bg-blue-500"
                      } />
                    </TableCell>
                    <TableCell className="tabular-nums">
                      {node.connection_count || 0}
                      {node.max_connections > 0 && <span className="text-muted-foreground">/{node.max_connections}</span>}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1 text-xs">
                        <ArrowDownRight className="h-3 w-3 text-green-500" />
                        {formatBytes(node.bandwidth_limit || 0)}/s
                      </div>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {node.last_seen ? new Date(node.last_seen).toLocaleString("zh-CN") : "-"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
