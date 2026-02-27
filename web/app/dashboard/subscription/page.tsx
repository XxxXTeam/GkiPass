"use client"

import { useEffect, useState } from "react"
import { CreditCard, Wallet, Clock, ArrowUpRight, Package } from "lucide-react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog, DialogContent, DialogDescription, DialogFooter,
  DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { toast } from "sonner"
import { walletApi, type WalletBalance, type Transaction } from "@/lib/api/wallet"
import { subscriptionApi, type Subscription } from "@/lib/api/subscriptions"
import { planApi } from "@/lib/api/plans"
import type { Plan } from "@/lib/types"
import { formatBytes } from "@/lib/utils"

export default function SubscriptionPage() {
  const [balance, setBalance] = useState<WalletBalance | null>(null)
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [subscription, setSubscription] = useState<Subscription | null>(null)
  const [plans, setPlans] = useState<Plan[]>([])
  const [loading, setLoading] = useState(true)
  const [rechargeOpen, setRechargeOpen] = useState(false)
  const [rechargeAmount, setRechargeAmount] = useState("")

  useEffect(() => {
    const fetchAll = async () => {
      try {
        const [balRes, txRes, subRes, planRes] = await Promise.allSettled([
          walletApi.balance(),
          walletApi.transactions(),
          subscriptionApi.current(),
          planApi.list(),
        ])
        if (balRes.status === "fulfilled" && balRes.value.success && balRes.value.data) {
          setBalance(balRes.value.data)
        }
        if (txRes.status === "fulfilled" && txRes.value.success && txRes.value.data) {
          setTransactions(txRes.value.data)
        }
        if (subRes.status === "fulfilled" && subRes.value.success && subRes.value.data) {
          setSubscription(subRes.value.data)
        }
        if (planRes.status === "fulfilled" && planRes.value.success && planRes.value.data) {
          setPlans(planRes.value.data.filter((p) => p.enabled))
        }
      } catch { /* 忽略 */ }
      finally { setLoading(false) }
    }
    fetchAll()
  }, [])

  const handleRecharge = async () => {
    const amount = parseFloat(rechargeAmount)
    if (!amount || amount <= 0) { toast.error("请输入有效金额"); return }
    try {
      await walletApi.createRechargeOrder(amount)
      toast.success("充值订单已创建")
      setRechargeOpen(false)
      setRechargeAmount("")
    } catch { toast.error("充值失败") }
  }

  const handleSubscribe = async (planId: string) => {
    try {
      const { apiGet } = await import("@/lib/api/client")
      await apiGet(`/plans/${planId}/subscribe`)
      toast.success("订阅成功")
      /* 刷新订阅状态 */
      const subRes = await subscriptionApi.current()
      if (subRes.success && subRes.data) setSubscription(subRes.data)
      const balRes = await walletApi.balance()
      if (balRes.success && balRes.data) setBalance(balRes.data)
    } catch { toast.error("订阅失败，请检查余额") }
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div><h1 className="text-2xl font-bold">订阅与钱包</h1></div>
        <div className="grid gap-4 md:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="h-32 animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">订阅与钱包</h1>
        <p className="text-muted-foreground">管理你的套餐订阅和钱包余额</p>
      </div>

      {/* 概览卡片 */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <p className="text-sm text-muted-foreground">钱包余额</p>
                <p className="text-3xl font-bold">¥{balance?.balance?.toFixed(2) || "0.00"}</p>
              </div>
              <div className="rounded-lg bg-green-500/10 p-3">
                <Wallet className="h-5 w-5 text-green-500" />
              </div>
            </div>
            <Button variant="outline" size="sm" className="mt-3 w-full" onClick={() => setRechargeOpen(true)}>
              <ArrowUpRight className="mr-2 h-4 w-4" /> 充值
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <p className="text-sm text-muted-foreground">当前套餐</p>
                <p className="text-xl font-bold">{subscription?.plan_name || "无"}</p>
                {subscription && (
                  <Badge variant={subscription.status === "active" ? "default" : "secondary"}>
                    {subscription.status === "active" ? "生效中" : "已过期"}
                  </Badge>
                )}
              </div>
              <div className="rounded-lg bg-blue-500/10 p-3">
                <Package className="h-5 w-5 text-blue-500" />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <p className="text-sm text-muted-foreground">到期时间</p>
                <p className="text-xl font-bold">
                  {subscription?.expires_at
                    ? new Date(subscription.expires_at).toLocaleDateString("zh-CN")
                    : "--"}
                </p>
                {subscription && (
                  <p className="text-xs text-muted-foreground">
                    已用流量 {formatBytes(subscription.traffic_used)} / {formatBytes(subscription.traffic_limit, "无限制")}
                  </p>
                )}
              </div>
              <div className="rounded-lg bg-amber-500/10 p-3">
                <Clock className="h-5 w-5 text-amber-500" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 可选套餐 */}
      {plans.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <CreditCard className="h-4 w-4" /> 可选套餐
            </CardTitle>
            <CardDescription>选择适合你的套餐计划</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {plans.map((plan) => (
                <div key={plan.id} className="rounded-lg border p-4 space-y-3">
                  <div>
                    <p className="font-semibold">{plan.name}</p>
                    {plan.description && (
                      <p className="text-xs text-muted-foreground mt-1">{plan.description}</p>
                    )}
                  </div>
                  <div className="text-2xl font-bold">
                    {plan.price > 0 ? `¥${plan.price}` : "免费"}
                    <span className="text-sm font-normal text-muted-foreground">/{plan.duration}天</span>
                  </div>
                  <ul className="text-xs text-muted-foreground space-y-1">
                    <li>流量限制：{formatBytes(plan.traffic_limit, "无限制")}</li>
                    <li>隧道数量：{plan.rule_limit > 0 ? plan.rule_limit : "无限制"}</li>
                  </ul>
                  <Button
                    variant="outline"
                    size="sm"
                    className="w-full"
                    onClick={() => handleSubscribe(plan.id)}
                    disabled={subscription?.plan_id === plan.id && subscription?.status === "active"}
                  >
                    {subscription?.plan_id === plan.id && subscription?.status === "active"
                      ? "当前套餐"
                      : "立即订阅"}
                  </Button>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* 交易记录 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">交易记录</CardTitle>
        </CardHeader>
        <CardContent>
          {transactions.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">暂无交易记录</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>类型</TableHead>
                  <TableHead>金额</TableHead>
                  <TableHead>描述</TableHead>
                  <TableHead>时间</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {transactions.slice(0, 20).map((tx) => (
                  <TableRow key={tx.id}>
                    <TableCell>
                      <Badge variant={tx.amount > 0 ? "default" : "secondary"}>
                        {tx.type}
                      </Badge>
                    </TableCell>
                    <TableCell className={tx.amount > 0 ? "text-green-600" : "text-red-600"}>
                      {tx.amount > 0 ? "+" : ""}{tx.amount.toFixed(2)}
                    </TableCell>
                    <TableCell className="text-sm">{tx.description}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {new Date(tx.created_at).toLocaleString("zh-CN")}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* 充值弹窗 */}
      <Dialog open={rechargeOpen} onOpenChange={setRechargeOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>钱包充值</DialogTitle>
            <DialogDescription>输入充值金额创建充值订单</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-2">
              <Label>充值金额 (元)</Label>
              <Input
                type="number"
                value={rechargeAmount}
                onChange={(e) => setRechargeAmount(e.target.value)}
                placeholder="请输入金额"
                min="1"
                step="0.01"
              />
            </div>
            <div className="flex gap-2">
              {[10, 50, 100, 500].map((amount) => (
                <Button key={amount} variant="outline" size="sm" onClick={() => setRechargeAmount(String(amount))}>
                  ¥{amount}
                </Button>
              ))}
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
