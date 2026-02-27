"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import Link from "next/link"
import { Plus, MoreHorizontal, Pencil, Trash2, Server, Settings } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem,
  DropdownMenuSeparator, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Dialog, DialogContent, DialogDescription, DialogFooter,
  DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import { toast } from "sonner"
import { nodeGroupApi } from "@/lib/api/nodes"
import type { NodeGroup } from "@/lib/types"

export default function NodeGroupsPage() {
  const router = useRouter()
  const [groups, setGroups] = useState<NodeGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [form, setForm] = useState({ name: "", description: "", type: "entry", region: "" })
  const [submitting, setSubmitting] = useState(false)

  const fetchGroups = async () => {
    try {
      const res = await nodeGroupApi.list()
      if (res.success && res.data) setGroups(res.data)
    } catch { setGroups([]) } finally { setLoading(false) }
  }

  useEffect(() => { fetchGroups() }, [])

  const handleCreate = () => {
    setEditingId(null)
    setForm({ name: "", description: "", type: "entry", region: "" })
    setDialogOpen(true)
  }

  const handleEdit = (group: NodeGroup) => {
    setEditingId(group.id)
    setForm({ name: group.name, description: group.description, type: group.type, region: group.region })
    setDialogOpen(true)
  }

  const handleSubmit = async () => {
    if (!form.name) { toast.error("请填写组名称"); return }
    setSubmitting(true)
    try {
      if (editingId) {
        await nodeGroupApi.update(editingId, form)
        toast.success("节点组已更新")
      } else {
        await nodeGroupApi.create(form)
        toast.success("节点组已创建")
      }
      setDialogOpen(false)
      fetchGroups()
    } catch { toast.error("操作失败") } finally { setSubmitting(false) }
  }

  const handleDelete = async () => {
    if (!deletingId) return
    try {
      await nodeGroupApi.delete(deletingId)
      toast.success("节点组已删除")
      setDeleteDialogOpen(false)
      fetchGroups()
    } catch { toast.error("删除失败") }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">节点组管理</h1>
          <p className="text-muted-foreground">管理节点分组，用于隧道入口和出口配置</p>
        </div>
        <Button onClick={handleCreate}>
          <Plus className="mr-2 h-4 w-4" />
          创建节点组
        </Button>
      </div>

      <Card>
        <CardHeader><CardTitle className="text-base">节点组列表</CardTitle></CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="h-12 animate-pulse rounded bg-muted" />
              ))}
            </div>
          ) : groups.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <p className="text-muted-foreground">暂无节点组</p>
              <Button variant="outline" className="mt-4" onClick={handleCreate}>
                <Plus className="mr-2 h-4 w-4" />
                创建第一个节点组
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>名称</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>地区</TableHead>
                  <TableHead>节点数</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead className="w-12" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {groups.map((group) => (
                  <TableRow key={group.id}>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <Server className="h-4 w-4 text-muted-foreground" />
                        <div>
                          <Link href={`/dashboard/node-groups/config?id=${group.id}`} className="font-medium hover:underline">{group.name}</Link>
                          {group.description && (
                            <p className="text-xs text-muted-foreground truncate max-w-[200px]">{group.description}</p>
                          )}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell><Badge variant="outline" className="text-xs">{group.type}</Badge></TableCell>
                    <TableCell>{group.region || "-"}</TableCell>
                    <TableCell>{group.node_count || 0}</TableCell>
                    <TableCell>
                      <Badge variant={group.enabled ? "default" : "secondary"}>
                        {group.enabled ? "启用" : "禁用"}
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
                          <DropdownMenuItem onClick={() => handleEdit(group)}>
                            <Pencil className="mr-2 h-4 w-4" /> 编辑
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={() => router.push(`/dashboard/node-groups/config?id=${group.id}`)}>
                            <Settings className="mr-2 h-4 w-4" /> 配置
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem className="text-destructive" onClick={() => { setDeletingId(group.id); setDeleteDialogOpen(true) }}>
                            <Trash2 className="mr-2 h-4 w-4" /> 删除
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

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{editingId ? "编辑节点组" : "创建节点组"}</DialogTitle>
            <DialogDescription>{editingId ? "修改节点组配置" : "创建新的节点分组"}</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-2">
              <Label>名称 *</Label>
              <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="节点组名称" />
            </div>
            <div className="grid gap-2">
              <Label>描述</Label>
              <Textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} placeholder="可选描述" rows={2} />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label>类型</Label>
                <Select value={form.type} onValueChange={(v) => setForm({ ...form, type: v })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="entry">入口组</SelectItem>
                    <SelectItem value="exit">出口组</SelectItem>
                    <SelectItem value="relay">中继组</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label>地区</Label>
                <Input value={form.region} onChange={(e) => setForm({ ...form, region: e.target.value })} placeholder="如：cn-sh" />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>取消</Button>
            <Button onClick={handleSubmit} disabled={submitting}>
              {submitting ? "处理中..." : editingId ? "保存" : "创建"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>删除节点组将移除组内所有节点的关联关系。此操作不可撤销。</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteDialogOpen(false)}>取消</Button>
            <Button variant="destructive" onClick={handleDelete}>确认删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
