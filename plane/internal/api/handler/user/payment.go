package user

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"gkipass/plane/internal/db/dao"
	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/api/middleware"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/types"
	"gkipass/plane/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type PaymentHandler struct {
	app *types.App
}

func NewPaymentHandler(app *types.App) *PaymentHandler {
	return &PaymentHandler{app: app}
}

// CreateRechargeOrderRequest 创建充值订单请求
type CreateRechargeOrderRequest struct {
	Amount        float64 `json:"amount" binding:"required,min=10"`  // 充值金额，最低10元
	PaymentMethod string  `json:"payment_method" binding:"required"` // alipay/wechat/crypto
}

// CreateRechargeOrder 创建充值订单
func (h *PaymentHandler) CreateRechargeOrder(c *gin.Context) {
	var req CreateRechargeOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	// 验证支付方式
	validMethods := map[string]bool{
		"alipay": true,
		"wechat": true,
		"crypto": true,
		"usdt":   true,
		"manual": false, // 手动充值仅管理员可用
	}
	if !validMethods[req.PaymentMethod] {
		response.GinBadRequest(c, "Invalid payment method")
		return
	}

	// 创建充值订单
	orderID := uuid.New().String()
	order := &models.Order{
		UserID:      userID,
		Type:        "recharge",
		Status:      "pending",
		Amount:      req.Amount,
		PayMethod:   req.PaymentMethod,
		Description: fmt.Sprintf("充值 %.2f 元", req.Amount),
	}
	order.ID = orderID

	if err := h.app.DAO.CreateOrder(order); err != nil {
		logger.Error("创建充值订单失败", zap.Error(err))
		response.InternalError(c, "Failed to create recharge order")
		return
	}

	// 生成支付参数
	var paymentData interface{}
	switch req.PaymentMethod {
	case "alipay":
		paymentData = h.generateAlipayParams(orderID, req.Amount)
	case "wechat":
		paymentData = h.generateWechatParams(orderID, req.Amount)
	case "crypto", "usdt":
		paymentData = h.generateCryptoParams(orderID, req.Amount)
	}

	logger.Info("创建充值订单",
		zap.String("orderID", orderID),
		zap.String("userID", userID),
		zap.Float64("amount", req.Amount),
		zap.String("method", req.PaymentMethod))

	response.GinSuccess(c, gin.H{
		"order_id":       orderID,
		"amount":         req.Amount,
		"payment_method": req.PaymentMethod,
		"status":         "pending",
		"payment_data":   paymentData,
		"created_at":     order.CreatedAt,
	})
}

// QueryOrderStatus 查询订单状态
func (h *PaymentHandler) QueryOrderStatus(c *gin.Context) {
	orderID := c.Param("id")
	userID := middleware.GetUserID(c)

	order, err := h.app.DAO.GetOrderByUser(orderID, userID)
	if err != nil || order == nil {
		response.GinNotFound(c, "Order not found")
		return
	}

	response.GinSuccess(c, order)
}

