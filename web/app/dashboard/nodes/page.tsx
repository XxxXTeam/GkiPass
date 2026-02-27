"use client"

import { useEffect, useState } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import Link from "next/link"
import {
  Plus,
  MoreHorizontal,
  Pencil,
  Trash2,
  Wifi,
  WifiOff,
  Cpu,
  HardDrive,
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
import { Textarea } from "@/components/ui/textarea"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { toast } from "sonner"
import { nodeApi } from "@/lib/api/nodes"
import type { Node } from "@/lib/types"
import { exportCsv } from "@/lib/export-csv"

const statusMap: Record<string, { label: string; variant: "default" | "secondary" | "destructive" }> = {
  online: { label: "在线", variant: "default" },
  offline: { label: "离线", variant: "secondary" },
  maintenance: { label: "维护", variant: "destructive" },
}

export default function NodesPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [nodes, setNodes] = useState<Node[]>([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [form, setForm] = useState({
    name: "",
    description: "",
    type: "entry" as const,
    ip: "",
    port: 0,
    region: "",
    provider: "",
  })
  const [submitting, setSubmitting] = useState(false)
  const [search, setSearch] = useState("")

  const fetchNodes = async () => {
    try {
      const res = await nodeApi.list()
      if (res.success && res.data) {
        setNodes(res.data)
      }
    } catch {
      setNodes([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchNodes()
  }, [])

  /* 响应 URL 参数 ?action=create 自动打开创建弹窗 */
  useEffect(() => {
    if (searchParams.get("action") === "create") {
      handleCreate()
    }
  }, [searchParams])

  const handleCreate = () => {
    setEditingId(null)
    setForm({ name: "", description: "", type: "entry", ip: "", port: 0, region: "", provider: "" })
    setDialogOpen(true)
  }

  const handleEdit = (node: Node) => {
    setEditingId(node.id)
    setForm({
      name: node.name,
      description: node.description,
      type: node.type as "entry",
      ip: node.ip,
      port: node.port,
      region: node.region,
      provider: node.provider,
    })
    setDialogOpen(true)
  }

  const handleSubmit = async () => {
    if (!form.name || !form.ip) {
      toast.error("请填写必要字段")
      return
    }
    setSubmitting(true)
    try {
      if (editingId) {
        await nodeApi.update(editingId, form)
        toast.success("节点已更新")
      } else {
        await nodeApi.create(form)
        toast.success("节点已创建")
      }
      setDialogOpen(false)
      fetchNodes()
    } catch {
      toast.error("操作失败")
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async () => {
    if (!deletingId) return
    try {
      await nodeApi.delete(deletingId)
      toast.success("节点已删除")
      setDeleteDialogOpen(false)
      fetchNodes()
    } catch {
      toast.error("删除失败")
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">节点管理</h1>
          <p className="text-muted-foreground">管理转发节点和服务器</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={() => {
            exportCsv(nodes, [
              { header: "名称", accessor: (n) => n.name },
              { header: "类型", accessor: (n) => n.type },
              { header: "IP", accessor: (n) => n.ip },
              { header: "端口", accessor: (n) => n.port },
              { header: "地区", accessor: (n) => n.region || "" },
              { header: "状态", accessor: (n) => n.status },
              { header: "连接数", accessor: (n) => n.connection_count || 0 },
            ], "节点列表")
          }}>
            导出 CSV
          </Button>
          <Button onClick={handleCreate}>
            <Plus className="mr-2 h-4 w-4" />
            添加节点
          </Button>
        </div>
      </div>

      {/* 节点概览卡片 */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <div className="rounded-lg bg-green-500/10 p-2">
              <Wifi className="h-4 w-4 text-green-500" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">在线节点</p>
              <p className="text-xl font-bold">
                {nodes.filter((n) => n.status === "online").length}
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <div className="rounded-lg bg-muted p-2">
              <WifiOff className="h-4 w-4 text-muted-foreground" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">离线节点</p>
              <p className="text-xl font-bold">
                {nodes.filter((n) => n.status === "offline").length}
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <div className="rounded-lg bg-blue-500/10 p-2">
              <HardDrive className="h-4 w-4 text-blue-500" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">节点总数</p>
              <p className="text-xl font-bold">{nodes.length}</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 节点列表 */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-base">节点列表</CardTitle>
          <div className="relative w-64">
            <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="搜索节点名称..."
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
          ) : nodes.filter((n) => !search || n.name.toLowerCase().includes(search.toLowerCase())).length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <p className="text-muted-foreground">暂无节点</p>
              <Button variant="outline" className="mt-4" onClick={handleCreate}>
                <Plus className="mr-2 h-4 w-4" />
                添加第一个节点
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>名称</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>地址</TableHead>
                  <TableHead>地区</TableHead>
                  <TableHead>资源</TableHead>
                  <TableHead>连接数</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead className="w-12" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {nodes.filter((n) => !search || n.name.toLowerCase().includes(search.toLowerCase())).map((node) => {
                  const status = statusMap[node.status] || statusMap.offline
                  return (
                    <TableRow key={node.id}>
                      <TableCell>
                        <div>
                          <Link href={`/dashboard/nodes/detail?id=${node.id}`} className="font-medium hover:underline">{node.name}</Link>
                          <p className="text-xs text-muted-foreground">{node.version}</p>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className="text-xs">
                          {node.type}
                        </Badge>
                      </TableCell>
                      <TableCell className="font-mono text-sm">
                        {node.ip}:{node.port}
                      </TableCell>
                      <TableCell>{node.region || "-"}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2 text-xs">
                          <Cpu className="h-3 w-3" />
                          {node.cpu_usage?.toFixed(0) || 0}%
                          <HardDrive className="h-3 w-3 ml-1" />
                          {node.memory_usage?.toFixed(0) || 0}%
                        </div>
                      </TableCell>
                      <TableCell>{node.connection_count || 0}</TableCell>
                      <TableCell>
                        <Badge variant={status.variant}>{status.label}</Badge>
                      </TableCell>
                      <TableCell>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="icon" className="h-8 w-8">
                              <MoreHorizontal className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem onClick={() => router.push(`/dashboard/nodes/detail?id=${node.id}`)}>
                              <Cpu className="mr-2 h-4 w-4" />
                              查看详情
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => handleEdit(node)}>
                              <Pencil className="mr-2 h-4 w-4" />
                              编辑
                            </DropdownMenuItem>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                              className="text-destructive"
                              onClick={() => {
                                setDeletingId(node.id)
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
                  )
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* 创建/编辑弹窗 */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{editingId ? "编辑节点" : "添加节点"}</DialogTitle>
            <DialogDescription>
              {editingId ? "修改节点配置" : "添加新的转发节点"}
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-2">
            <div className="grid gap-2">
              <Label>名称 *</Label>
              <Input
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="节点名称"
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
                <Label>类型</Label>
                <Select
                  value={form.type}
                  onValueChange={(v) => setForm({ ...form, type: v as "entry" })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="entry">入口节点</SelectItem>
                    <SelectItem value="exit">出口节点</SelectItem>
                    <SelectItem value="relay">中继节点</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label>地区</Label>
                <Input
                  value={form.region}
                  onChange={(e) => setForm({ ...form, region: e.target.value })}
                  placeholder="如：cn-sh"
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label>IP 地址 *</Label>
                <Input
                  value={form.ip}
                  onChange={(e) => setForm({ ...form, ip: e.target.value })}
                  placeholder="192.168.1.1"
                />
              </div>
              <div className="grid gap-2">
                <Label>端口</Label>
                <Input
                  type="number"
                  value={form.port || ""}
                  onChange={(e) => setForm({ ...form, port: parseInt(e.target.value) || 0 })}
                  placeholder="8080"
                />
              </div>
            </div>
            <div className="grid gap-2">
              <Label>提供商</Label>
              <Input
                value={form.provider}
                onChange={(e) => setForm({ ...form, provider: e.target.value })}
                placeholder="如：Alibaba Cloud"
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleSubmit} disabled={submitting}>
              {submitting ? "处理中..." : editingId ? "保存" : "添加"}
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
              删除节点将断开其所有连接并移除相关规则。此操作不可撤销。
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
