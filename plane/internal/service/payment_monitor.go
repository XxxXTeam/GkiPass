package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gkipass/plane/internal/db/dao"
	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/pkg/logger"

	"go.uber.org/zap"
)

// PaymentMonitorService 支付监听服务
type PaymentMonitorService struct {
	dao    *dao.DAO
	ctx    context.Context
	cancel context.CancelFunc
}

// NewPaymentMonitorService 创建支付监听服务
func NewPaymentMonitorService(d *dao.DAO) *PaymentMonitorService {
	ctx, cancel := context.WithCancel(context.Background())
	return &PaymentMonitorService{
		dao:    d,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动支付监听服务
func (s *PaymentMonitorService) Start() {
	logger.Info("支付监听服务启动")

	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			logger.Info("支付监听服务停止")
			return
		case <-ticker.C:
			s.checkPendingPayments()
		}
	}
}

// Stop 停止支付监听服务
func (s *PaymentMonitorService) Stop() {
	s.cancel()
}

// checkPendingPayments 检查待确认的支付
func (s *PaymentMonitorService) checkPendingPayments() {
	monitors, err := s.dao.ListPendingMonitors()
	if err != nil {
		logger.Error("查询待监听订单失败", zap.Error(err))
		return
	}

	if len(monitors) == 0 {
		return
	}

	logger.Debug("检查待确认支付", zap.Int("count", len(monitors)))

	for i := range monitors {
		m := &monitors[i]
		// 检查是否超时
		if time.Now().After(m.ExpiresAt) {
			s.handleTimeout(m)
			continue
		}

		// 根据支付类型检查
		switch m.PaymentType {
		case "crypto":
			s.checkCryptoPayment(m)
		case "epay":
			s.checkEpayPayment(m)
		}
	}
}

// checkCryptoPayment 检查加密货币支付
func (s *PaymentMonitorService) checkCryptoPayment(monitor *models.PaymentMonitor) {
	// 获取加密货币配置
	config, err := s.dao.GetPaymentConfig("crypto_usdt")
	if err != nil || config == nil || !config.Enabled {
		return
	}

	var cryptoConfig struct {
		Network       string `json:"network"`
		Address       string `json:"address"`
		APIKey        string `json:"api_key"`
		CheckInterval int    `json:"check_interval"`
	}
	if err := json.Unmarshal([]byte(config.Config), &cryptoConfig); err != nil {
		logger.Error("解析加密货币配置失败", zap.Error(err))
		return
	}

	// 调用区块链浏览器API检查交易
	// 示例：TronScan API
	received, err := s.checkTRC20Balance(cryptoConfig.Address, cryptoConfig.APIKey)
	if err != nil {
		logger.Error("检查TRC20余额失败", zap.Error(err))
		return
	}

	// 检查是否收到足够金额
	if received >= monitor.ExpectedAmount {
		s.confirmPayment(monitor)
	}
}

