"use client"

import { useEffect, useState } from "react"
import {
  Plus, MoreHorizontal, Pencil, Trash2, ShieldCheck, Rocket, ToggleLeft, ToggleRight,
} from "lucide-react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
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
import { Checkbox } from "@/components/ui/checkbox"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { toast } from "sonner"
import { policyApi } from "@/lib/api/acl"
import type { Policy } from "@/lib/types"

/*
  系统支持的全部隧道协议，对齐后端 TunnelProtocol 枚举
*/
const ALL_PROTOCOLS = [
  { value: "tcp", label: "TCP", desc: "传输控制协议" },
  { value: "udp", label: "UDP", desc: "用户数据报协议" },
  { value: "ws", label: "WebSocket", desc: "WebSocket 协议" },
  { value: "wss", label: "WSS", desc: "WebSocket over TLS" },
  { value: "tls", label: "TLS", desc: "传输层安全协议" },
  { value: "tls-mux", label: "TLS-Mux", desc: "TLS 多路复用" },
  { value: "kcp", label: "KCP", desc: "快速可靠 UDP 协议" },
  { value: "quic", label: "QUIC", desc: "快速 UDP 互联网连接" },
]

interface FormState {
  name: string
  enabled: boolean
  protocols: string[]
  description: string
  priority: number
}

const defaultForm: FormState = {
  name: "", enabled: true, protocols: [], description: "", priority: 0,
}

