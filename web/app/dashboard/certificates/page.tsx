"use client"

import { useEffect, useState } from "react"
import { Shield, Plus, Download, Ban } from "lucide-react"
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
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { toast } from "sonner"
import { certificateApi, type Certificate } from "@/lib/api/certificates"

export default function CertificatesPage() {
  const [certs, setCerts] = useState<Certificate[]>([])
  const [loading, setLoading] = useState(true)
  const [generateOpen, setGenerateOpen] = useState(false)
  const [leafForm, setLeafForm] = useState({ common_name: "", node_id: "" })
  const [revokeId, setRevokeId] = useState<string | null>(null)

  const fetchCerts = async () => {
    try {
      const res = await certificateApi.list()
      if (res.success && res.data) setCerts(res.data)
    } catch { setCerts([]) }
    finally { setLoading(false) }
  }

  useEffect(() => { fetchCerts() }, [])

  const handleGenerateCA = async () => {
    try {
      await certificateApi.generateCA()
      toast.success("CA 证书已生成")
      fetchCerts()
    } catch { toast.error("生成失败") }
  }

  const handleGenerateLeaf = async () => {
    if (!leafForm.common_name) { toast.error("请填写通用名称"); return }
    try {
      await certificateApi.generateLeaf(leafForm)
      toast.success("叶子证书已生成")
      setGenerateOpen(false)
      setLeafForm({ common_name: "", node_id: "" })
      fetchCerts()
    } catch { toast.error("生成失败") }
  }

  const handleRevoke = async () => {
    if (!revokeId) return
    try {
      await certificateApi.revoke(revokeId)
      toast.success("证书已吊销")
      setRevokeId(null)
      fetchCerts()
    } catch { toast.error("吊销失败") }
  }

  const caCerts = certs.filter((c) => c.type === "ca")
  const leafCerts = certs.filter((c) => c.type === "leaf")

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">证书管理</h1>
          <p className="text-muted-foreground">管理 CA 和节点证书</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={handleGenerateCA}>
            <Shield className="mr-2 h-4 w-4" /> 生成 CA
          </Button>
          <Button size="sm" onClick={() => setGenerateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" /> 签发证书
          </Button>
        </div>
      </div>

      {/* CA 证书 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">CA 证书</CardTitle>
          <CardDescription>根证书颁发机构</CardDescription>
        </CardHeader>
        <CardContent>
          {caCerts.length === 0 ? (
            <p className="py-6 text-center text-sm text-muted-foreground">暂无 CA 证书</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>主题</TableHead>
                  <TableHead>序列号</TableHead>
                  <TableHead>有效期</TableHead>
                  <TableHead>状态</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {caCerts.map((cert) => (
                  <TableRow key={cert.id}>
                    <TableCell className="font-medium">{cert.subject}</TableCell>
                    <TableCell className="font-mono text-xs">{cert.serial_number}</TableCell>
                    <TableCell className="text-xs">
                      {new Date(cert.not_before).toLocaleDateString("zh-CN")} ~ {new Date(cert.not_after).toLocaleDateString("zh-CN")}
                    </TableCell>
                    <TableCell>
                      <Badge variant={cert.is_revoked ? "destructive" : "default"}>
                        {cert.is_revoked ? "已吊销" : "有效"}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* 叶子证书 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">节点证书</CardTitle>
          <CardDescription>已签发的节点 TLS 证书</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-3">
              {Array.from({ length: 2 }).map((_, i) => (
                <div key={i} className="h-12 animate-pulse rounded bg-muted" />
              ))}
            </div>
          ) : leafCerts.length === 0 ? (
            <p className="py-6 text-center text-sm text-muted-foreground">暂无节点证书</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>主题</TableHead>
                  <TableHead>颁发者</TableHead>
                  <TableHead>序列号</TableHead>
                  <TableHead>有效期</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead className="w-24" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {leafCerts.map((cert) => (
                  <TableRow key={cert.id}>
                    <TableCell className="font-medium">{cert.subject}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{cert.issuer}</TableCell>
                    <TableCell className="font-mono text-xs">{cert.serial_number}</TableCell>
                    <TableCell className="text-xs">
                      {new Date(cert.not_before).toLocaleDateString("zh-CN")} ~ {new Date(cert.not_after).toLocaleDateString("zh-CN")}
                    </TableCell>
                    <TableCell>
                      <Badge variant={cert.is_revoked ? "destructive" : "default"}>
                        {cert.is_revoked ? "已吊销" : "有效"}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        <Button
                          variant="ghost" size="icon" className="h-7 w-7"
                          onClick={() => window.open(`${process.env.NEXT_PUBLIC_API_URL}/api/v1${certificateApi.downloadUrl(cert.id)}`, "_blank")}
                        >
                          <Download className="h-3 w-3" />
                        </Button>
                        {!cert.is_revoked && (
                          <Button
                            variant="ghost" size="icon" className="h-7 w-7 text-destructive"
                            onClick={() => setRevokeId(cert.id)}
                          >
                            <Ban className="h-3 w-3" />
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* 签发证书弹窗 */}
      <Dialog open={generateOpen} onOpenChange={setGenerateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>签发叶子证书</DialogTitle>
            <DialogDescription>为节点签发 TLS 证书</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-2">
              <Label>通用名称 (CN) *</Label>
              <Input
                value={leafForm.common_name}
                onChange={(e) => setLeafForm({ ...leafForm, common_name: e.target.value })}
                placeholder="如：node-01.gkipass.com"
              />
            </div>
            <div className="grid gap-2">
              <Label>关联节点 ID（可选）</Label>
              <Input
                value={leafForm.node_id}
                onChange={(e) => setLeafForm({ ...leafForm, node_id: e.target.value })}
                placeholder="节点 UUID"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setGenerateOpen(false)}>取消</Button>
            <Button onClick={handleGenerateLeaf}>签发</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 吊销确认弹窗 */}
      <Dialog open={!!revokeId} onOpenChange={() => setRevokeId(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认吊销</DialogTitle>
            <DialogDescription>吊销后该证书将不再有效，此操作不可撤销。</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRevokeId(null)}>取消</Button>
            <Button variant="destructive" onClick={handleRevoke}>确认吊销</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
