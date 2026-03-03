"use client"

import { useEffect, useState } from "react"
import { usePathname } from "next/navigation"

/*
RouteProgress 路由切换进度条
功能：页面路由切换时在顶部显示蓝色进度条，提升用户感知
纯 CSS 实现，无额外依赖
*/
export function RouteProgress() {
  const pathname = usePathname()
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    setLoading(true)
    const timer = setTimeout(() => setLoading(false), 500)
    return () => clearTimeout(timer)
  }, [pathname])

  if (!loading) return null

  return (
    <div className="fixed top-0 left-0 right-0 z-[9999] h-0.5">
      <div className="h-full bg-primary animate-progress rounded-r-full" />
      <style jsx>{`
        @keyframes progress {
          0% { width: 0%; }
          50% { width: 70%; }
          100% { width: 100%; opacity: 0; }
        }
        .animate-progress {
          animation: progress 500ms ease-out forwards;
        }
      `}</style>
    </div>
  )
}
