import { type LucideIcon, Inbox } from "lucide-react"
import { Button } from "@/components/ui/button"
import Link from "next/link"

/*
EmptyState 空状态提示组件
功能：当列表数据为空时显示友好的引导提示
支持自定义图标、标题、描述和操作按钮
*/
interface EmptyStateProps {
  icon?: LucideIcon
  title?: string
  description?: string
  actionLabel?: string
  actionHref?: string
  onAction?: () => void
}

export function EmptyState({
  icon: Icon = Inbox,
  title = "暂无数据",
  description = "当前没有可显示的内容",
  actionLabel,
  actionHref,
  onAction,
}: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <div className="rounded-full bg-muted p-4 mb-4">
        <Icon className="h-8 w-8 text-muted-foreground" />
      </div>
      <h3 className="text-lg font-semibold mb-1">{title}</h3>
      <p className="text-sm text-muted-foreground max-w-sm mb-4">{description}</p>
      {actionLabel && actionHref && (
        <Button asChild size="sm">
          <Link href={actionHref}>{actionLabel}</Link>
        </Button>
      )}
      {actionLabel && onAction && !actionHref && (
        <Button size="sm" onClick={onAction}>{actionLabel}</Button>
      )}
    </div>
  )
}
