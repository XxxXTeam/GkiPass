import type { Metadata } from "next"
import { SidebarProvider, SidebarInset } from "@/components/ui/sidebar"
import { AppSidebar } from "@/components/layout/app-sidebar"
import { Header } from "@/components/layout/header"
import { TooltipProvider } from "@/components/ui/tooltip"
import { AuthGuard } from "@/components/auth-guard"
import { UserProvider } from "@/lib/user-context"

export const metadata: Metadata = {
  title: {
    template: "%s | GkiPass",
    default: "仪表盘 | GkiPass",
  },
}

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <AuthGuard>
      <UserProvider>
        <TooltipProvider>
          <SidebarProvider>
            <AppSidebar />
            <SidebarInset>
              <Header />
              <main className="flex-1 overflow-auto p-6">{children}</main>
            </SidebarInset>
          </SidebarProvider>
        </TooltipProvider>
      </UserProvider>
    </AuthGuard>
  )
}