// PaymentCallback 支付回调
func (h *PaymentHandler) PaymentCallback(c *gin.Context) {
	var req struct {
		OrderID       string  `json:"order_id" binding:"required"`
		PaymentMethod string  `json:"payment_method" binding:"required"`
		Amount        float64 `json:"amount" binding:"required"`
		TransactionID string  `json:"transaction_id"` // 第三方交易号
		Sign          string  `json:"sign"`           // 签名
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 验证签名
	if !h.verifyPaymentSign(req.OrderID, req.PaymentMethod, req.Amount, req.TransactionID, req.Sign) {
		response.GinBadRequest(c, "Invalid signature")
		return
	}

	// 查询订单
	order, err := h.app.DAO.GetOrder(req.OrderID)
	if err != nil || order == nil {
		response.GinNotFound(c, "Order not found")
		return
	}

	if order.Status != "pending" {
		response.GinSuccess(c, gin.H{"message": "Order already processed"})
		return
	}

	if order.Amount != req.Amount {
		logger.Warn("充值金额不匹配",
			zap.String("orderID", req.OrderID),
			zap.Float64("expected", order.Amount),
			zap.Float64("actual", req.Amount))
		response.GinBadRequest(c, "Amount mismatch")
		return
	}

	// 获取钱包
	wallet, err := h.app.DAO.GetWalletByUserID(order.UserID)
	if err != nil || wallet == nil {
		response.InternalError(c, "Wallet not found")
		return
	}

	// 更新钱包余额
	newBalance := wallet.Balance + req.Amount
	if err := h.app.DAO.UpdateWalletBalance(wallet.ID, newBalance, wallet.FrozenAmount); err != nil {
		logger.Error("更新钱包余额失败", zap.Error(err))
		response.InternalError(c, "Failed to update wallet balance")
		return
	}

	// 更新订单状态
	if err := h.app.DAO.UpdateOrderStatus(req.OrderID, "completed", req.TransactionID); err != nil {
		logger.Error("更新订单状态失败", zap.Error(err))
		response.InternalError(c, "Failed to update order status")
		return
	}

	// 如果是购买套餐，激活订阅
	if order.Type == "purchase" && order.PlanID != "" {
		logger.Info("套餐购买成功，需要激活订阅",
			zap.String("orderID", req.OrderID),
			zap.String("userID", order.UserID),
			zap.String("planID", order.PlanID))
	}

	logger.Info("支付成功",
		zap.String("orderID", req.OrderID),
		zap.String("userID", order.UserID),
		zap.String("type", order.Type),
		zap.Float64("amount", req.Amount),
		zap.Float64("newBalance", newBalance))

	response.GinSuccess(c, gin.H{
		"message":     "Payment successful",
		"order_id":    req.OrderID,
		"type":        order.Type,
		"new_balance": newBalance,
	})
}

// ManualRecharge 管理员手动充值
func (h *PaymentHandler) ManualRecharge(c *gin.Context) {
	var req struct {
		UserID      string  `json:"user_id" binding:"required"`
		Amount      float64 `json:"amount" binding:"required,gt=0"`
		Description string  `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	adminID := middleware.GetUserID(c)

	// 获取用户钱包
	wallet, err := h.app.DAO.GetWalletByUserID(req.UserID)
	if err != nil || wallet == nil {
		response.GinBadRequest(c, "User wallet not found")
		return
	}

	/* 余额更新 + 交易记录在同一事务中执行，保证数据一致性 */
	newBalance := wallet.Balance + req.Amount
	description := req.Description
	if description == "" {
		description = fmt.Sprintf("管理员充值 %.2f 元", req.Amount)
	}

	if err := h.app.DAO.Transaction(func(txDAO *dao.DAO) error {
		if err := txDAO.UpdateWalletBalance(wallet.ID, newBalance, wallet.FrozenAmount); err != nil {
			return err
		}
		tx := &models.Transaction{}
		tx.ID = uuid.New().String()
		tx.WalletID = wallet.ID
		tx.Type = "recharge"
		tx.Amount = req.Amount
		tx.Balance = newBalance
		tx.Description = description
		tx.OrderID = "admin_" + adminID
		return txDAO.CreateTransaction(tx)
	}); err != nil {
		logger.Error("管理员充值事务失败", zap.Error(err))
		response.InternalError(c, "Failed to process recharge")
		return
	}

	logger.Info("管理员手动充值",
		zap.String("adminID", adminID),
		zap.String("userID", req.UserID),
		zap.Float64("amount", req.Amount),
		zap.Float64("newBalance", newBalance))

	response.GinSuccess(c, gin.H{
		"user_id":     req.UserID,
		"amount":      req.Amount,
		"new_balance": newBalance,
	})
}

// 生成支付宝支付参数
func (h *PaymentHandler) generateAlipayParams(orderID string, amount float64) map[string]string {
	return map[string]string{
		"type":        "alipay",
		"order_id":    orderID,
		"amount":      fmt.Sprintf("%.2f", amount),
		"qr_code":     fmt.Sprintf("alipay://pay?order_id=%s&amount=%.2f", orderID, amount),
		"payment_url": fmt.Sprintf("/payment/alipay?order_id=%s", orderID),
		"expires_in":  "900", // 15分钟
	}
}

// 生成微信支付参数
func (h *PaymentHandler) generateWechatParams(orderID string, amount float64) map[string]string {
	return map[string]string{
		"type":        "wechat",
		"order_id":    orderID,
		"amount":      fmt.Sprintf("%.2f", amount),
		"qr_code":     fmt.Sprintf("weixin://wxpay/bizpayurl?order_id=%s&amount=%.2f", orderID, amount),
		"payment_url": fmt.Sprintf("/payment/wechat?order_id=%s", orderID),
		"expires_in":  "900",
	}
}

// 生成加密货币支付参数
func (h *PaymentHandler) generateCryptoParams(orderID string, amount float64) map[string]string {
	// 生成唯一的USDT-TRC20地址（示例）
	address := h.generateCryptoAddress(orderID)

	return map[string]string{
		"type":        "crypto",
		"order_id":    orderID,
		"amount":      fmt.Sprintf("%.2f", amount),
		"crypto":      "USDT-TRC20",
		"address":     address,
		"rate":        "7.2", // 汇率: 1 USDT = 7.2 CNY
		"usdt_amount": fmt.Sprintf("%.2f", amount/7.2),
		"expires_in":  "1800", // 30分钟
	}
}

// 生成加密货币地址（示例）
func (h *PaymentHandler) generateCryptoAddress(orderID string) string {
	salt := h.app.Config.Payment.CryptoSalt
	if salt == "" {
		salt = h.app.Config.Auth.JWTSecret /* 回退使用 JWT 密钥 */
	}
	hash := md5.Sum([]byte(orderID + salt))
	return "T" + hex.EncodeToString(hash[:])[:33] // TRC20地址格式
}

// verifyPaymentSign 验证支付回调签名
func (h *PaymentHandler) verifyPaymentSign(orderID, method string, amount float64, transactionID, sign string) bool {
	secret := h.app.Config.Payment.CallbackSecret
	if secret == "" {
		secret = h.app.Config.Auth.JWTSecret /* 回退使用 JWT 密钥 */
	}

	/* 构造签名字符串 */
	signStr := fmt.Sprintf("%s|%s|%.2f|%s|%s", orderID, method, amount, transactionID, secret)

	/* 计算 MD5 签名 */
	hash := md5.Sum([]byte(signStr))
	expectedSign := hex.EncodeToString(hash[:])

	return sign == expectedSign
}
