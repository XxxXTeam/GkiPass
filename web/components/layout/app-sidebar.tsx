"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import {
  LayoutDashboard,
  Network,
  Server,
  FolderTree,
  Users,
  CreditCard,
  Activity,
  Settings,
  Shield,
  LogOut,
  Wallet,
  UserCircle,
  Megaphone,
  BarChart3,
  Bell,
  FileKey2,
  Banknote,
} from "lucide-react"

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"
import { Button } from "@/components/ui/button"
import { removeToken } from "@/lib/auth"
import { useUser } from "@/lib/user-context"

/*
  侧边栏导航配置
  功能：定义主导航和管理员专属导航，根据用户角色动态展示
*/
const mainNav = [
  { title: "仪表盘", icon: LayoutDashboard, href: "/dashboard" },
  { title: "隧道管理", icon: Network, href: "/dashboard/tunnels" },
  { title: "节点管理", icon: Server, href: "/dashboard/nodes" },
  { title: "节点组", icon: FolderTree, href: "/dashboard/node-groups" },
  { title: "流量统计", icon: BarChart3, href: "/dashboard/traffic" },
  { title: "通知中心", icon: Bell, href: "/dashboard/notifications" },
  { title: "订阅与钱包", icon: Wallet, href: "/dashboard/subscription" },
  { title: "个人资料", icon: UserCircle, href: "/dashboard/profile" },
]

const adminNav = [
  { title: "用户管理", icon: Users, href: "/dashboard/users" },
  { title: "套餐管理", icon: CreditCard, href: "/dashboard/plans" },
  { title: "支付配置", icon: Banknote, href: "/dashboard/payment" },
  { title: "公告管理", icon: Megaphone, href: "/dashboard/announcements" },
  { title: "证书管理", icon: FileKey2, href: "/dashboard/certificates" },
  { title: "协议限制", icon: Shield, href: "/dashboard/acl" },
  { title: "系统监控", icon: Activity, href: "/dashboard/monitoring" },
  { title: "系统设置", icon: Settings, href: "/dashboard/settings" },
]

export function AppSidebar() {
  const pathname = usePathname()
  const { user } = useUser()

  const isAdmin = user?.role === "admin"

  const isActive = (href: string) => {
    if (href === "/dashboard") return pathname === "/dashboard"
    return pathname.startsWith(href)
  }

  return (
    <Sidebar variant="inset" collapsible="icon">
      <SidebarHeader className="border-b border-sidebar-border">
        <Link href="/dashboard" className="flex items-center gap-2 px-2 py-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground font-bold text-sm">
            G
          </div>
          <div className="flex flex-col group-data-[collapsible=icon]:hidden">
            <span className="font-semibold text-sm">GkiPass</span>
            <span className="text-[10px] text-muted-foreground">隧道管理面板</span>
          </div>
        </Link>
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>导航</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {mainNav.map((item) => (
                <SidebarMenuItem key={item.href}>
                  <SidebarMenuButton asChild isActive={isActive(item.href)}>
                    <Link href={item.href}>
                      <item.icon className="h-4 w-4" />
                      <span>{item.title}</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        {isAdmin && (
          <SidebarGroup>
            <SidebarGroupLabel>管理</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {adminNav.map((item) => (
                  <SidebarMenuItem key={item.href}>
                    <SidebarMenuButton asChild isActive={isActive(item.href)}>
                      <Link href={item.href}>
                        <item.icon className="h-4 w-4" />
                        <span>{item.title}</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        )}
      </SidebarContent>

      <SidebarFooter className="border-t border-sidebar-border">
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton asChild>
              <Button
                variant="ghost"
                className="w-full justify-start"
                onClick={() => {
                  removeToken()
                  window.location.href = "/login"
                }}
              >
                <LogOut className="h-4 w-4" />
                <span>退出登录</span>
              </Button>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  )
}
