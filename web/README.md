# GkiPass Web 前端

高性能双向隧道转发系统 - 管理控制面板前端

## 技术栈

- **Next.js 15** + **React 19** + **TypeScript**
- **Tailwind CSS** + **shadcn/ui** 组件库
- **recharts** 数据可视化
- **sonner** Toast 通知
- **Lucide** 图标库

## 快速开始

```bash
# 安装依赖
npm install

# 复制环境变量
cp .env.example .env.local

# 启动开发服务器
npm run dev

# 生产构建（需增大内存）
NODE_OPTIONS="--max-old-space-size=4096" npm run build
npm start
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `NEXT_PUBLIC_API_URL` | 后端 API 地址 | `http://localhost:8080` |

## 项目结构

```
web/
├── app/                          # Next.js App Router
│   ├── layout.tsx                # 根布局（主题+Toast）
│   ├── page.tsx                  # 首页（重定向到仪表盘）
│   ├── login/                    # 登录页
│   ├── error.tsx                 # 全局错误边界
│   ├── not-found.tsx             # 404 页面
│   └── dashboard/                # 仪表盘（受保护路由）
│       ├── layout.tsx            # 仪表盘布局（Sidebar+Header+AuthGuard+UserProvider）
│       ├── page.tsx              # 仪表盘首页（统计+公告+图表+最近隧道）
│       ├── tunnels/              # 隧道管理（搜索+CSV导出+批量操作）
│       ├── nodes/                # 节点管理（搜索+CSV导出）
│       ├── nodes/[id]/           # 节点详情（状态+证书+CK管理）
│       ├── node-groups/          # 节点组管理
│       ├── node-groups/[id]/config/ # 节点组配置（协议+端口+流量倍率）
│       ├── users/                # 用户管理（搜索+CSV导出）
│       ├── plans/                # 套餐管理
│       ├── acl/                  # 访问控制/策略管理
│       ├── monitoring/           # 系统监控（自动刷新）
│       ├── settings/             # 系统设置（分类Tab）
│       ├── announcements/        # 公告管理（管理员CRUD）
│       ├── subscription/         # 订阅与钱包（余额+充值+套餐）
│       ├── profile/              # 个人资料（修改密码）
│       ├── payment/              # 支付配置（管理员）
│       ├── traffic/              # 流量统计（时间过滤+CSV导出）
│       └── notifications/        # 通知中心（标记已读+删除）
├── components/
│   ├── layout/                   # 布局组件（Sidebar+Header）
│   ├── charts/                   # 图表组件（TrafficChart）
│   ├── ui/                       # shadcn/ui 组件
│   ├── auth-guard.tsx            # 客户端认证守卫
│   ├── loading-skeleton.tsx      # 加载骨架屏
│   └── theme-provider.tsx        # 主题提供者
├── lib/
│   ├── api/                      # API 服务层（18个文件）
│   │   ├── client.ts             # axios 客户端（JWT注入+401处理）
│   │   ├── index.ts              # 统一导出
│   │   └── [auth|dashboard|tunnels|nodes|users|plans|acl|monitoring|
│   │        settings|notifications|wallet|subscriptions|announcements|
│   │        payment|traffic].ts
│   ├── auth.ts                   # Token+Role cookie 管理
│   ├── user-context.tsx          # 共享用户状态（UserProvider+useUser）
│   ├── types.ts                  # TypeScript 类型定义
│   ├── utils.ts                  # 工具函数（cn+formatBytes）
│   └── export-csv.ts             # CSV 导出工具
├── middleware.ts                  # Next.js 中间件（token+role校验）
├── .env.example                  # 环境变量模板
└── .env.local                    # 本地环境变量
```

## 权限系统（三重保护）

1. **Next.js Middleware**：服务端 token cookie 校验 + 管理员页面 role cookie 校验
2. **AuthGuard**：客户端 localStorage token 检查
3. **Sidebar 动态菜单**：`useUser()` 获取角色，仅管理员显示管理菜单

## 功能特性

- **仪表盘**：统计卡片 + 活跃公告 + 流量趋势图表 + 最近隧道 + 30s 自动刷新
- **隧道管理**：搜索过滤 + CSV 导出 + 批量操作（全选/启停/删除）+ 节点组选择器
- **节点管理**：搜索过滤 + CSV 导出 + 详情页（证书管理 + Connection Key 管理）
- **节点组配置**：协议选择 + 端口范围 + 流量倍率
- **用户管理**：搜索过滤 + CSV 导出 + 角色管理 + 状态切换
- **套餐管理**：增删改查 + 启停控制
- **公告管理**：管理员 CRUD + 仪表盘/登录页展示
- **订阅与钱包**：余额展示 + 快捷充值 + 套餐订阅 + 交易记录
- **协议限制**：协议白名单策略（Checkbox 多选 8 种协议）+ 启停 + 下发节点
- **证书管理**：CA 生成 + 叶子证书签发 + 吊销 + 下载
- **系统设置**：分类 Tab 独立保存（基础/安全/通知）
- **流量统计**：汇总 + 明细 + 时间过滤 + CSV 导出
- **通知中心**：列表 + 标记已读 + 删除
- **支付配置**：渠道管理 + 启停切换 + 管理员手动充值
- **系统监控**：节点资源使用 + 30s 自动刷新