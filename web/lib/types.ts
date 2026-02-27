/* 隧道相关类型 */
export interface Tunnel {
  id: string
  name: string
  description: string
  enabled: boolean
  created_by: string
  ingress_node_id: string
  egress_node_id: string
  ingress_group_id: string
  egress_group_id: string
  protocol: string
  ingress_protocol: string
  egress_protocol: string
  listen_port: number
  target_address: string
  target_port: number
  enable_encryption: boolean
  encryption_method: string
  rate_limit_bps: number
  max_connections: number
  idle_timeout: number
  load_balance_mode: string
  connection_count: number
  bytes_in: number
  bytes_out: number
  last_active: string
  created_at: string
  updated_at: string
  rules?: Rule[]
  targets?: TunnelTarget[]
}

export interface TunnelTarget {
  id: string
  tunnel_id: string
  host: string
  port: number
  weight: number
  enabled: boolean
}

export interface CreateTunnelRequest {
  name: string
  description?: string
  ingress_node_id?: string
  egress_node_id?: string
  ingress_group_id?: string
  egress_group_id?: string
  protocol?: string
  ingress_protocol?: string
  egress_protocol?: string
  listen_port: number
  target_address: string
  target_port: number
  enable_encryption?: boolean
  encryption_method?: string
  rate_limit_bps?: number
  max_connections?: number
  idle_timeout?: number
  load_balance_mode?: string
}

/* 规则相关类型 */
export interface Rule {
  id: string
  name: string
  description: string
  enabled: boolean
  priority: number
  version: number
  tunnel_id: string
  group_id: string
  protocol: string
  listen_port: number
  target_address: string
  target_port: number
  ingress_node_id: string
  egress_node_id: string
  created_at: string
  updated_at: string
}

/* 节点相关类型 */
export interface Node {
  id: string
  name: string
  description: string
  type: "entry" | "exit" | "relay"
  status: "online" | "offline" | "maintenance"
  ip: string
  port: number
  region: string
  provider: string
  os_info: string
  version: string
  cpu_usage: number
  memory_usage: number
  bandwidth_limit: number
  connection_count: number
  max_connections: number
  last_seen: string
  created_at: string
  updated_at: string
  groups?: NodeGroup[]
}

export interface NodeGroup {
  id: string
  name: string
  description: string
  type: string
  region: string
  enabled: boolean
  node_count: number
  nodes?: Node[]
  created_at: string
  updated_at: string
}

/* 用户相关类型 */
export interface User {
  id: string
  username: string
  email: string
  role: "admin" | "user" | "agent"
  status: "active" | "disabled" | "banned"
  avatar: string
  last_login: string
  plan_id: string
  plan_name: string
  traffic_used: number
  traffic_limit: number
  tunnel_count: number
  tunnel_limit: number
  created_at: string
  updated_at: string
}

export interface LoginRequest {
  username: string
  password: string
  captcha_id?: string
  captcha_code?: string
}

export interface LoginResponse {
  token: string
  user: User
}

/* 套餐相关类型（对齐后端 GORM models.Plan） */
export interface Plan {
  id: string
  name: string
  description: string
  price: number
  duration: number
  duration_unit: "month" | "year" | "permanent"
  traffic_limit: number
  speed_limit: number
  connection_limit: number
  rule_limit: number
  node_group_ids: string
  enabled: boolean
  sort_order: number
  created_at: string
  updated_at: string
}

/* 统计概览类型 */
export interface DashboardStats {
  total_tunnels: number
  active_tunnels: number
  total_nodes: number
  online_nodes: number
  total_users: number
  active_users: number
  traffic_in_today: number
  traffic_out_today: number
  total_connections: number
}

/* 协议转发策略类型（对齐后端 Policy + PolicyConfig） */
export interface Policy {
  id: string
  name: string
  type: "protocol" | "acl" | "routing"
  priority: number
  enabled: boolean
  config: PolicyConfig
  node_ids: string[]
  description: string
  created_at: string
  updated_at: string
}

export interface PolicyConfig {
  protocols?: string[]
  allow_ips?: string[]
  deny_ips?: string[]
  allow_ports?: number[]
  deny_ports?: number[]
  target_host?: string
  target_port?: number
}

/* 监控数据类型 */
export interface MonitoringStats {
  nodes: NodeMetrics[]
  system: SystemMetrics
}

export interface NodeMetrics {
  node_id: string
  node_name: string
  cpu_usage: number
  memory_usage: number
  disk_usage: number
  bandwidth_in: number
  bandwidth_out: number
  connection_count: number
  latency: number
  status: string
  uptime: number
  timestamp: string
}

export interface SystemMetrics {
  total_traffic_in: number
  total_traffic_out: number
  total_connections: number
  active_tunnels: number
  error_rate: number
  avg_latency: number
}

/* 容灾事件类型（对齐后端 FailoverEvent / FailoverEventReport） */
export interface FailoverEvent {
  id: string
  node_id: string
  tunnel_id: string
  event_type: "failover" | "recovery"
  from_group_id: string
  to_group_id: string
  reason: string
  failure_duration: number
  timestamp: string
}

export interface ActiveFailover {
  node_id: string
  tunnel_id: string
  event_type: string
  from_group_id: string
  to_group_id: string
  reason: string
  failure_duration: number
  timestamp: number
}

export interface GroupFailoverSummary {
  group_id: string
  total_events: number
  active_failovers: number
  affected_tunnels: string[]
}

/* 系统设置类型 */
export interface SystemSettings {
  site_name: string
  site_url: string
  admin_email: string
  allow_registration: boolean
  require_email_verification: boolean
  max_tunnels_per_user: number
  max_connections_per_tunnel: number
  default_rate_limit: number
  session_timeout: number
  enable_captcha: boolean
  smtp_host: string
  smtp_port: number
  smtp_user: string
  smtp_from: string
  enable_notifications: boolean
}
