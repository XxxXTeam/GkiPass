import { apiGet, apiPost } from "./client"

/*
  钱包相关类型
*/
export interface WalletBalance {
  balance: number
  currency: string
}

export interface Transaction {
  id: string
  type: string
  amount: number
  description: string
  created_at: string
}

/*
  walletApi 钱包 API 服务
  功能：封装钱包余额查询、充值和交易记录
  对齐后端路由：/wallet/*、/payment/*
*/
export const walletApi = {
  /* 获取钱包余额 */
  balance: () => apiGet<WalletBalance>("/wallet/balance"),

  /* 获取交易记录 */
  transactions: () => apiGet<Transaction[]>("/wallet/transactions"),

  /* 创建充值订单 */
  createRechargeOrder: (amount: number) =>
    apiPost("/payment/recharge", { amount }),

  /* 查询订单状态 */
  queryOrder: (orderId: string) => apiGet(`/payment/orders/${orderId}`),
}
