"use client"

import { useEffect } from "react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { AlertTriangle, RefreshCw, Home } from "lucide-react"
import Link from "next/link"

/*
  Dashboard 局部错误边界
  功能：捕获 Dashboard 子页面的运行时错误，保持侧边栏和头部布局不被破坏
*/
export default function DashboardError({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  useEffect(() => {
    console.error("[GkiPass Dashboard] 页面错误:", error)
  }, [error])

  return (
    <div className="flex flex-1 items-center justify-center p-6">
      <Card className="max-w-md w-full">
        <CardContent className="pt-6 text-center space-y-4">
          <AlertTriangle className="h-12 w-12 text-destructive mx-auto" />
          <h2 className="text-xl font-semibold">页面加载出错</h2>
          <p className="text-sm text-muted-foreground">
            当前页面发生了意外错误，请尝试刷新或返回仪表盘。
          </p>
          {error.digest && (
            <p className="text-xs text-muted-foreground font-mono">
              错误代码: {error.digest}
            </p>
          )}
          <div className="flex gap-3 justify-center pt-2">
            <Button size="sm" onClick={reset}>
              <RefreshCw className="mr-2 h-3 w-3" /> 重试
            </Button>
            <Button variant="outline" size="sm" asChild>
              <Link href="/dashboard"><Home className="mr-2 h-3 w-3" /> 仪表盘</Link>
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
