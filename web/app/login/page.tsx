"use client"

import { Suspense, useState, useEffect } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { Lock, User, Eye, EyeOff } from "lucide-react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { toast } from "sonner"
import { authApi } from "@/lib/api/auth"
import { announcementApi, type Announcement } from "@/lib/api/announcements"
import { setToken, setRole, isAuthenticated } from "@/lib/auth"

export default function LoginPage() {
  return (
    <Suspense fallback={
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    }>
      <LoginForm />
    </Suspense>
  )
}

function LoginForm() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [form, setForm] = useState({ username: "", password: "" })
  const [showPassword, setShowPassword] = useState(false)
  const [loading, setLoading] = useState(false)
  const [rememberMe, setRememberMe] = useState(false)
  const [announcements, setAnnouncements] = useState<Announcement[]>([])

  /* 已登录自动跳转 + 加载公告 + 恢复记住的用户名 */
  useEffect(() => {
    if (isAuthenticated()) {
      router.replace("/dashboard")
    }
    const saved = localStorage.getItem("gkipass_remember_username")
    if (saved) {
      setForm((prev) => ({ ...prev, username: saved }))
      setRememberMe(true)
    }
    announcementApi.listActive().then((res) => {
      if (res.success && res.data) setAnnouncements(res.data.slice(0, 2))
    }).catch(() => {})
  }, [router])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!form.username || !form.password) {
      toast.error("请输入用户名和密码")
      return
    }

    setLoading(true)
    try {
      const res = await authApi.login(form)
      if (res.success && res.data) {
        setToken(res.data.token)
        if (res.data.user?.role) setRole(res.data.user.role)
        if (rememberMe) {
          localStorage.setItem("gkipass_remember_username", form.username)
        } else {
          localStorage.removeItem("gkipass_remember_username")
        }
        toast.success("登录成功")
        const redirect = searchParams.get("redirect") || "/dashboard"
        router.push(redirect)
      } else {
        toast.error(res.message || "登录失败")
      }
    } catch {
      toast.error("登录失败，请检查用户名和密码")
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <div className="w-full max-w-md space-y-6">
        {/* Logo */}
        <div className="flex flex-col items-center space-y-2 text-center">
          <div className="flex h-14 w-14 items-center justify-center rounded-xl bg-primary text-primary-foreground text-2xl font-bold">
            G
          </div>
          <h1 className="text-2xl font-bold">GkiPass</h1>
          <p className="text-sm text-muted-foreground">隧道管理控制面板</p>
        </div>

        {/* 登录卡片 */}
        <Card>
          <CardHeader className="space-y-1">
            <CardTitle className="text-xl">登录</CardTitle>
            <CardDescription>输入您的账号信息以访问控制面板</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="username">用户名</Label>
                <div className="relative">
                  <User className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    id="username"
                    value={form.username}
                    onChange={(e) => setForm({ ...form, username: e.target.value })}
                    placeholder="请输入用户名"
                    className="pl-9"
                    autoComplete="username"
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="password">密码</Label>
                <div className="relative">
                  <Lock className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    id="password"
                    type={showPassword ? "text" : "password"}
                    value={form.password}
                    onChange={(e) => setForm({ ...form, password: e.target.value })}
                    placeholder="请输入密码"
                    className="pl-9 pr-9"
                    autoComplete="current-password"
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
                    onClick={() => setShowPassword(!showPassword)}
                  >
                    {showPassword ? (
                      <EyeOff className="h-4 w-4 text-muted-foreground" />
                    ) : (
                      <Eye className="h-4 w-4 text-muted-foreground" />
                    )}
                  </Button>
                </div>
              </div>

              <div className="flex items-center gap-2">
                <Checkbox
                  id="remember"
                  checked={rememberMe}
                  onCheckedChange={(v) => setRememberMe(v === true)}
                />
                <Label htmlFor="remember" className="text-sm font-normal text-muted-foreground cursor-pointer">
                  记住用户名
                </Label>
              </div>

              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? "登录中..." : "登录"}
              </Button>
            </form>
          </CardContent>
        </Card>

        {/* 活跃公告 */}
        {announcements.length > 0 && (
          <div className="space-y-2">
            {announcements.map((a) => (
              <div key={a.id} className="rounded-lg border bg-muted/50 px-3 py-2 text-xs text-muted-foreground">
                <span className="font-medium text-foreground">{a.title}</span>
                {a.content && <span className="ml-1">{a.content}</span>}
              </div>
            ))}
          </div>
        )}

        <p className="text-center text-xs text-muted-foreground">
          GkiPass v2.0.0 - 高性能双向隧道转发系统
        </p>
      </div>
    </div>
  )
}
