"use client"

import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts"
import { formatBytes } from "@/lib/utils"

/*
  TrafficChart 流量趋势图表
  功能：展示入站/出站流量随时间变化的趋势，使用 recharts 双色面积图
*/

interface TrafficPoint {
  time: string
  inbound: number
  outbound: number
}

interface TrafficChartProps {
  data: TrafficPoint[]
  height?: number
}

export function TrafficChart({ data, height = 260 }: TrafficChartProps) {
  if (data.length === 0) {
    return (
      <div className="flex items-center justify-center" style={{ height }}>
        <p className="text-sm text-muted-foreground">暂无流量数据</p>
      </div>
    )
  }

  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={data} margin={{ top: 5, right: 10, left: 0, bottom: 0 }}>
        <defs>
          <linearGradient id="colorIn" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#22c55e" stopOpacity={0.3} />
            <stop offset="95%" stopColor="#22c55e" stopOpacity={0} />
          </linearGradient>
          <linearGradient id="colorOut" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
            <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
        <XAxis
          dataKey="time"
          tick={{ fontSize: 11 }}
          className="text-muted-foreground"
          tickLine={false}
          axisLine={false}
        />
        <YAxis
          tick={{ fontSize: 11 }}
          className="text-muted-foreground"
          tickLine={false}
          axisLine={false}
          tickFormatter={(v) => formatBytes(v)}
          width={70}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: "hsl(var(--popover))",
            border: "1px solid hsl(var(--border))",
            borderRadius: "8px",
            fontSize: "12px",
          }}
          formatter={(value: number, name: string) => [
            formatBytes(value),
            name === "inbound" ? "入站" : "出站",
          ]}
          labelFormatter={(label) => `时间: ${label}`}
        />
        <Area
          type="monotone"
          dataKey="inbound"
          stroke="#22c55e"
          fillOpacity={1}
          fill="url(#colorIn)"
          strokeWidth={2}
          name="inbound"
        />
        <Area
          type="monotone"
          dataKey="outbound"
          stroke="#3b82f6"
          fillOpacity={1}
          fill="url(#colorOut)"
          strokeWidth={2}
          name="outbound"
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}

/*
  generateMockTrafficData 生成模拟流量数据
  功能：当后端尚未实现流量历史 API 时，生成演示用数据
*/
export function generateMockTrafficData(hours = 24): TrafficPoint[] {
  const now = new Date()
  return Array.from({ length: hours }, (_, i) => {
    const time = new Date(now.getTime() - (hours - 1 - i) * 3600000)
    return {
      time: `${time.getHours().toString().padStart(2, "0")}:00`,
      inbound: Math.floor(Math.random() * 500 * 1024 * 1024),
      outbound: Math.floor(Math.random() * 300 * 1024 * 1024),
    }
  })
}
