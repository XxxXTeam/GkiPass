"use client"

import { useEffect, useState } from "react"
import {
  Plus, MoreHorizontal, Pencil, Trash2, KeyRound, ShieldCheck, ShieldBan, Search,
} from "lucide-react"
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
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import { toast } from "sonner"
import { userApi } from "@/lib/api/users"
import type { User } from "@/lib/types"
import { formatBytes } from "@/lib/utils"
import { exportCsv } from "@/lib/export-csv"

const roleMap: Record<string, string> = { admin: "管理员", user: "用户", agent: "代理" }
const statusMap: Record<string, { label: string; variant: "default" | "secondary" | "destructive" }> = {
  active: { label: "正常", variant: "default" },
  disabled: { label: "禁用", variant: "secondary" },
  banned: { label: "封禁", variant: "destructive" },
}

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [pwDialogOpen, setPwDialogOpen] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [pwUserId, setPwUserId] = useState<string | null>(null)
  const [newPassword, setNewPassword] = useState("")
  const [form, setForm] = useState({ username: "", email: "", role: "user", password: "" })
  const [submitting, setSubmitting] = useState(false)
  const [search, setSearch] = useState("")

  const fetchUsers = async () => {
    try {
      const res = await userApi.list()
      if (res.success && res.data) setUsers(res.data)
    } catch { setUsers([]) } finally { setLoading(false) }
  }

  useEffect(() => { fetchUsers() }, [])

  const handleCreate = () => {
    setEditingId(null)
    setForm({ username: "", email: "", role: "user", password: "" })
    setDialogOpen(true)
  }

  const handleEdit = (user: User) => {
    setEditingId(user.id)
    setForm({ username: user.username, email: user.email, role: user.role, password: "" })
    setDialogOpen(true)
  }

  const handleSubmit = async () => {
    if (!form.username) { toast.error("请填写用户名"); return }
    if (!editingId && !form.password) { toast.error("请设置密码"); return }
    setSubmitting(true)
    try {
      if (editingId) {
        await userApi.updateRole(editingId, form.role)
        toast.success("用户角色已更新")
      } else {
        await userApi.create({ username: form.username, email: form.email, password: form.password })
        toast.success("用户已创建")
      }
      setDialogOpen(false)
      fetchUsers()
    } catch { toast.error("操作失败") } finally { setSubmitting(false) }
  }

  const handleDelete = async () => {
    if (!deletingId) return
    try {
      await userApi.delete(deletingId)
      toast.success("用户已删除")
      setDeleteDialogOpen(false)
      fetchUsers()
    } catch { toast.error("删除失败") }
  }

  const handleResetPassword = async () => {
    if (!pwUserId || !newPassword) { toast.error("请输入新密码"); return }
    try {
      /* 暂无后端独立的管理员重置密码 API，提示管理员 */
      toast.info("管理员重置密码功能待后端实现")
      setPwDialogOpen(false)
      setNewPassword("")
    } catch { toast.error("重置失败") }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">用户管理</h1>
          <p className="text-muted-foreground">管理系统用户和权限</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={() => {
            exportCsv(users, [
              { header: "用户名", accessor: (u) => u.username },
              { header: "邮箱", accessor: (u) => u.email || "" },
              { header: "角色", accessor: (u) => roleMap[u.role] || u.role },
              { header: "状态", accessor: (u) => statusMap[u.status]?.label || u.status },
              { header: "流量已用", accessor: (u) => formatBytes(u.traffic_used || 0) },
            ], "用户列表")
          }}>
            导出 CSV
          </Button>
          <Button onClick={handleCreate}>
            <Plus className="mr-2 h-4 w-4" /> 创建用户
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-base">用户列表</CardTitle>
          <div className="relative w-64">
            <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="搜索用户名..."
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
          ) : users.filter((u) => !search || u.username.toLowerCase().includes(search.toLowerCase())).length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12">
              <p className="text-muted-foreground">暂无用户</p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>用户名</TableHead>
                  <TableHead>邮箱</TableHead>
                  <TableHead>角色</TableHead>
                  <TableHead>流量</TableHead>
                  <TableHead>隧道</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead className="w-12" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {users.filter((u) => !search || u.username.toLowerCase().includes(search.toLowerCase())).map((user) => {
                  const status = statusMap[user.status] || statusMap.active
                  return (
                    <TableRow key={user.id}>
                      <TableCell className="font-medium">{user.username}</TableCell>
                      <TableCell className="text-sm text-muted-foreground">{user.email || "-"}</TableCell>
                      <TableCell>
                        <Badge variant="outline" className="text-xs">
                          {roleMap[user.role] || user.role}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-sm">
                        {formatBytes(user.traffic_used || 0)}
                        {user.traffic_limit > 0 && (
                          <span className="text-muted-foreground"> / {formatBytes(user.traffic_limit)}</span>
                        )}
                      </TableCell>
                      <TableCell className="text-sm">
                        {user.tunnel_count || 0}
                        {user.tunnel_limit > 0 && (
                          <span className="text-muted-foreground"> / {user.tunnel_limit}</span>
                        )}
                      </TableCell>
                      <TableCell><Badge variant={status.variant}>{status.label}</Badge></TableCell>
                      <TableCell>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="icon" className="h-8 w-8">
                              <MoreHorizontal className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem onClick={() => handleEdit(user)}>
                              <Pencil className="mr-2 h-4 w-4" /> 编辑
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => { setPwUserId(user.id); setPwDialogOpen(true) }}>
                              <KeyRound className="mr-2 h-4 w-4" /> 重置密码
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={async () => {
                              try {
                                await userApi.toggleStatus(user.id)
                                toast.success(user.status === "active" ? "已禁用" : "已启用")
                                fetchUsers()
                              } catch { toast.error("操作失败") }
                            }}>
                              {user.status === "active" ? (
                                <><ShieldBan className="mr-2 h-4 w-4" /> 禁用</>
                              ) : (
                                <><ShieldCheck className="mr-2 h-4 w-4" /> 启用</>
                              )}
                            </DropdownMenuItem>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem className="text-destructive" onClick={() => { setDeletingId(user.id); setDeleteDialogOpen(true) }}>
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
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{editingId ? "编辑用户" : "创建用户"}</DialogTitle>
            <DialogDescription>{editingId ? "修改用户信息" : "创建新的系统用户"}</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-2">
              <Label>用户名 *</Label>
              <Input value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} placeholder="用户名" />
            </div>
            <div className="grid gap-2">
              <Label>邮箱</Label>
              <Input type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} placeholder="user@example.com" />
            </div>
            <div className="grid gap-2">
              <Label>角色</Label>
              <Select value={form.role} onValueChange={(v) => setForm({ ...form, role: v })}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="user">普通用户</SelectItem>
                  <SelectItem value="agent">代理</SelectItem>
                  <SelectItem value="admin">管理员</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {!editingId && (
              <div className="grid gap-2">
                <Label>密码 *</Label>
                <Input type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} placeholder="设置密码" />
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>取消</Button>
            <Button onClick={handleSubmit} disabled={submitting}>
              {submitting ? "处理中..." : editingId ? "保存" : "创建"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 重置密码弹窗 */}
      <Dialog open={pwDialogOpen} onOpenChange={setPwDialogOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>重置密码</DialogTitle>
            <DialogDescription>为用户设置新密码</DialogDescription>
          </DialogHeader>
          <div className="grid gap-2">
            <Label>新密码</Label>
            <Input type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} placeholder="输入新密码" />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setPwDialogOpen(false)}>取消</Button>
            <Button onClick={handleResetPassword}>确认重置</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 删除确认 */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>删除用户将同时删除其所有隧道和数据。此操作不可撤销。</DialogDescription>
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
