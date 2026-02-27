"use client"

import { useEffect, useState } from "react"
import { ArrowDownRight, ArrowUpRight, Activity, RefreshCw } from "lucide-react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import { trafficApi, type TrafficStat, type TrafficSummary } from "@/lib/api/traffic"
import { formatBytes } from "@/lib/utils"
import { exportCsv } from "@/lib/export-csv"

export default function TrafficPage() {
  const [stats, setStats] = useState<TrafficStat[]>([])
  const [summary, setSummary] = useState<TrafficSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [period, setPeriod] = useState("today")

  const fetchData = async () => {
    setLoading(true)
    try {
      const [statsRes, summaryRes] = await Promise.allSettled([
        trafficApi.list({ period }),
        trafficApi.summary({ period }),
      ])
      if (statsRes.status === "fulfilled" && statsRes.value.success && statsRes.value.data) {
        setStats(statsRes.value.data)
      }
      if (summaryRes.status === "fulfilled" && summaryRes.value.success && summaryRes.value.data) {
        setSummary(summaryRes.value.data)
      }
    } catch { /* 忽略 */ }
    finally { setLoading(false) }
  }

  useEffect(() => { fetchData() }, [period]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">流量统计</h1>
          <p className="text-muted-foreground">查看系统流量使用情况</p>
        </div>
        <div className="flex items-center gap-2">
          <Select value={period} onValueChange={setPeriod}>
            <SelectTrigger className="w-32 h-8 text-sm">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="today">今日</SelectItem>
              <SelectItem value="week">本周</SelectItem>
              <SelectItem value="month">本月</SelectItem>
              <SelectItem value="all">全部</SelectItem>
            </SelectContent>
          </Select>
          <Button variant="outline" size="sm" onClick={() => {
            exportCsv(stats, [
              { header: "节点", accessor: (s) => s.node_name || s.node_id },
              { header: "隧道", accessor: (s) => s.tunnel_name || s.tunnel_id },
              { header: "入站流量", accessor: (s) => formatBytes(s.bytes_in) },
              { header: "出站流量", accessor: (s) => formatBytes(s.bytes_out) },
              { header: "连接数", accessor: (s) => s.connections },
            ], "流量统计")
          }}>
            导出 CSV
          </Button>
          <Button variant="outline" size="sm" onClick={fetchData}>
            <RefreshCw className="mr-1 h-3 w-3" /> 刷新
          </Button>
        </div>
      </div>

      {/* 汇总卡片 */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <p className="text-sm text-muted-foreground">入站流量</p>
                <p className="text-3xl font-bold">{summary ? formatBytes(summary.total_in) : "--"}</p>
              </div>
              <div className="rounded-lg bg-green-500/10 p-3">
                <ArrowDownRight className="h-5 w-5 text-green-500" />
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <p className="text-sm text-muted-foreground">出站流量</p>
                <p className="text-3xl font-bold">{summary ? formatBytes(summary.total_out) : "--"}</p>
              </div>
              <div className="rounded-lg bg-blue-500/10 p-3">
                <ArrowUpRight className="h-5 w-5 text-blue-500" />
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <p className="text-sm text-muted-foreground">总连接数</p>
                <p className="text-3xl font-bold">{summary?.total_connections ?? "--"}</p>
              </div>
              <div className="rounded-lg bg-amber-500/10 p-3">
                <Activity className="h-5 w-5 text-amber-500" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 明细表格 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">流量明细</CardTitle>
          <CardDescription>按节点和隧道统计的流量数据</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="h-12 animate-pulse rounded bg-muted" />
              ))}
            </div>
          ) : stats.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">暂无流量数据</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>节点</TableHead>
                  <TableHead>隧道</TableHead>
                  <TableHead>入站</TableHead>
                  <TableHead>出站</TableHead>
                  <TableHead>连接数</TableHead>
                  <TableHead>时间</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {stats.map((s) => (
                  <TableRow key={s.id}>
                    <TableCell className="font-medium">{s.node_name || s.node_id}</TableCell>
                    <TableCell>{s.tunnel_name || s.tunnel_id}</TableCell>
                    <TableCell>
                      <span className="flex items-center gap-1 text-green-600">
                        <ArrowDownRight className="h-3 w-3" /> {formatBytes(s.bytes_in)}
                      </span>
                    </TableCell>
                    <TableCell>
                      <span className="flex items-center gap-1 text-blue-600">
                        <ArrowUpRight className="h-3 w-3" /> {formatBytes(s.bytes_out)}
                      </span>
                    </TableCell>
                    <TableCell>
                      <Badge variant="secondary">{s.connections}</Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {new Date(s.created_at).toLocaleString("zh-CN")}
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
