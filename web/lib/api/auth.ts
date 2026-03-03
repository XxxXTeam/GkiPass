import { apiGet, apiPost } from "./client"
import type { LoginRequest, LoginResponse, RegisterRequest, SetupStatus, User } from "@/lib/types"

/*
  authApi 认证 API 服务
  功能：封装登录、登出、注册、系统状态等认证相关请求
  对齐后端路由：/auth/* + /users/me + /users/password + /setup/status
*/
export const authApi = {
  login: (data: LoginRequest) => apiPost<LoginResponse>("/auth/login", data),

  register: (data: RegisterRequest) => apiPost<LoginResponse>("/auth/register", data),

  logout: () => apiPost("/auth/logout"),

  refresh: () => apiPost<LoginResponse>("/auth/refresh"),

  me: () => apiGet<User>("/users/me"),

  changePassword: (data: { old_password: string; new_password: string }) =>
    apiPost("/users/password/update", data),

  setupStatus: () => apiGet<SetupStatus>("/setup/status"),
}
