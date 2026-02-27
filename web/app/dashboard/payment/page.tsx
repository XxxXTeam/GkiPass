"use client"

import { useEffect, useState } from "react"
import { CreditCard, ToggleLeft, ToggleRight, UserPlus } from "lucide-react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  Dialog, DialogContent, DialogDescription, DialogFooter,
  DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import { toast } from "sonner"
import { paymentApi, type PaymentConfig } from "@/lib/api/payment"

export default function PaymentPage() {
  const [configs, setConfigs] = useState<PaymentConfig[]>([])
  const [loading, setLoading] = useState(true)
  const [rechargeOpen, setRechargeOpen] = useState(false)
  const [rechargeForm, setRechargeForm] = useState({ user_id: "", amount: "", reason: "" })

  const fetchConfigs = async () => {
    try {
      const res = await paymentApi.listConfigs()
      if (res.success && res.data) setConfigs(res.data)
    } catch { setConfigs([]) }
    finally { setLoading(false) }
  }

  useEffect(() => { fetchConfigs() }, [])

  const handleToggle = async (id: string) => {
    try {
      await paymentApi.toggleConfig(id)
      toast.success("状态已切换")
      fetchConfigs()
    } catch { toast.error("操作失败") }
  }

  const handleRecharge = async () => {
    const amount = parseFloat(rechargeForm.amount)
    if (!rechargeForm.user_id || !amount || amount <= 0) {
      toast.error("请填写用户ID和有效金额")
      return
    }
    try {
      await paymentApi.manualRecharge({
        user_id: rechargeForm.user_id,
        amount,
        reason: rechargeForm.reason || "管理员手动充值",
      })
      toast.success("充值成功")
      setRechargeOpen(false)
      setRechargeForm({ user_id: "", amount: "", reason: "" })
    } catch { toast.error("充值失败") }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">支付配置</h1>
          <p className="text-muted-foreground">管理支付渠道和手动充值</p>
        </div>
        <Button onClick={() => setRechargeOpen(true)}>
          <UserPlus className="mr-2 h-4 w-4" /> 手动充值
        </Button>
      </div>

      {/* 支付渠道列表 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <CreditCard className="h-4 w-4" /> 支付渠道
          </CardTitle>
          <CardDescription>管理系统支持的支付方式</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-3">
              {Array.from({ length: 2 }).map((_, i) => (
                <div key={i} className="h-12 animate-pulse rounded bg-muted" />
              ))}
            </div>
          ) : configs.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">暂无支付配置</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>名称</TableHead>
                  <TableHead>提供商</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>更新时间</TableHead>
                  <TableHead className="w-20" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {configs.map((config) => (
                  <TableRow key={config.id}>
                    <TableCell className="font-medium">{config.name}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{config.provider}</Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={config.enabled ? "default" : "secondary"}>
                        {config.enabled ? "启用" : "禁用"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {new Date(config.updated_at).toLocaleDateString("zh-CN")}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-7"
                        onClick={() => handleToggle(config.id)}
                      >
                        {config.enabled ? (
                          <ToggleRight className="mr-1 h-4 w-4 text-green-500" />
                        ) : (
                          <ToggleLeft className="mr-1 h-4 w-4 text-muted-foreground" />
                        )}
                        {config.enabled ? "禁用" : "启用"}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* 手动充值弹窗 */}
      <Dialog open={rechargeOpen} onOpenChange={setRechargeOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>手动充值</DialogTitle>
            <DialogDescription>为指定用户手动添加余额</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-2">
              <Label>用户 ID</Label>
              <Input
                value={rechargeForm.user_id}
                onChange={(e) => setRechargeForm({ ...rechargeForm, user_id: e.target.value })}
                placeholder="输入用户 ID"
              />
            </div>
            <div className="grid gap-2">
              <Label>充值金额 (元)</Label>
              <Input
                type="number"
                value={rechargeForm.amount}
                onChange={(e) => setRechargeForm({ ...rechargeForm, amount: e.target.value })}
                placeholder="0.00"
                min="0.01"
                step="0.01"
              />
            </div>
            <div className="grid gap-2">
              <Label>充值原因</Label>
              <Textarea
                value={rechargeForm.reason}
                onChange={(e) => setRechargeForm({ ...rechargeForm, reason: e.target.value })}
                placeholder="可选，如：补偿、活动赠送等"
                rows={2}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRechargeOpen(false)}>取消</Button>
            <Button onClick={handleRecharge}>确认充值</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
