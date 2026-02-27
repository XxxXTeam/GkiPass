import { apiGet, apiPost } from "./client"
import type { Tunnel, CreateTunnelRequest } from "@/lib/types"

/*
  tunnelApi 隧道 API 服务
  功能：封装隧道相关的所有 HTTP 请求
*/
export const tunnelApi = {
  list: () => apiGet<Tunnel[]>("/tunnels/list"),

  get: (id: string) => apiGet<Tunnel>(`/tunnels/${id}`),

  create: (data: CreateTunnelRequest) => apiPost<Tunnel>("/tunnels/create", data),

  update: (id: string, data: Partial<CreateTunnelRequest>) =>
    apiPost<Tunnel>(`/tunnels/${id}/update`, data),

  delete: (id: string) => apiPost(`/tunnels/${id}/delete`),

  toggle: (id: string, enabled: boolean) =>
    apiPost<Tunnel>(`/tunnels/${id}/toggle`, { enabled }),
}
