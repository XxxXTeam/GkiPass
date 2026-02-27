"use client"

import { useEffect, useState } from "react"
import { Plus, MoreHorizontal, Pencil, Trash2 } from "lucide-react"
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
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { toast } from "sonner"
import { planApi } from "@/lib/api/plans"
import type { Plan } from "@/lib/types"
import { formatBytes } from "@/lib/utils"

const defaultForm = {
  name: "", description: "", price: 0, duration: 1, duration_unit: "month" as const,
  traffic_limit: 0, rule_limit: 10, speed_limit: 0, connection_limit: 0, enabled: true,
}

export default function PlansPage() {
  const [plans, setPlans] = useState<Plan[]>([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [form, setForm] = useState(defaultForm)
  const [submitting, setSubmitting] = useState(false)

  const fetchPlans = async () => {
    try {
      const res = await planApi.list()
      if (res.success && res.data) setPlans(res.data)
    } catch { setPlans([]) } finally { setLoading(false) }
  }

  useEffect(() => { fetchPlans() }, [])

  const handleCreate = () => {
    setEditingId(null)
    setForm(defaultForm)
    setDialogOpen(true)
  }

  const handleEdit = (plan: Plan) => {
    setEditingId(plan.id)
    setForm({
      name: plan.name, description: plan.description, price: plan.price,
      duration: plan.duration, duration_unit: plan.duration_unit || "month",
      traffic_limit: plan.traffic_limit, rule_limit: plan.rule_limit,
      speed_limit: plan.speed_limit, connection_limit: plan.connection_limit,
      enabled: plan.enabled,
    })
    setDialogOpen(true)
  }

  const handleSubmit = async () => {
    if (!form.name) { toast.error("请填写套餐名称"); return }
    setSubmitting(true)
    try {
      if (editingId) {
        await planApi.update(editingId, form)
        toast.success("套餐已更新")
      } else {
        await planApi.create(form)
        toast.success("套餐已创建")
      }
      setDialogOpen(false)
      fetchPlans()
    } catch { toast.error("操作失败") } finally { setSubmitting(false) }
  }

  const handleDelete = async () => {
    if (!deletingId) return
    try {
      await planApi.delete(deletingId)
      toast.success("套餐已删除")
      setDeleteDialogOpen(false)
      fetchPlans()
    } catch { toast.error("删除失败") }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">套餐管理</h1>
          <p className="text-muted-foreground">管理用户订阅套餐和资源限制</p>
        </div>
        <Button onClick={handleCreate}>
          <Plus className="mr-2 h-4 w-4" /> 创建套餐
        </Button>
      </div>

      <Card>
        <CardHeader><CardTitle className="text-base">套餐列表</CardTitle></CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="h-12 animate-pulse rounded bg-muted" />
              ))}
            </div>
          ) : plans.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12">
              <p className="text-muted-foreground">暂无套餐</p>
              <Button variant="outline" className="mt-4" onClick={handleCreate}>
                <Plus className="mr-2 h-4 w-4" /> 创建第一个套餐
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>名称</TableHead>
                  <TableHead>价格</TableHead>
                  <TableHead>时长</TableHead>
                  <TableHead>流量限制</TableHead>
                  <TableHead>隧道限制</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead className="w-12" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {plans.map((plan) => (
                  <TableRow key={plan.id}>
                    <TableCell>
                      <div>
                        <p className="font-medium">{plan.name}</p>
                        {plan.description && <p className="text-xs text-muted-foreground truncate max-w-[200px]">{plan.description}</p>}
                      </div>
                    </TableCell>
                    <TableCell className="font-medium">{plan.price > 0 ? `¥${plan.price}` : "免费"}</TableCell>
                    <TableCell>{plan.duration} {plan.duration_unit === "year" ? "年" : plan.duration_unit === "permanent" ? "永久" : "月"}</TableCell>
                    <TableCell>{formatBytes(plan.traffic_limit, "无限制")}</TableCell>
                    <TableCell>{plan.rule_limit > 0 ? plan.rule_limit : "无限制"}</TableCell>
                    <TableCell>
                      <Badge variant={plan.enabled ? "default" : "secondary"}>
                        {plan.enabled ? "启用" : "禁用"}
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
                          <DropdownMenuItem onClick={() => handleEdit(plan)}>
                            <Pencil className="mr-2 h-4 w-4" /> 编辑
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem className="text-destructive" onClick={() => { setDeletingId(plan.id); setDeleteDialogOpen(true) }}>
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
        <DialogContent className="max-w-lg max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{editingId ? "编辑套餐" : "创建套餐"}</DialogTitle>
            <DialogDescription>{editingId ? "修改套餐配置" : "创建新的订阅套餐"}</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-2">
              <Label>名称 *</Label>
              <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="套餐名称" />
            </div>
            <div className="grid gap-2">
              <Label>描述</Label>
              <Textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} rows={2} />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label>价格 (元)</Label>
                <Input type="number" value={form.price} onChange={(e) => setForm({ ...form, price: parseFloat(e.target.value) || 0 })} />
              </div>
              <div className="grid gap-2">
                <Label>时长</Label>
                <Input type="number" value={form.duration} onChange={(e) => setForm({ ...form, duration: parseInt(e.target.value) || 1 })} />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label>流量限制 (bytes, 0=无限)</Label>
                <Input type="number" value={form.traffic_limit} onChange={(e) => setForm({ ...form, traffic_limit: parseInt(e.target.value) || 0 })} />
              </div>
              <div className="grid gap-2">
                <Label>隧道数限制 (0=无限)</Label>
                <Input type="number" value={form.rule_limit} onChange={(e) => setForm({ ...form, rule_limit: parseInt(e.target.value) || 0 })} />
              </div>
            </div>
            <div className="flex items-center justify-between rounded-lg border p-3">
              <div>
                <p className="text-sm font-medium">启用套餐</p>
                <p className="text-xs text-muted-foreground">禁用后用户无法购买此套餐</p>
              </div>
              <Switch checked={form.enabled} onCheckedChange={(v) => setForm({ ...form, enabled: v })} />
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
            <DialogDescription>删除套餐后，已订阅此套餐的用户将保持现有权益直到到期。</DialogDescription>
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
