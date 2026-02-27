package user

import (
	"strconv"

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

// WalletHandler 钱包处理器
type WalletHandler struct {
	app *types.App
}

// NewWalletHandler 创建钱包处理器
func NewWalletHandler(app *types.App) *WalletHandler {
	return &WalletHandler{app: app}
}

// GetBalance 获取余额
func (h *WalletHandler) GetBalance(c *gin.Context) {
	userID := middleware.GetUserID(c)

	wallet, err := h.app.DAO.GetOrCreateWallet(userID)
	if err != nil {
		logger.Error("获取钱包失败", zap.Error(err))
		response.InternalError(c, "Failed to get wallet")
		return
	}

	response.GinSuccess(c, gin.H{
		"balance": wallet.Balance,
		"frozen":  wallet.FrozenAmount,
	})
}

// RechargeRequest 充值请求
type RechargeRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

// Recharge 充值 当前为debug版本，后续完善实现
func (h *WalletHandler) Recharge(c *gin.Context) {
	var req RechargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	wallet, err := h.app.DAO.GetOrCreateWallet(userID)
	if err != nil {
		response.InternalError(c, "Failed to get wallet")
		return
	}

	/* 余额更新 + 交易记录在同一事务中执行，保证数据一致性 */
	newBalance := wallet.Balance + req.Amount
	txID := uuid.New().String()

	if err := h.app.DAO.Transaction(func(txDAO *dao.DAO) error {
		if err := txDAO.UpdateWalletBalance(wallet.ID, newBalance, wallet.FrozenAmount); err != nil {
			return err
		}
		tx := &models.Transaction{}
		tx.ID = txID
		tx.WalletID = wallet.ID
		tx.Type = "recharge"
		tx.Amount = req.Amount
		tx.Balance = newBalance
		tx.Description = "充值"
		return txDAO.CreateTransaction(tx)
	}); err != nil {
		logger.Error("充值事务失败", zap.Error(err))
		response.InternalError(c, "Failed to process recharge")
		return
	}

	response.GinSuccess(c, gin.H{
		"balance":        newBalance,
		"transaction_id": txID,
	})
}

// ListTransactions 获取交易记录
func (h *WalletHandler) ListTransactions(c *gin.Context) {
	userID := middleware.GetUserID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	/* 分页参数边界保护 */
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	/* 先获取钱包ID，再查交易 */
	wallet, wErr := h.app.DAO.GetOrCreateWallet(userID)
	if wErr != nil {
		response.InternalError(c, "Failed to get wallet")
		return
	}
	transactions, total64, err := h.app.DAO.ListTransactions(wallet.ID, page, limit)
	total := int(total64)
	if err != nil {
		response.InternalError(c, "Failed to list transactions")
		return
	}

	response.GinSuccess(c, gin.H{
		"data":        transactions,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": (int(total) + limit - 1) / limit,
	})
}
