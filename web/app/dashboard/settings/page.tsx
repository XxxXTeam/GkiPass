"use client"

import { useEffect, useState } from "react"
import { Save } from "lucide-react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Separator } from "@/components/ui/separator"
import { toast } from "sonner"
import { settingsApi } from "@/lib/api/settings"

/*
  设置表单分类状态
  功能：每个 Tab 独立管理自己的表单数据，对应后端不同的 settings API
*/
interface GeneralSettings {
  site_name: string
  site_description: string
  max_tunnels_per_user: number
  max_bandwidth_per_tunnel: number
}

interface SecuritySettings {
  registration_enabled: boolean
  captcha_enabled: boolean
}

interface NotificationSettings {
  enable_email_notification: boolean
  smtp_host: string
  smtp_port: number
  smtp_user: string
  smtp_password: string
}

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState("general")
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  const [general, setGeneral] = useState<GeneralSettings>({
    site_name: "GkiPass", site_description: "高性能双向隧道转发系统",
    max_tunnels_per_user: 10, max_bandwidth_per_tunnel: 0,
  })

  const [security, setSecurity] = useState<SecuritySettings>({
    registration_enabled: true, captcha_enabled: false,
  })

  const [notification, setNotification] = useState<NotificationSettings>({
    enable_email_notification: false, smtp_host: "", smtp_port: 587,
    smtp_user: "", smtp_password: "",
  })

  useEffect(() => {
    const fetchAll = async () => {
      try {
        const [genRes, secRes, notifRes] = await Promise.allSettled([
          settingsApi.get(),
          settingsApi.getSecurity(),
          settingsApi.getNotification(),
        ])
        if (genRes.status === "fulfilled" && genRes.value.success && genRes.value.data) {
          setGeneral(genRes.value.data as unknown as GeneralSettings)
        }
        if (secRes.status === "fulfilled" && secRes.value.success && secRes.value.data) {
          setSecurity(secRes.value.data as unknown as SecuritySettings)
        }
        if (notifRes.status === "fulfilled" && notifRes.value.success && notifRes.value.data) {
          setNotification(notifRes.value.data as unknown as NotificationSettings)
        }
      } catch { /* 使用默认值 */ }
      finally { setLoading(false) }
    }
    fetchAll()
  }, [])

  /* 按当前活动 Tab 保存对应分类设置 */
  const handleSave = async () => {
    setSaving(true)
    try {
      switch (activeTab) {
        case "general":
          await settingsApi.update(general as unknown as Record<string, unknown>)
          break
        case "security":
          await settingsApi.updateSecurity(security as unknown as Record<string, unknown>)
          break
        case "email":
          await settingsApi.updateNotification(notification as unknown as Record<string, unknown>)
          break
      }
      toast.success("设置已保存")
    } catch { toast.error("保存失败") }
    finally { setSaving(false) }
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div><h1 className="text-2xl font-bold">系统设置</h1></div>
        <div className="space-y-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="h-32 animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">系统设置</h1>
          <p className="text-muted-foreground">管理系统全局配置</p>
        </div>
        <Button onClick={handleSave} disabled={saving}>
          <Save className="mr-2 h-4 w-4" />
          {saving ? "保存中..." : "保存设置"}
        </Button>
      </div>

      <Tabs defaultValue="general" onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="general">基础设置</TabsTrigger>
          <TabsTrigger value="security">安全设置</TabsTrigger>
          <TabsTrigger value="email">邮件通知</TabsTrigger>
        </TabsList>

        <TabsContent value="general" className="space-y-4 mt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">站点信息</CardTitle>
              <CardDescription>配置系统的基本展示信息</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-2">
                <Label>站点名称</Label>
                <Input value={general.site_name} onChange={(e) => setGeneral({ ...general, site_name: e.target.value })} />
              </div>
              <div className="grid gap-2">
                <Label>站点描述</Label>
                <Input value={general.site_description} onChange={(e) => setGeneral({ ...general, site_description: e.target.value })} />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">默认限制</CardTitle>
              <CardDescription>新用户的默认资源限制</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="grid gap-2">
                  <Label>每用户最大隧道数</Label>
                  <Input type="number" value={general.max_tunnels_per_user} onChange={(e) => setGeneral({ ...general, max_tunnels_per_user: parseInt(e.target.value) || 0 })} />
                </div>
                <div className="grid gap-2">
                  <Label>每隧道最大带宽 (bps, 0=无限)</Label>
                  <Input type="number" value={general.max_bandwidth_per_tunnel} onChange={(e) => setGeneral({ ...general, max_bandwidth_per_tunnel: parseInt(e.target.value) || 0 })} />
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="security" className="space-y-4 mt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">注册与验证</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between rounded-lg border p-3">
                <div>
                  <p className="text-sm font-medium">开放注册</p>
                  <p className="text-xs text-muted-foreground">允许新用户自行注册</p>
                </div>
                <Switch checked={security.registration_enabled} onCheckedChange={(v) => setSecurity({ ...security, registration_enabled: v })} />
              </div>
              <Separator />
              <div className="flex items-center justify-between rounded-lg border p-3">
                <div>
                  <p className="text-sm font-medium">验证码</p>
                  <p className="text-xs text-muted-foreground">登录和注册时启用验证码</p>
                </div>
                <Switch checked={security.captcha_enabled} onCheckedChange={(v) => setSecurity({ ...security, captcha_enabled: v })} />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="email" className="space-y-4 mt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">邮件通知</CardTitle>
              <CardDescription>配置 SMTP 服务器用于发送系统通知邮件</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between rounded-lg border p-3">
                <div>
                  <p className="text-sm font-medium">启用邮件通知</p>
                  <p className="text-xs text-muted-foreground">发送节点离线、流量告警等通知</p>
                </div>
                <Switch checked={notification.enable_email_notification} onCheckedChange={(v) => setNotification({ ...notification, enable_email_notification: v })} />
              </div>
              {notification.enable_email_notification && (
                <>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="grid gap-2">
                      <Label>SMTP 主机</Label>
                      <Input value={notification.smtp_host} onChange={(e) => setNotification({ ...notification, smtp_host: e.target.value })} placeholder="smtp.example.com" />
                    </div>
                    <div className="grid gap-2">
                      <Label>SMTP 端口</Label>
                      <Input type="number" value={notification.smtp_port} onChange={(e) => setNotification({ ...notification, smtp_port: parseInt(e.target.value) || 587 })} />
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="grid gap-2">
                      <Label>SMTP 用户名</Label>
                      <Input value={notification.smtp_user} onChange={(e) => setNotification({ ...notification, smtp_user: e.target.value })} />
                    </div>
                    <div className="grid gap-2">
                      <Label>SMTP 密码</Label>
                      <Input type="password" value={notification.smtp_password} onChange={(e) => setNotification({ ...notification, smtp_password: e.target.value })} />
                    </div>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
