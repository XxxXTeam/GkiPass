"use client"

import { useEffect, useState } from "react"
import { useSearchParams } from "next/navigation"
import {
  Plus,
  MoreHorizontal,
  Pencil,
  Trash2,
  Power,
  PowerOff,
  Lock,
  Unlock,
  ArrowDownRight,
  ArrowUpRight,
  Search,
} from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { toast } from "sonner"
import { Checkbox } from "@/components/ui/checkbox"
import { tunnelApi } from "@/lib/api/tunnels"
import { nodeGroupApi } from "@/lib/api/nodes"
import type { Tunnel, CreateTunnelRequest, NodeGroup } from "@/lib/types"
import { formatBytes } from "@/lib/utils"
import { exportCsv } from "@/lib/export-csv"

const defaultForm: CreateTunnelRequest = {
  name: "",
  description: "",
  protocol: "tcp",
  listen_port: 0,
  target_address: "",
  target_port: 0,
  enable_encryption: false,
  encryption_method: "aes-256-gcm",
  max_connections: 0,
  rate_limit_bps: 0,
  idle_timeout: 300,
}

export default function TunnelsPage() {
  const searchParams = useSearchParams()
  const [tunnels, setTunnels] = useState<Tunnel[]>([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [form, setForm] = useState<CreateTunnelRequest>(defaultForm)
  const [submitting, setSubmitting] = useState(false)
  const [nodeGroups, setNodeGroups] = useState<NodeGroup[]>([])
  const [search, setSearch] = useState("")
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())

  const fetchTunnels = async () => {
    try {
      const res = await tunnelApi.list()
      if (res.success && res.data) {
        setTunnels(res.data)
      }
    } catch {
      setTunnels([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchTunnels()
    nodeGroupApi.list().then((res) => {
      if (res.success && res.data) setNodeGroups(res.data)
    }).catch(() => {})
  }, [])

  /* 响应 URL 参数 ?action=create 自动打开创建弹窗 */
  useEffect(() => {
    if (searchParams.get("action") === "create") {
      handleCreate()
    }
  }, [searchParams])

  const handleCreate = () => {
    setEditingId(null)
    setForm(defaultForm)
    setDialogOpen(true)
  }

  const handleEdit = (tunnel: Tunnel) => {
    setEditingId(tunnel.id)
    setForm({
      name: tunnel.name,
      description: tunnel.description,
      protocol: tunnel.protocol,
      listen_port: tunnel.listen_port,
      target_address: tunnel.target_address,
      target_port: tunnel.target_port,
      enable_encryption: tunnel.enable_encryption,
      encryption_method: tunnel.encryption_method,
      max_connections: tunnel.max_connections,
      rate_limit_bps: tunnel.rate_limit_bps,
      idle_timeout: tunnel.idle_timeout,
      ingress_group_id: tunnel.ingress_group_id,
      egress_group_id: tunnel.egress_group_id,
    })
    setDialogOpen(true)
  }

  const handleSubmit = async () => {
    if (!form.name || !form.listen_port || !form.target_address || !form.target_port) {
      toast.error("请填写必要字段")
      return
    }
    setSubmitting(true)
    try {
      if (editingId) {
        await tunnelApi.update(editingId, form)
        toast.success("隧道已更新")
      } else {
        await tunnelApi.create(form)
        toast.success("隧道已创建")
      }
      setDialogOpen(false)
      fetchTunnels()
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "操作失败"
      toast.error(message)
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async () => {
    if (!deletingId) return
    try {
      await tunnelApi.delete(deletingId)
      toast.success("隧道已删除")
      setDeleteDialogOpen(false)
      fetchTunnels()
    } catch {
      toast.error("删除失败")
    }
  }

  const handleToggle = async (id: string, enabled: boolean) => {
    try {
      await tunnelApi.toggle(id, !enabled)
      toast.success(enabled ? "隧道已禁用" : "隧道已启用")
      fetchTunnels()
    } catch {
      toast.error("操作失败")
    }
  }

  /* 批量操作 */
  const handleBatchToggle = async (enable: boolean) => {
    const ids = Array.from(selectedIds)
    if (ids.length === 0) return
    try {
      await Promise.all(ids.map((id) => tunnelApi.toggle(id, enable)))
      toast.success(`已${enable ? "启用" : "禁用"} ${ids.length} 条隧道`)
      setSelectedIds(new Set())
      fetchTunnels()
    } catch { toast.error("批量操作失败") }
  }

  const handleBatchDelete = async () => {
    const ids = Array.from(selectedIds)
    if (ids.length === 0) return
    try {
      await Promise.all(ids.map((id) => tunnelApi.delete(id)))
      toast.success(`已删除 ${ids.length} 条隧道`)
      setSelectedIds(new Set())
      fetchTunnels()
    } catch { toast.error("批量删除失败") }
  }

  const toggleSelect = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id); else next.add(id)
      return next
    })
  }

  const toggleSelectAll = () => {
    const filtered = tunnels.filter((t) => !search || t.name.toLowerCase().includes(search.toLowerCase()))
    if (selectedIds.size === filtered.length) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(filtered.map((t) => t.id)))
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">隧道管理</h1>
          <p className="text-muted-foreground">管理转发隧道和规则</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={() => {
            exportCsv(tunnels, [
              { header: "名称", accessor: (t) => t.name },
              { header: "协议", accessor: (t) => t.protocol },
              { header: "监听端口", accessor: (t) => t.listen_port },
              { header: "目标地址", accessor: (t) => t.target_address },
              { header: "目标端口", accessor: (t) => t.target_port },
              { header: "状态", accessor: (t) => t.enabled ? "启用" : "禁用" },
              { header: "加密", accessor: (t) => t.enable_encryption ? "是" : "否" },
            ], "隧道列表")
          }}>
            导出 CSV
          </Button>
          <Button onClick={handleCreate}>
            <Plus className="mr-2 h-4 w-4" />
            创建隧道
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div className="flex items-center gap-3">
            <CardTitle className="text-base">隧道列表</CardTitle>
            {selectedIds.size > 0 && (
              <div className="flex items-center gap-2 text-xs">
                <span className="text-muted-foreground">已选 {selectedIds.size} 项</span>
                <Button variant="outline" size="sm" className="h-7 text-xs" onClick={() => handleBatchToggle(true)}>
                  <Power className="mr-1 h-3 w-3" /> 批量启用
                </Button>
                <Button variant="outline" size="sm" className="h-7 text-xs" onClick={() => handleBatchToggle(false)}>
                  <PowerOff className="mr-1 h-3 w-3" /> 批量禁用
                </Button>
                <Button variant="destructive" size="sm" className="h-7 text-xs" onClick={handleBatchDelete}>
                  <Trash2 className="mr-1 h-3 w-3" /> 批量删除
                </Button>
              </div>
            )}
          </div>
          <div className="relative w-64">
            <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="搜索隧道名称..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-8 h-8 text-sm"
            />
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="h-12 animate-pulse rounded bg-muted" />
              ))}
            </div>
          ) : tunnels.filter((t) => !search || t.name.toLowerCase().includes(search.toLowerCase())).length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <p className="text-muted-foreground">暂无隧道</p>
              <Button variant="outline" className="mt-4" onClick={handleCreate}>
                <Plus className="mr-2 h-4 w-4" />
                创建第一个隧道
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-10">
                    <Checkbox
                      checked={selectedIds.size > 0 && selectedIds.size === tunnels.filter((t) => !search || t.name.toLowerCase().includes(search.toLowerCase())).length}
                      onCheckedChange={toggleSelectAll}
                    />
                  </TableHead>
                  <TableHead>名称</TableHead>
                  <TableHead>协议</TableHead>
                  <TableHead>监听端口</TableHead>
                  <TableHead>目标</TableHead>
                  <TableHead>加密</TableHead>
                  <TableHead>流量</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead className="w-12" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {tunnels.filter((t) => !search || t.name.toLowerCase().includes(search.toLowerCase())).map((tunnel) => (
                  <TableRow key={tunnel.id}>
                    <TableCell>
                      <Checkbox
                        checked={selectedIds.has(tunnel.id)}
                        onCheckedChange={() => toggleSelect(tunnel.id)}
                      />
                    </TableCell>
                    <TableCell>
                      <div>
                        <button className="font-medium text-left hover:underline" onClick={() => handleEdit(tunnel)}>{tunnel.name}</button>
                        {tunnel.description && (
                          <p className="text-xs text-muted-foreground truncate max-w-[200px]">
                            {tunnel.description}
                          </p>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline" className="uppercase text-xs">
                        {tunnel.protocol}
                      </Badge>
                    </TableCell>
                    <TableCell className="font-mono text-sm">
                      {tunnel.listen_port}
                    </TableCell>
                    <TableCell className="font-mono text-sm">
                      {tunnel.target_address}:{tunnel.target_port}
                    </TableCell>
                    <TableCell>
                      {tunnel.enable_encryption ? (
                        <Lock className="h-4 w-4 text-green-500" />
                      ) : (
                        <Unlock className="h-4 w-4 text-muted-foreground" />
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2 text-xs">
                        <ArrowDownRight className="h-3 w-3 text-green-500" />
                        {formatBytes(tunnel.bytes_in)}
                        <ArrowUpRight className="h-3 w-3 text-blue-500" />
                        {formatBytes(tunnel.bytes_out)}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={tunnel.enabled ? "default" : "secondary"}>
                        {tunnel.enabled ? "启用" : "禁用"}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon" className="h-8 w-8">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => handleEdit(tunnel)}>
                            <Pencil className="mr-2 h-4 w-4" />
                            编辑
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={() => handleToggle(tunnel.id, tunnel.enabled)}>
                            {tunnel.enabled ? (
                              <>
                                <PowerOff className="mr-2 h-4 w-4" />
                                禁用
                              </>
                            ) : (
                              <>
                                <Power className="mr-2 h-4 w-4" />
                                启用
                              </>
                            )}
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            className="text-destructive"
                            onClick={() => {
                              setDeletingId(tunnel.id)
                              setDeleteDialogOpen(true)
                            }}
                          >
                            <Trash2 className="mr-2 h-4 w-4" />
                            删除
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* 创建/编辑弹窗 */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{editingId ? "编辑隧道" : "创建隧道"}</DialogTitle>
            <DialogDescription>
              {editingId ? "修改隧道配置" : "配置新的转发隧道"}
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-2">
            <div className="grid gap-2">
              <Label>名称 *</Label>
              <Input
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="隧道名称"
              />
            </div>

            <div className="grid gap-2">
              <Label>描述</Label>
              <Textarea
                value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
                placeholder="可选描述"
                rows={2}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label>协议 *</Label>
                <Select
                  value={form.protocol}
                  onValueChange={(v) => setForm({ ...form, protocol: v })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="tcp">TCP</SelectItem>
                    <SelectItem value="udp">UDP</SelectItem>
                    <SelectItem value="ws">WebSocket</SelectItem>
                    <SelectItem value="wss">WSS</SelectItem>
                    <SelectItem value="tls">TLS</SelectItem>
                    <SelectItem value="tls-mux">TLS-Mux</SelectItem>
                    <SelectItem value="kcp">KCP</SelectItem>
                    <SelectItem value="quic">QUIC</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label>监听端口 *</Label>
                <Input
                  type="number"
                  value={form.listen_port || ""}
                  onChange={(e) =>
                    setForm({ ...form, listen_port: parseInt(e.target.value) || 0 })
                  }
                  placeholder="1-65535"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label>目标地址 *</Label>
                <Input
                  value={form.target_address}
                  onChange={(e) => setForm({ ...form, target_address: e.target.value })}
                  placeholder="127.0.0.1"
                />
              </div>
              <div className="grid gap-2">
                <Label>目标端口 *</Label>
                <Input
                  type="number"
                  value={form.target_port || ""}
                  onChange={(e) =>
                    setForm({ ...form, target_port: parseInt(e.target.value) || 0 })
                  }
                  placeholder="1-65535"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label>最大连接数</Label>
                <Input
                  type="number"
                  value={form.max_connections || ""}
                  onChange={(e) =>
                    setForm({ ...form, max_connections: parseInt(e.target.value) || 0 })
                  }
                  placeholder="0=无限制"
                />
              </div>
              <div className="grid gap-2">
                <Label>空闲超时(秒)</Label>
                <Input
                  type="number"
                  value={form.idle_timeout || ""}
                  onChange={(e) =>
                    setForm({ ...form, idle_timeout: parseInt(e.target.value) || 300 })
                  }
                  placeholder="300"
                />
              </div>
            </div>

            <div className="flex items-center justify-between rounded-lg border p-3">
              <div>
                <p className="text-sm font-medium">启用加密</p>
                <p className="text-xs text-muted-foreground">对隧道数据进行端到端加密</p>
              </div>
              <Switch
                checked={form.enable_encryption}
                onCheckedChange={(v) => setForm({ ...form, enable_encryption: v })}
              />
            </div>

            {form.enable_encryption && (
              <div className="grid gap-2">
                <Label>加密算法</Label>
                <Select
                  value={form.encryption_method}
                  onValueChange={(v) => setForm({ ...form, encryption_method: v })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="aes-256-gcm">AES-256-GCM</SelectItem>
                    <SelectItem value="aes-192-gcm">AES-192-GCM</SelectItem>
                    <SelectItem value="aes-128-gcm">AES-128-GCM</SelectItem>
                    <SelectItem value="chacha20-poly1305">ChaCha20-Poly1305</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            )}

            {/* 节点组选择 */}
            {nodeGroups.length > 0 && (
              <div className="grid grid-cols-2 gap-4">
                <div className="grid gap-2">
                  <Label>入口节点组</Label>
                  <Select
                    value={form.ingress_group_id || "none"}
                    onValueChange={(v) => setForm({ ...form, ingress_group_id: v === "none" ? undefined : v })}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="选择入口节点组" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="none">自动分配</SelectItem>
                      {nodeGroups.map((g) => (
                        <SelectItem key={g.id} value={g.id}>{g.name}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="grid gap-2">
                  <Label>出口节点组</Label>
                  <Select
                    value={form.egress_group_id || "none"}
                    onValueChange={(v) => setForm({ ...form, egress_group_id: v === "none" ? undefined : v })}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="选择出口节点组" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="none">自动分配</SelectItem>
                      {nodeGroups.map((g) => (
                        <SelectItem key={g.id} value={g.id}>{g.name}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleSubmit} disabled={submitting}>
              {submitting ? "处理中..." : editingId ? "保存" : "创建"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 删除确认弹窗 */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>
              删除隧道将同时删除其关联的所有规则。此操作不可撤销。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteDialogOpen(false)}>
              取消
            </Button>
            <Button variant="destructive" onClick={handleDelete}>
              确认删除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
