import { apiGet, apiPost } from "./client"
import type { User } from "@/lib/types"

/*
  userApi 用户 API 服务
  功能：封装用户管理相关的所有 HTTP 请求
  对齐后端路由：/users/*、/auth/register
*/
export const userApi = {
  /* 管理员：获取用户列表 */
  list: () => apiGet<User[]>("/users"),

  /* 获取当前用户信息 */
  me: () => apiGet<User>("/users/me"),

  /* 获取当前用户权限 */
  permissions: () => apiGet("/users/permissions"),

  /* 注册新用户（管理员创建用户时也可使用） */
  create: (data: { username: string; password: string; email?: string }) =>
    apiPost<User>("/auth/register", data),

  /* 切换用户状态（启用/禁用） */
  toggleStatus: (id: string) => apiPost(`/users/${id}/status/update`),

  /* 更新用户角色 */
  updateRole: (id: string, role: string) => apiPost(`/users/${id}/role/update`, { role }),

  /* 删除用户 */
  delete: (id: string) => apiPost(`/users/${id}/delete`),

  /* 修改当前用户密码 */
  changePassword: (data: { old_password: string; new_password: string }) =>
    apiPost("/users/password/update", data),
}
