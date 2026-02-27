import Link from "next/link"
import { Button } from "@/components/ui/button"

/*
  404 页面
  功能：当用户访问不存在的路由时显示友好的提示页面
*/
export default function NotFound() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 bg-background">
      <div className="text-center space-y-4">
        <h1 className="text-8xl font-bold text-muted-foreground/30">404</h1>
        <h2 className="text-2xl font-semibold">页面未找到</h2>
        <p className="text-muted-foreground max-w-md">
          您访问的页面不存在或已被移除，请检查链接是否正确。
        </p>
        <div className="flex gap-3 justify-center pt-4">
          <Button asChild>
            <Link href="/dashboard">返回仪表盘</Link>
          </Button>
          <Button variant="outline" asChild>
            <Link href="/login">前往登录</Link>
          </Button>
        </div>
      </div>
    </div>
  )
}