export default function ProtocolPolicyPage() {
  const [policies, setPolicies] = useState<Policy[]>([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [form, setForm] = useState<FormState>(defaultForm)
  const [submitting, setSubmitting] = useState(false)

  const fetchPolicies = async () => {
    try {
      const res = await policyApi.list("protocol")
      if (res.success && res.data) {
        setPolicies(res.data.policies || [])
      }
    } catch { setPolicies([]) }
    finally { setLoading(false) }
  }

  useEffect(() => { fetchPolicies() }, [])

  const handleCreate = () => {
    setEditingId(null)
    setForm(defaultForm)
    setDialogOpen(true)
  }

  const handleEdit = (policy: Policy) => {
    setEditingId(policy.id)
    setForm({
      name: policy.name,
      enabled: policy.enabled,
      protocols: policy.config?.protocols || [],
      description: policy.description || "",
      priority: policy.priority || 0,
    })
    setDialogOpen(true)
  }

  const handleSubmit = async () => {
    if (!form.name) { toast.error("请填写策略名称"); return }
    if (form.protocols.length === 0) { toast.error("请至少选择一个允许的协议"); return }
    setSubmitting(true)
    try {
      if (editingId) {
        await policyApi.update(editingId, {
          name: form.name,
          enabled: form.enabled,
          config: { protocols: form.protocols },
          description: form.description,
          priority: form.priority,
        })
        toast.success("策略已更新")
      } else {
        await policyApi.create({
          name: form.name,
          type: "protocol",
          enabled: form.enabled,
          config: { protocols: form.protocols },
          description: form.description,
          priority: form.priority,
        })
        toast.success("策略已创建")
      }
      setDialogOpen(false)
      fetchPolicies()
    } catch { toast.error("操作失败") }
    finally { setSubmitting(false) }
  }

  const handleDelete = async () => {
    if (!deletingId) return
    try {
      await policyApi.delete(deletingId)
      toast.success("策略已删除")
      setDeleteDialogOpen(false)
      fetchPolicies()
    } catch { toast.error("删除失败") }
  }

  const handleToggle = async (policy: Policy) => {
    try {
      await policyApi.update(policy.id, { enabled: !policy.enabled })
      toast.success(policy.enabled ? "已禁用" : "已启用")
      fetchPolicies()
    } catch { toast.error("操作失败") }
  }

  const handleDeploy = async (id: string) => {
    try {
      await policyApi.deploy(id)
      toast.success("策略已下发到节点")
    } catch { toast.error("下发失败") }
  }

  const toggleProtocol = (proto: string) => {
    setForm((prev) => ({
      ...prev,
      protocols: prev.protocols.includes(proto)
        ? prev.protocols.filter((p) => p !== proto)
        : [...prev.protocols, proto],
    }))
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">协议转发限制</h1>
          <p className="text-muted-foreground">管理隧道允许使用的转发协议，未选中的协议将被禁止</p>
        </div>
        <Button onClick={handleCreate}>
          <Plus className="mr-2 h-4 w-4" /> 新建策略
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">策略列表</CardTitle>
          <CardDescription>每条策略定义一组允许转发的协议白名单</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-3">
              {Array.from({ length: 2 }).map((_, i) => (
                <div key={i} className="h-14 animate-pulse rounded bg-muted" />
              ))}
            </div>
          ) : policies.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12">
              <ShieldCheck className="h-10 w-10 text-muted-foreground mb-3" />
              <p className="text-muted-foreground">暂无协议限制策略</p>
              <p className="text-xs text-muted-foreground mt-1">未配置策略时，所有协议均允许转发</p>
              <Button variant="outline" className="mt-4" onClick={handleCreate}>
                <Plus className="mr-2 h-4 w-4" /> 创建第一条策略
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>策略名称</TableHead>
                  <TableHead>允许的协议</TableHead>
                  <TableHead>优先级</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead className="w-12" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {policies.map((policy) => {
                  const protocols = policy.config?.protocols || []
                  return (
                    <TableRow key={policy.id}>
                      <TableCell>
                        <div>
                          <p className="font-medium">{policy.name}</p>
                          {policy.description && (
                            <p className="text-xs text-muted-foreground truncate max-w-[200px]">{policy.description}</p>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1">
                          {protocols.length > 0 ? protocols.map((p) => (
                            <Badge key={p} variant="outline" className="text-xs uppercase">{p}</Badge>
                          )) : (
                            <span className="text-xs text-muted-foreground">无</span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>{policy.priority}</TableCell>
                      <TableCell>
                        <Badge variant={policy.enabled ? "default" : "secondary"}>
                          {policy.enabled ? "生效" : "禁用"}
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
                            <DropdownMenuItem onClick={() => handleEdit(policy)}>
                              <Pencil className="mr-2 h-4 w-4" /> 编辑
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => handleToggle(policy)}>
                              {policy.enabled
                                ? <><ToggleLeft className="mr-2 h-4 w-4" /> 禁用</>
                                : <><ToggleRight className="mr-2 h-4 w-4" /> 启用</>}
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => handleDeploy(policy.id)}>
                              <Rocket className="mr-2 h-4 w-4" /> 下发到节点
                            </DropdownMenuItem>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem className="text-destructive" onClick={() => { setDeletingId(policy.id); setDeleteDialogOpen(true) }}>
                              <Trash2 className="mr-2 h-4 w-4" /> 删除
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
            <DialogTitle>{editingId ? "编辑策略" : "新建策略"}</DialogTitle>
            <DialogDescription>
              选择允许隧道使用的转发协议，未勾选的协议将被禁止转发
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-2">
              <Label>策略名称 *</Label>
              <Input
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="如：仅允许 TCP/UDP"
              />
            </div>

            <div className="grid gap-2">
              <Label>允许的转发协议 *</Label>
              <div className="grid grid-cols-2 gap-2 rounded-lg border p-3">
                {ALL_PROTOCOLS.map((proto) => (
                  <label
                    key={proto.value}
                    className="flex items-center gap-2 rounded-md px-2 py-1.5 cursor-pointer hover:bg-accent transition-colors"
                  >
                    <Checkbox
                      checked={form.protocols.includes(proto.value)}
                      onCheckedChange={() => toggleProtocol(proto.value)}
                    />
                    <div>
                      <span className="text-sm font-medium">{proto.label}</span>
                      <span className="text-[10px] text-muted-foreground ml-1">({proto.desc})</span>
                    </div>
                  </label>
                ))}
              </div>
              <div className="flex gap-2">
                <Button
                  type="button" variant="ghost" size="sm" className="h-6 text-xs"
                  onClick={() => setForm({ ...form, protocols: ALL_PROTOCOLS.map((p) => p.value) })}
                >
                  全选
                </Button>
                <Button
                  type="button" variant="ghost" size="sm" className="h-6 text-xs"
                  onClick={() => setForm({ ...form, protocols: [] })}
                >
                  清空
                </Button>
                <Button
                  type="button" variant="ghost" size="sm" className="h-6 text-xs"
                  onClick={() => setForm({ ...form, protocols: ["tcp", "udp"] })}
                >
                  仅 TCP/UDP
                </Button>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label>优先级</Label>
                <Input
                  type="number"
                  value={form.priority}
                  onChange={(e) => setForm({ ...form, priority: parseInt(e.target.value) || 0 })}
                  placeholder="数字越小优先级越高"
                />
              </div>
              <div className="grid gap-2">
                <Label>启用</Label>
                <div className="flex items-center h-9">
                  <Switch checked={form.enabled} onCheckedChange={(v) => setForm({ ...form, enabled: v })} />
                  <span className="ml-2 text-sm text-muted-foreground">{form.enabled ? "立即生效" : "暂不生效"}</span>
                </div>
              </div>
            </div>

            <div className="grid gap-2">
              <Label>备注</Label>
              <Textarea
                value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
                placeholder="可选，描述此策略的用途"
                rows={2}
              />
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

      {/* 删除确认 */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>删除后该策略将立即失效，已下发的节点需要重新部署。</DialogDescription>
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
