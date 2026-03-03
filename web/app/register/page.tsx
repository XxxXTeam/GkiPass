"use client"

import { Suspense, useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import { Lock, User, Mail, Eye, EyeOff, ArrowLeft } from "lucide-react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { toast } from "sonner"
import { authApi } from "@/lib/api/auth"
import { setToken, setRole, isAuthenticated } from "@/lib/auth"
import Link from "next/link"

export default function RegisterPage() {
  return (
    <Suspense fallback={
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    }>
      <RegisterForm />
    </Suspense>
  )
}

function RegisterForm() {
  const router = useRouter()
  const [form, setForm] = useState({ username: "", password: "", confirmPassword: "", email: "" })
  const [showPassword, setShowPassword] = useState(false)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (isAuthenticated()) {
      router.replace("/dashboard")
    }
  }, [router])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!form.username || !form.password || !form.email) {
      toast.error("请填写所有必填字段")
      return
    }
    if (form.username.length < 3) {
      toast.error("用户名至少 3 个字符")
      return
    }
    if (form.password.length < 8) {
      toast.error("密码至少 8 个字符")
      return
    }
    if (form.password !== form.confirmPassword) {
      toast.error("两次输入的密码不一致")
      return
    }

    setLoading(true)
    try {
      const res = await authApi.register({
        username: form.username,
        password: form.password,
        email: form.email,
      })
      if (res.success && res.data) {
        setToken(res.data.token)
        if (res.data.role) setRole(res.data.role)
        toast.success(res.data.is_first_user ? "管理员账户创建成功" : "注册成功")
        router.push("/dashboard")
      } else {
        toast.error(res.message || "注册失败")
      }
    } catch {
      toast.error("注册失败，请稍后重试")
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
          <p className="text-sm text-muted-foreground">创建新账户</p>
        </div>

        <Card>
          <CardHeader className="space-y-1">
            <CardTitle className="text-xl">注册</CardTitle>
            <CardDescription>填写以下信息创建您的账户</CardDescription>
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
                    placeholder="至少 3 个字符"
                    className="pl-9"
                    autoComplete="username"
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="email">邮箱</Label>
                <div className="relative">
                  <Mail className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    id="email"
                    type="email"
                    value={form.email}
                    onChange={(e) => setForm({ ...form, email: e.target.value })}
                    placeholder="your@email.com"
                    className="pl-9"
                    autoComplete="email"
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
                    placeholder="至少 8 个字符"
                    className="pl-9 pr-9"
                    autoComplete="new-password"
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

              {/* 密码强度指示器 */}
              {form.password.length > 0 && (
                <PasswordStrength password={form.password} />
              )}

              <div className="space-y-2">
                <Label htmlFor="confirmPassword">确认密码</Label>
                <div className="relative">
                  <Lock className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    id="confirmPassword"
                    type={showPassword ? "text" : "password"}
                    value={form.confirmPassword}
                    onChange={(e) => setForm({ ...form, confirmPassword: e.target.value })}
                    placeholder="再次输入密码"
                    className="pl-9"
                    autoComplete="new-password"
                  />
                </div>
              </div>

              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? "注册中..." : "创建账户"}
              </Button>
            </form>

            <div className="mt-4 text-center text-sm text-muted-foreground">
              已有账户？
              <Link href="/login" className="ml-1 text-primary hover:underline">
                返回登录
              </Link>
            </div>
          </CardContent>
        </Card>

        <div className="text-center">
          <Link href="/login" className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors">
            <ArrowLeft className="h-3 w-3" />
            返回登录页
          </Link>
        </div>
      </div>
    </div>
  )
}

/*
PasswordStrength 密码强度指示器
功能：实时评估密码强度并显示进度条和提示文字
规则：长度≥8 +1, 大写 +1, 小写 +1, 数字 +1, 特殊字符 +1
*/
function PasswordStrength({ password }: { password: string }) {
  const checks = [
    password.length >= 8,
    /[A-Z]/.test(password),
    /[a-z]/.test(password),
    /[0-9]/.test(password),
    /[^A-Za-z0-9]/.test(password),
  ]
  const score = checks.filter(Boolean).length

  const levels = [
    { label: "极弱", color: "bg-red-500", width: "w-1/5" },
    { label: "弱", color: "bg-orange-500", width: "w-2/5" },
    { label: "一般", color: "bg-yellow-500", width: "w-3/5" },
    { label: "强", color: "bg-blue-500", width: "w-4/5" },
    { label: "极强", color: "bg-green-500", width: "w-full" },
  ]
  const level = levels[Math.max(0, score - 1)] || levels[0]

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between text-xs">
        <span className="text-muted-foreground">密码强度</span>
        <span className="text-muted-foreground">{level.label}</span>
      </div>
      <div className="h-1.5 w-full rounded-full bg-muted">
        <div className={`h-full rounded-full transition-all duration-300 ${level.color} ${level.width}`} />
      </div>
    </div>
  )
}
