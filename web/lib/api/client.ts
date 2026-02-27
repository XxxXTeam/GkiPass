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
    if (error.response?.status === 401) {
      removeToken()
      if (typeof window !== "undefined") {
        window.location.href = "/login"
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