// checkTRC20Balance 检查TRC20地址余额（示例）
func (s *PaymentMonitorService) checkTRC20Balance(address, apiKey string) (float64, error) {
	// 这里是示例实现，实际应该调用 TronGrid 或 TronScan API
	url := fmt.Sprintf("https://api.trongrid.io/v1/accounts/%s/transactions/trc20", address)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	if apiKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", apiKey)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	/* 限制外部 API 响应体最大 1MB，防止恶意响应导致 OOM */
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, err
	}

	// 解析响应并计算收到的金额
	var result struct {
		Data []struct {
			Value string `json:"value"`
			Type  string `json:"type"`
			To    string `json:"to"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	// 这里需要根据实际API响应格式解析
	// 示例：简单返回0，实际需要累加所有收款
	return 0, nil
}

// checkEpayPayment 检查易支付订单
func (s *PaymentMonitorService) checkEpayPayment(monitor *models.PaymentMonitor) {
	// 获取易支付配置
	config, err := s.dao.GetPaymentConfig("epay_default")
	if err != nil || config == nil || !config.Enabled {
		return
	}

	var epayConfig struct {
		APIURL      string `json:"api_url"`
		MerchantID  string `json:"merchant_id"`
		MerchantKey string `json:"merchant_key"`
	}
	if err := json.Unmarshal([]byte(config.Config), &epayConfig); err != nil {
		logger.Error("解析易支付配置失败", zap.Error(err))
		return
	}

	// 调用易支付查询接口
	status, err := s.queryEpayOrder(monitor.TransactionID, epayConfig)
	if err != nil {
		logger.Error("查询易支付订单失败", zap.Error(err))
		return
	}

	if status == "success" {
		s.confirmPayment(monitor)
	}
}

// queryEpayOrder 查询易支付订单状态
func (s *PaymentMonitorService) queryEpayOrder(orderID string, config struct {
	APIURL      string `json:"api_url"`
	MerchantID  string `json:"merchant_id"`
	MerchantKey string `json:"merchant_key"`
}) (string, error) {
	// 构建查询参数
	params := map[string]string{
		"act":          "order",
		"pid":          config.MerchantID,
		"out_trade_no": orderID,
	}

	// 生成签名
	sign := s.generateEpaySign(params, config.MerchantKey)
	params["sign"] = sign
	params["sign_type"] = "MD5"

	// 发送请求
	url := fmt.Sprintf("%s/api.php?act=order&pid=%s&out_trade_no=%s&sign=%s&sign_type=MD5",
		config.APIURL, config.MerchantID, orderID, sign)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	/* 限制外部 API 响应体最大 1MB，防止恶意响应导致 OOM */
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}

	var result struct {
		Code   int    `json:"code"`
		Msg    string `json:"msg"`
		Status string `json:"status"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Status, nil
}

// generateEpaySign 生成易支付签名
func (s *PaymentMonitorService) generateEpaySign(params map[string]string, key string) string {
	// 按照易支付规则生成签名
	str := fmt.Sprintf("act=%s&out_trade_no=%s&pid=%s%s",
		params["act"], params["out_trade_no"], params["pid"], key)

	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}

/*
confirmPayment 确认支付
功能：使用事务原子化完成 监听状态更新 → 钱包余额增加 → 交易状态完成，
防止并发竞态导致余额不一致。
*/
func (s *PaymentMonitorService) confirmPayment(monitor *models.PaymentMonitor) {
	logger.Info("支付确认",
		zap.String("transactionID", monitor.TransactionID),
		zap.String("paymentType", monitor.PaymentType),
		zap.Float64("amount", monitor.ExpectedAmount))

	err := s.dao.Transaction(func(txDAO *dao.DAO) error {
		/* 1. 更新监听状态 */
		if err := txDAO.UpdatePaymentMonitorStatus(monitor.ID, "confirmed", monitor.ConfirmCount+1); err != nil {
			return fmt.Errorf("更新监听状态失败: %w", err)
		}

		/* 2. 获取交易记录 */
		tx, err := txDAO.GetTransactionByID(monitor.TransactionID)
		if err != nil || tx == nil {
			return fmt.Errorf("查询交易记录失败: %w", err)
		}

		/* 3. 获取钱包（事务内加行锁） */
		wallet, err := txDAO.GetWalletByID(tx.WalletID)
		if err != nil || wallet == nil {
			return fmt.Errorf("获取钱包失败: %w", err)
		}

		/* 4. 原子更新钱包余额 */
		newBalance := wallet.Balance + monitor.ExpectedAmount
		if err := txDAO.UpdateWalletBalance(wallet.ID, newBalance, wallet.FrozenAmount); err != nil {
			return fmt.Errorf("更新钱包余额失败: %w", err)
		}

		/* 5. 更新交易状态 */
		if err := txDAO.UpdateTransactionStatus(monitor.TransactionID, "completed", newBalance); err != nil {
			return fmt.Errorf("更新交易状态失败: %w", err)
		}

		logger.Info("充值成功",
			zap.String("walletUserID", wallet.UserID),
			zap.Float64("amount", monitor.ExpectedAmount),
			zap.Float64("newBalance", newBalance))
		return nil
	})

	if err != nil {
		logger.Error("支付确认事务失败", zap.Error(err))
	}
}

// handleTimeout 处理超时订单
func (s *PaymentMonitorService) handleTimeout(monitor *models.PaymentMonitor) {
	logger.Warn("支付超时",
		zap.String("transactionID", monitor.TransactionID),
		zap.String("paymentType", monitor.PaymentType))

	// 更新状态为超时
	if err := s.dao.UpdatePaymentMonitorStatus(monitor.ID, "timeout", monitor.ConfirmCount); err != nil {
		logger.Error("更新监听状态失败", zap.Error(err))
		return
	}

	// 更新交易状态为失败
	if err := s.dao.UpdateTransactionStatus(monitor.TransactionID, "failed", 0); err != nil {
		logger.Error("更新交易状态失败", zap.Error(err))
	}
}
