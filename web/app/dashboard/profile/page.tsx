"use client"

import { useState } from "react"
import { KeyRound, User } from "lucide-react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { toast } from "sonner"
import { useUser } from "@/lib/user-context"
import { userApi } from "@/lib/api/users"

export default function ProfilePage() {
  const { user, loading } = useUser()
  const [pwForm, setPwForm] = useState({ old_password: "", new_password: "", confirm: "" })
  const [saving, setSaving] = useState(false)

  const handleChangePassword = async () => {
    if (!pwForm.old_password || !pwForm.new_password) {
      toast.error("请填写旧密码和新密码")
      return
    }
    if (pwForm.new_password.length < 6) {
      toast.error("新密码至少6个字符")
      return
    }
    if (pwForm.new_password !== pwForm.confirm) {
      toast.error("两次输入的新密码不一致")
      return
    }
    setSaving(true)
    try {
      await userApi.changePassword({
        old_password: pwForm.old_password,
        new_password: pwForm.new_password,
      })
      toast.success("密码已修改")
      setPwForm({ old_password: "", new_password: "", confirm: "" })
    } catch {
      toast.error("修改失败，请检查旧密码是否正确")
    } finally {
      setSaving(false)
    }
  }

  if (loading || !user) {
    return (
      <div className="space-y-6">
        <div><h1 className="text-2xl font-bold">个人资料</h1></div>
        <div className="space-y-4">
          {Array.from({ length: 2 }).map((_, i) => (
            <div key={i} className="h-40 animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      </div>
    )
  }

  const roleMap: Record<string, string> = { admin: "管理员", user: "用户", agent: "代理" }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">个人资料</h1>
        <p className="text-muted-foreground">查看和管理你的账户信息</p>
      </div>

      {/* 基本信息 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <User className="h-4 w-4" /> 账户信息
          </CardTitle>
          <CardDescription>你的基本账户信息</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-sm text-muted-foreground">用户名</p>
              <p className="font-medium">{user.username}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">邮箱</p>
              <p className="font-medium">{user.email || "未设置"}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">角色</p>
              <Badge variant="secondary">{roleMap[user.role] || user.role}</Badge>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">注册时间</p>
              <p className="font-medium">
                {user.created_at ? new Date(user.created_at).toLocaleDateString("zh-CN") : "--"}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 修改密码 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <KeyRound className="h-4 w-4" /> 修改密码
          </CardTitle>
          <CardDescription>更新你的登录密码</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-2 max-w-md">
            <Label>当前密码</Label>
            <Input
              type="password"
              value={pwForm.old_password}
              onChange={(e) => setPwForm({ ...pwForm, old_password: e.target.value })}
              placeholder="请输入当前密码"
            />
          </div>
          <Separator />
          <div className="grid gap-2 max-w-md">
            <Label>新密码</Label>
            <Input
              type="password"
              value={pwForm.new_password}
              onChange={(e) => setPwForm({ ...pwForm, new_password: e.target.value })}
              placeholder="请输入新密码（至少6位）"
            />
          </div>
          <div className="grid gap-2 max-w-md">
            <Label>确认新密码</Label>
            <Input
              type="password"
              value={pwForm.confirm}
              onChange={(e) => setPwForm({ ...pwForm, confirm: e.target.value })}
              placeholder="再次输入新密码"
            />
          </div>
          <Button onClick={handleChangePassword} disabled={saving}>
            {saving ? "保存中..." : "修改密码"}
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
