"use client"

import { useEffect, useState } from "react"
import { useSearchParams, useRouter } from "next/navigation"
import {
  ArrowLeft, Shield, Key, RefreshCw, Download, Trash2, Plus, Copy,
  Wifi, WifiOff, Cpu, HardDrive, Clock,
} from "lucide-react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  Dialog, DialogContent, DialogDescription, DialogFooter,
  DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import { toast } from "sonner"
import { nodeApi } from "@/lib/api/nodes"
import type { Node } from "@/lib/types"

/*
  连接密钥类型
*/
interface ConnectionKey {
  id: string
  key: string
  node_id: string
  created_at: string
  expires_at: string
  is_active: boolean
}

/*
  证书信息类型
*/
interface CertInfo {
  has_cert: boolean
  subject: string
  issuer: string
  not_before: string
  not_after: string
  serial_number: string
}

/*
  节点详情页
  功能：通过 ?id=xxx 查询参数获取节点 ID，展示节点状态、证书管理和 Connection Key 管理
  路由：/dashboard/nodes/detail?id=xxx
*/
export default function NodeDetailPage() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const nodeId = searchParams.get("id") || ""

  const [node, setNode] = useState<Node | null>(null)
  const [cks, setCks] = useState<ConnectionKey[]>([])
  const [certInfo, setCertInfo] = useState<CertInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [revokeDialogOpen, setRevokeDialogOpen] = useState(false)
  const [revokingCkId, setRevokingCkId] = useState<string | null>(null)

  const fetchData = async () => {
    if (!nodeId) { setLoading(false); return }
    try {
      const [nodeRes, cksRes, certRes] = await Promise.allSettled([
        nodeApi.get(nodeId),
        nodeApi.listCKs(nodeId),
        nodeApi.certInfo(nodeId),
      ])
      if (nodeRes.status === "fulfilled" && nodeRes.value.success && nodeRes.value.data) {
        setNode(nodeRes.value.data)
      }
      if (cksRes.status === "fulfilled" && cksRes.value.success && cksRes.value.data) {
        setCks(cksRes.value.data as unknown as ConnectionKey[])
      }
      if (certRes.status === "fulfilled" && certRes.value.success && certRes.value.data) {
        setCertInfo(certRes.value.data as unknown as CertInfo)
      }
    } catch { /* 忽略 */ }
    finally { setLoading(false) }
  }

  useEffect(() => { fetchData() }, [nodeId]) // eslint-disable-line react-hooks/exhaustive-deps

  const handleGenerateCK = async () => {
    try {
      await nodeApi.generateCK(nodeId)
      toast.success("Connection Key 已生成")
      fetchData()
    } catch { toast.error("生成失败") }
  }

  const handleRevokeCK = async () => {
    if (!revokingCkId) return
    try {
      await nodeApi.revokeCK(revokingCkId)
      toast.success("Connection Key 已吊销")
      setRevokeDialogOpen(false)
      fetchData()
    } catch { toast.error("吊销失败") }
  }

  const handleGenerateCert = async () => {
    try {
      await nodeApi.generateCert(nodeId)
      toast.success("证书已生成")
      fetchData()
    } catch { toast.error("生成失败") }
  }

  const handleRenewCert = async () => {
    try {
      await nodeApi.renewCert(nodeId)
      toast.success("证书已续期")
      fetchData()
    } catch { toast.error("续期失败") }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text).then(() => toast.success("已复制"))
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="h-7 w-60 animate-pulse rounded bg-muted" />
        <div className="grid gap-4 md:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="h-32 animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      </div>
    )
  }

  if (!nodeId || !node) {
    return (
      <div className="flex flex-col items-center justify-center py-20">
        <p className="text-muted-foreground">{!nodeId ? "缺少节点 ID" : "节点不存在"}</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push("/dashboard/nodes")}>
          返回节点列表
        </Button>
      </div>
    )
  }

  const isOnline = node.status === "online"

  return (
    <div className="space-y-6">
      {/* 页头 */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" onClick={() => router.push("/dashboard/nodes")}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-2xl font-bold tracking-tight">{node.name}</h1>
              <Badge variant={isOnline ? "default" : "secondary"}>
                {isOnline ? "在线" : "离线"}
              </Badge>
            </div>
            <p className="text-muted-foreground">{node.ip}:{node.port} · {node.region || "未设置地区"}</p>
          </div>
        </div>
        <Button variant="outline" size="sm" onClick={fetchData}>
          <RefreshCw className="mr-2 h-4 w-4" /> 刷新
        </Button>
      </div>

      {/* 概览卡片 */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <div className={`rounded-lg p-2 ${isOnline ? "bg-green-500/10" : "bg-muted"}`}>
              {isOnline ? <Wifi className="h-4 w-4 text-green-500" /> : <WifiOff className="h-4 w-4 text-muted-foreground" />}
            </div>
            <div>
              <p className="text-xs text-muted-foreground">状态</p>
              <p className="font-medium">{isOnline ? "在线" : "离线"}</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <div className="rounded-lg bg-blue-500/10 p-2"><Cpu className="h-4 w-4 text-blue-500" /></div>
            <div>
              <p className="text-xs text-muted-foreground">CPU</p>
              <p className="font-medium">{(node.cpu_usage || 0).toFixed(1)}%</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <div className="rounded-lg bg-violet-500/10 p-2"><HardDrive className="h-4 w-4 text-violet-500" /></div>
            <div>
              <p className="text-xs text-muted-foreground">内存</p>
              <p className="font-medium">{(node.memory_usage || 0).toFixed(1)}%</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <div className="rounded-lg bg-amber-500/10 p-2"><Clock className="h-4 w-4 text-amber-500" /></div>
            <div>
              <p className="text-xs text-muted-foreground">连接数</p>
              <p className="font-medium">{node.connection_count || 0}</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 证书管理 */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2 text-base">
                <Shield className="h-4 w-4" /> 证书管理
              </CardTitle>
              <CardDescription>管理此节点的 TLS 证书</CardDescription>
            </div>
            <div className="flex gap-2">
              {certInfo?.has_cert ? (
                <>
                  <Button variant="outline" size="sm" onClick={handleRenewCert}>
                    <RefreshCw className="mr-1 h-3 w-3" /> 续期
                  </Button>
                  <Button variant="outline" size="sm" onClick={() => {
                    window.open(`${process.env.NEXT_PUBLIC_API_URL}/api/v1/nodes/${nodeId}/cert/download`, "_blank")
                  }}>
                    <Download className="mr-1 h-3 w-3" /> 下载
                  </Button>
                </>
              ) : (
                <Button size="sm" onClick={handleGenerateCert}>
                  <Plus className="mr-1 h-3 w-3" /> 生成证书
                </Button>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {certInfo?.has_cert ? (
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <p className="text-muted-foreground">主题</p>
                <p className="font-mono text-xs">{certInfo.subject}</p>
              </div>
              <div>
                <p className="text-muted-foreground">颁发者</p>
                <p className="font-mono text-xs">{certInfo.issuer}</p>
              </div>
              <div>
                <p className="text-muted-foreground">有效期起</p>
                <p>{certInfo.not_before ? new Date(certInfo.not_before).toLocaleDateString("zh-CN") : "--"}</p>
              </div>
              <div>
                <p className="text-muted-foreground">有效期止</p>
                <p>{certInfo.not_after ? new Date(certInfo.not_after).toLocaleDateString("zh-CN") : "--"}</p>
              </div>
            </div>
          ) : (
            <p className="py-4 text-center text-sm text-muted-foreground">暂无证书，请点击&ldquo;生成证书&rdquo;创建</p>
          )}
        </CardContent>
      </Card>

      {/* Connection Key 管理 */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2 text-base">
                <Key className="h-4 w-4" /> Connection Key
              </CardTitle>
              <CardDescription>管理节点连接密钥，用于节点客户端认证</CardDescription>
            </div>
            <Button size="sm" onClick={handleGenerateCK}>
              <Plus className="mr-1 h-3 w-3" /> 生成密钥
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {cks.length === 0 ? (
            <p className="py-4 text-center text-sm text-muted-foreground">暂无连接密钥</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>密钥</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead>过期时间</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead className="w-20" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {cks.map((ck) => (
                  <TableRow key={ck.id}>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <code className="text-xs font-mono bg-muted px-1.5 py-0.5 rounded max-w-[200px] truncate">
                          {ck.key}
                        </code>
                        <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => copyToClipboard(ck.key)}>
                          <Copy className="h-3 w-3" />
                        </Button>
                      </div>
                    </TableCell>
                    <TableCell className="text-xs">
                      {new Date(ck.created_at).toLocaleDateString("zh-CN")}
                    </TableCell>
                    <TableCell className="text-xs">
                      {ck.expires_at ? new Date(ck.expires_at).toLocaleDateString("zh-CN") : "永不过期"}
                    </TableCell>
                    <TableCell>
                      <Badge variant={ck.is_active ? "default" : "secondary"}>
                        {ck.is_active ? "有效" : "已吊销"}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {ck.is_active && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 text-xs text-destructive"
                          onClick={() => { setRevokingCkId(ck.id); setRevokeDialogOpen(true) }}
                        >
                          <Trash2 className="mr-1 h-3 w-3" /> 吊销
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* 吊销确认弹窗 */}
      <Dialog open={revokeDialogOpen} onOpenChange={setRevokeDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认吊销</DialogTitle>
            <DialogDescription>吊销后该密钥将无法用于节点连接，此操作不可撤销。</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRevokeDialogOpen(false)}>取消</Button>
            <Button variant="destructive" onClick={handleRevokeCK}>确认吊销</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
