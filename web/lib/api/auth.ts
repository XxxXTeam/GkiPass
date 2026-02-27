import { apiGet, apiPost } from "./client"
import type { LoginRequest, LoginResponse, User } from "@/lib/types"

/*
  authApi 认证 API 服务
  功能：封装登录、登出、用户信息等认证相关请求
  对齐后端路由：/auth/* + /users/me + /users/password
*/
export const authApi = {
  login: (data: LoginRequest) => apiPost<LoginResponse>("/auth/login", data),

  logout: () => apiPost("/auth/logout"),

  refresh: () => apiPost<LoginResponse>("/auth/refresh"),

  me: () => apiGet<User>("/users/me"),

  changePassword: (data: { old_password: string; new_password: string }) =>
    apiPost("/users/password/update", data),
}
