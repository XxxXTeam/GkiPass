"use client"

import { useEffect, useState } from "react"
import { Bell, Check, CheckCheck, Trash2 } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { toast } from "sonner"
import { notificationApi, type Notification } from "@/lib/api/notifications"

export default function NotificationsPage() {
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [loading, setLoading] = useState(true)

  const fetchNotifications = async () => {
    try {
      const res = await notificationApi.list()
      if (res.success && res.data) setNotifications(res.data)
    } catch { setNotifications([]) }
    finally { setLoading(false) }
  }

  useEffect(() => { fetchNotifications() }, [])

  const handleMarkAsRead = async (id: string) => {
    try {
      await notificationApi.markAsRead(id)
      setNotifications((prev) => prev.map((n) => n.id === id ? { ...n, read: true } : n))
    } catch { toast.error("操作失败") }
  }

  const handleMarkAllAsRead = async () => {
    try {
      await notificationApi.markAllAsRead()
      setNotifications((prev) => prev.map((n) => ({ ...n, read: true })))
      toast.success("全部标记为已读")
    } catch { toast.error("操作失败") }
  }

  const handleDelete = async (id: string) => {
    try {
      await notificationApi.delete(id)
      setNotifications((prev) => prev.filter((n) => n.id !== id))
      toast.success("已删除")
    } catch { toast.error("删除失败") }
  }

  const unreadCount = notifications.filter((n) => !n.read).length

  const typeColorMap: Record<string, string> = {
    info: "bg-blue-500",
    warning: "bg-amber-500",
    error: "bg-red-500",
    success: "bg-green-500",
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">通知中心</h1>
          <p className="text-muted-foreground">
            {unreadCount > 0 ? `${unreadCount} 条未读通知` : "暂无未读通知"}
          </p>
        </div>
        {unreadCount > 0 && (
          <Button variant="outline" size="sm" onClick={handleMarkAllAsRead}>
            <CheckCheck className="mr-2 h-4 w-4" /> 全部已读
          </Button>
        )}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Bell className="h-4 w-4" /> 通知列表
          </CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="h-16 animate-pulse rounded bg-muted" />
              ))}
            </div>
          ) : notifications.length === 0 ? (
            <p className="py-12 text-center text-sm text-muted-foreground">暂无通知</p>
          ) : (
            <div className="space-y-2">
              {notifications.map((n) => (
                <div
                  key={n.id}
                  className={`flex items-start gap-3 rounded-lg border p-4 transition-colors ${
                    n.read ? "opacity-60" : "bg-accent/30"
                  }`}
                >
                  <div className={`mt-1 h-2 w-2 shrink-0 rounded-full ${typeColorMap[n.type] || "bg-muted-foreground"}`} />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <p className="text-sm font-medium">{n.title}</p>
                      {!n.read && <Badge variant="default" className="text-[10px] h-4">未读</Badge>}
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">{n.content}</p>
                    <p className="text-[10px] text-muted-foreground mt-2">
                      {new Date(n.created_at).toLocaleString("zh-CN")}
                    </p>
                  </div>
                  <div className="flex shrink-0 gap-1">
                    {!n.read && (
                      <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => handleMarkAsRead(n.id)}>
                        <Check className="h-3 w-3" />
                      </Button>
                    )}
                    <Button variant="ghost" size="icon" className="h-7 w-7 text-destructive" onClick={() => handleDelete(n.id)}>
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
