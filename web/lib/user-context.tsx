"use client"

import { createContext, useContext, useEffect, useState, type ReactNode } from "react"
import { authApi } from "@/lib/api/auth"
import { setRole } from "@/lib/auth"
import type { User } from "@/lib/types"

/*
  UserContext 用户上下文
  功能：在 Dashboard 布局内共享当前用户信息，避免 Header、Sidebar 等组件重复请求 /users/me
*/

interface UserContextValue {
  user: User | null
  loading: boolean
  refresh: () => void
}

const UserContext = createContext<UserContextValue>({
  user: null,
  loading: true,
  refresh: () => {},
})

export function UserProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchUser = () => {
    authApi.me().then((res) => {
      if (res.success && res.data) {
        setUser(res.data)
        setRole(res.data.role)
      }
    }).catch(() => {}).finally(() => setLoading(false))
  }

  useEffect(() => { fetchUser() }, [])

  return (
    <UserContext.Provider value={{ user, loading, refresh: fetchUser }}>
      {children}
    </UserContext.Provider>
  )
}

export function useUser() {
  return useContext(UserContext)
}
