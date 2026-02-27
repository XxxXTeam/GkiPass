"use client"

import { useEffect, useState, useCallback } from "react"
import { useUser } from "@/lib/user-context"
import { usePathname } from "next/navigation"
import { Moon, Sun, Bell, User, LogOut, Settings, Check } from "lucide-react"
import { useTheme } from "next-themes"
import { Button } from "@/components/ui/button"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { Separator } from "@/components/ui/separator"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { removeToken } from "@/lib/auth"
import { notificationApi, type Notification } from "@/lib/api/notifications"

/*
  面包屑路径映射
  功能：将 URL 路径段映射为中文显示名
*/
const breadcrumbMap: Record<string, string> = {
  dashboard: "仪表盘",
  tunnels: "隧道管理",
  nodes: "节点管理",
  "node-groups": "节点组",
  users: "用户管理",
  plans: "套餐管理",
  acl: "协议限制",
  monitoring: "系统监控",
  settings: "系统设置",
  announcements: "公告管理",
  subscription: "订阅与钱包",
  profile: "个人资料",
  payment: "支付配置",
  traffic: "流量统计",
  notifications: "通知中心",
  certificates: "证书管理",
}

export function Header() {
  const { setTheme, theme } = useTheme()
  const pathname = usePathname()
  const { user } = useUser()
  const [notifications, setNotifications] = useState<Notification[]>([])

  const username = user?.username || ""
  const unreadCount = notifications.filter((n) => !n.read).length

  const fetchNotifications = useCallback(() => {
    notificationApi.list().then((res) => {
      if (res.success && res.data) setNotifications(res.data)
    }).catch(() => {})
  }, [])

  useEffect(() => {
    fetchNotifications()
  }, [fetchNotifications])

  const handleMarkAllRead = async () => {
    try {
      await notificationApi.markAllAsRead()
      setNotifications((prev) => prev.map((n) => ({ ...n, read: true })))
    } catch { /* 忽略 */ }
  }

  /* 生成面包屑 */
  const segments = pathname.split("/").filter(Boolean)
  const breadcrumbs = segments.map((seg) => breadcrumbMap[seg] || seg)

  return (
    <header className="flex h-14 shrink-0 items-center gap-2 border-b px-4">
      <SidebarTrigger className="-ml-1" />
      <Separator orientation="vertical" className="mr-2 h-4" />

      {/* 面包屑导航 */}
      <nav className="flex items-center gap-1 text-sm text-muted-foreground">
        {breadcrumbs.map((crumb, i) => (
          <span key={i} className="flex items-center gap-1">
            {i > 0 && <span>/</span>}
            <span className={i === breadcrumbs.length - 1 ? "text-foreground font-medium" : ""}>
              {crumb}
            </span>
          </span>
        ))}
      </nav>

      <div className="flex-1" />

      <div className="flex items-center gap-2">
        {/* 通知面板 */}
        <Popover>
          <PopoverTrigger asChild>
            <Button variant="ghost" size="icon" className="relative">
              <Bell className="h-4 w-4" />
              {unreadCount > 0 && (
                <span className="absolute -top-0.5 -right-0.5 h-3 w-3 rounded-full bg-destructive text-[8px] text-white flex items-center justify-center">
                  {unreadCount > 9 ? "9+" : unreadCount}
                </span>
              )}
            </Button>
          </PopoverTrigger>
          <PopoverContent align="end" className="w-80 p-0">
            <div className="flex items-center justify-between border-b px-4 py-3">
              <p className="text-sm font-medium">通知</p>
              {unreadCount > 0 && (
                <Button variant="ghost" size="sm" className="h-auto py-1 px-2 text-xs" onClick={handleMarkAllRead}>
                  <Check className="mr-1 h-3 w-3" /> 全部已读
                </Button>
              )}
            </div>
            <div className="max-h-72 overflow-y-auto">
              {notifications.length === 0 ? (
                <p className="py-8 text-center text-sm text-muted-foreground">暂无通知</p>
              ) : (
                notifications.slice(0, 10).map((n) => (
                  <div key={n.id} className={`border-b px-4 py-3 text-sm ${n.read ? "opacity-60" : ""}`}>
                    <p className="font-medium">{n.title}</p>
                    <p className="text-xs text-muted-foreground mt-0.5 line-clamp-2">{n.content}</p>
                  </div>
                ))
              )}
            </div>
          </PopoverContent>
        </Popover>

        {/* 主题切换 */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon">
              {theme === "dark" ? (
                <Moon className="h-4 w-4" />
              ) : (
                <Sun className="h-4 w-4" />
              )}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => setTheme("light")}>
              浅色
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setTheme("dark")}>
              深色
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setTheme("system")}>
              跟随系统
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        {/* 用户菜单 */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="gap-2">
              <div className="flex h-6 w-6 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs font-bold">
                {username ? username.charAt(0).toUpperCase() : "U"}
              </div>
              <span className="hidden sm:inline text-sm">{username || "用户"}</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-48">
            <DropdownMenuLabel className="font-normal">
              <p className="text-sm font-medium">{username || "用户"}</p>
              <p className="text-xs text-muted-foreground">管理员</p>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem>
              <User className="mr-2 h-4 w-4" />
              个人资料
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => window.location.href = "/dashboard/settings"}>
              <Settings className="mr-2 h-4 w-4" />
              系统设置
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="text-destructive"
              onClick={() => {
                removeToken()
                window.location.href = "/login"
              }}
            >
              <LogOut className="mr-2 h-4 w-4" />
              退出登录
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  )
}
