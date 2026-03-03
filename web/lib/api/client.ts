import axios, { type AxiosInstance, type AxiosRequestConfig } from "axios"
import { getToken, removeToken } from "@/lib/auth"

/*
  ApiClient API 客户端
  功能：封装 axios 实例，统一管理请求拦截、响应处理和 JWT 认证
*/

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

export interface ApiResponse<T = unknown> {
  success: boolean
  data?: T
  message?: string
  total?: number
}

export interface PaginationParams {
  page?: number
  page_size?: number
}

const client: AxiosInstance = axios.create({
  baseURL: `${API_BASE_URL}/api/v1`,
  timeout: 30000,
  headers: {
    "Content-Type": "application/json",
  },
})

/*
  简单重试机制：网络错误或 5xx 错误自动重试 1 次（延迟 1 秒）
  避免网络抖动导致用户操作失败，仅对 GET 请求重试（幂等安全）
*/
client.interceptors.response.use(undefined, async (error) => {
  const config = error.config
  if (
    config &&
    !config._retried &&
    config.method === "get" &&
    (!error.response || error.response.status >= 500)
  ) {
    config._retried = true
    await new Promise((r) => setTimeout(r, 1000))
    return client(config)
  }
  return Promise.reject(error)
})

/* 请求拦截：注入 JWT Token */
client.interceptors.request.use(
  (config) => {
    const token = getToken()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

/* 响应拦截：统一错误处理 */
client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401 && typeof window !== "undefined") {
      /* 登录/注册接口的 401 不触发跳转，由页面自行处理 */
      const url = error.config?.url || ""
      const isAuthEndpoint = url.includes("/auth/login") || url.includes("/auth/register")
      if (!isAuthEndpoint) {
        removeToken()
        const current = window.location.pathname
        const redirect = current !== "/login" ? `?redirect=${encodeURIComponent(current)}` : ""
        window.location.href = `/login${redirect}`
      }
    }
    return Promise.reject(error)
  }
)

export async function apiGet<T>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
  const res = await client.get<ApiResponse<T>>(url, config)
  return res.data
}

export async function apiPost<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
  const res = await client.post<ApiResponse<T>>(url, data, config)
  return res.data
}

export async function apiPut<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
  const res = await client.put<ApiResponse<T>>(url, data, config)
  return res.data
}

export async function apiDelete<T>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
  const res = await client.delete<ApiResponse<T>>(url, config)
  return res.data
}

export default client
