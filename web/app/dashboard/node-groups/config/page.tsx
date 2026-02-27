"use client"

import { useEffect, useState } from "react"
import { useSearchParams, useRouter } from "next/navigation"
import { ArrowLeft, Save, RotateCcw } from "lucide-react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { toast } from "sonner"
import { nodeGroupApi } from "@/lib/api/nodes"

/*
  节点组配置类型
  功能：对齐后端 NodeGroupConfigRequest 结构
*/
interface GroupConfig {
  allowed_protocols: string[]
  port_range: string
  port_range_start: number
  port_range_end: number
  traffic_multiplier: number
}

const PROTOCOLS = ["tcp", "udp", "ws", "wss", "tls", "tls-mux", "kcp", "quic"]

const defaultConfig: GroupConfig = {
  allowed_protocols: [],
  port_range: "10000-60000",
  port_range_start: 10000,
  port_range_end: 60000,
  traffic_multiplier: 1.0,
}

/*
  节点组配置页
  功能：通过 ?id=xxx 查询参数获取节点组 ID，管理协议、端口和流量配置
  路由：/dashboard/node-groups/config?id=xxx
*/
export default function NodeGroupConfigPage() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const groupId = searchParams.get("id") || ""

  const [config, setConfig] = useState<GroupConfig>(defaultConfig)
  const [groupName, setGroupName] = useState("")
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!groupId) { setLoading(false); return }
    const fetchData = async () => {
      try {
        const [groupRes, configRes] = await Promise.allSettled([
          nodeGroupApi.get(groupId),
          nodeGroupApi.getConfig(groupId),
        ])
        if (groupRes.status === "fulfilled" && groupRes.value.success && groupRes.value.data) {
          setGroupName(groupRes.value.data.name)
        }
        if (configRes.status === "fulfilled" && configRes.value.success && configRes.value.data) {
          const d = configRes.value.data as Record<string, unknown>
          setConfig({
            allowed_protocols: (d.allowed_protocols as string[]) || [],
            port_range: (d.port_range as string) || "10000-60000",
            port_range_start: (d.port_range_start as number) || 10000,
            port_range_end: (d.port_range_end as number) || 60000,
            traffic_multiplier: (d.traffic_multiplier as number) || 1.0,
          })
        }
      } catch { /* 使用默认值 */ }
      finally { setLoading(false) }
    }
    fetchData()
  }, [groupId])

  const handleSave = async () => {
    setSaving(true)
    try {
      await nodeGroupApi.updateConfig(groupId, {
        ...config,
        port_range: `${config.port_range_start}-${config.port_range_end}`,
      })
      toast.success("配置已保存")
    } catch { toast.error("保存失败") }
    finally { setSaving(false) }
  }

  const handleReset = async () => {
    try {
      await nodeGroupApi.resetConfig(groupId)
      setConfig(defaultConfig)
      toast.success("配置已重置为默认值")
    } catch { toast.error("重置失败") }
  }

  const toggleProtocol = (protocol: string) => {
    setConfig((prev) => ({
      ...prev,
      allowed_protocols: prev.allowed_protocols.includes(protocol)
        ? prev.allowed_protocols.filter((p) => p !== protocol)
        : [...prev.allowed_protocols, protocol],
    }))
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="h-7 w-60 animate-pulse rounded bg-muted" />
        <div className="space-y-4">
          {Array.from({ length: 2 }).map((_, i) => (
            <div key={i} className="h-40 animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      </div>
    )
  }

  if (!groupId) {
    return (
      <div className="flex flex-col items-center justify-center py-20">
        <p className="text-muted-foreground">缺少节点组 ID</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push("/dashboard/node-groups")}>
          返回节点组列表
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" onClick={() => router.push("/dashboard/node-groups")}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              {groupName || "节点组"} - 配置
            </h1>
            <p className="text-muted-foreground">管理节点组的协议、端口和流量配置</p>
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={handleReset}>
            <RotateCcw className="mr-2 h-4 w-4" /> 重置默认
          </Button>
          <Button size="sm" onClick={handleSave} disabled={saving}>
            <Save className="mr-2 h-4 w-4" />
            {saving ? "保存中..." : "保存配置"}
          </Button>
        </div>
      </div>

      {/* 协议配置 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">允许的协议</CardTitle>
          <CardDescription>选择此节点组允许使用的协议类型，留空表示允许全部</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-3">
            {PROTOCOLS.map((protocol) => (
              <div key={protocol} className="flex items-center gap-2 rounded-lg border px-3 py-2">
                <Switch
                  checked={config.allowed_protocols.includes(protocol)}
                  onCheckedChange={() => toggleProtocol(protocol)}
                />
                <span className="text-sm font-medium uppercase">{protocol}</span>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* 端口范围 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">端口范围</CardTitle>
          <CardDescription>设置此节点组可分配的端口范围</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4 max-w-md">
            <div className="grid gap-2">
              <Label>起始端口</Label>
              <Input
                type="number"
                value={config.port_range_start}
                onChange={(e) => setConfig({ ...config, port_range_start: parseInt(e.target.value) || 10000 })}
                min={1}
                max={65535}
              />
            </div>
            <div className="grid gap-2">
              <Label>结束端口</Label>
              <Input
                type="number"
                value={config.port_range_end}
                onChange={(e) => setConfig({ ...config, port_range_end: parseInt(e.target.value) || 60000 })}
                min={1}
                max={65535}
              />
            </div>
          </div>
          <p className="text-xs text-muted-foreground mt-2">
            当前范围: {config.port_range_start} - {config.port_range_end}
            （共 {Math.max(0, config.port_range_end - config.port_range_start + 1)} 个端口）
          </p>
        </CardContent>
      </Card>

      {/* 流量倍率 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">流量倍率</CardTitle>
          <CardDescription>设置流量计费倍率，1.0 表示原始计费</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-2 max-w-xs">
            <Label>倍率</Label>
            <Input
              type="number"
              value={config.traffic_multiplier}
              onChange={(e) => setConfig({ ...config, traffic_multiplier: parseFloat(e.target.value) || 1.0 })}
              min={0.1}
              max={10}
              step={0.1}
            />
            <p className="text-xs text-muted-foreground">
              使用 1GB 流量实际扣除 {(config.traffic_multiplier).toFixed(1)} GB
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
