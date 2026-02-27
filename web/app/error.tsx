"use client"

import { useEffect } from "react"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { AlertTriangle } from "lucide-react"

/*
  全局错误边界
  功能：捕获页面级运行时错误，显示友好的错误提示并提供重试操作
*/
export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  const router = useRouter()

  useEffect(() => {
    console.error("[GkiPass] 页面错误:", error)
  }, [error])

  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 bg-background">
      <div className="text-center space-y-4">
        <AlertTriangle className="h-16 w-16 text-destructive mx-auto" />
        <h2 className="text-2xl font-semibold">页面出现错误</h2>
        <p className="text-muted-foreground max-w-md">
          加载页面时发生了意外错误，请尝试重新加载。如果问题持续存在，请联系管理员。
        </p>
        {error.digest && (
          <p className="text-xs text-muted-foreground font-mono">
            错误代码: {error.digest}
          </p>
        )}
        <div className="flex gap-3 justify-center pt-4">
          <Button onClick={reset}>重试</Button>
          <Button variant="outline" onClick={() => router.push("/dashboard")}>
            返回仪表盘
          </Button>
        </div>
      </div>
    </div>
  )
}
