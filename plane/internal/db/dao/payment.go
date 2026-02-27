package dao

import (
	"gkipass/plane/internal/db/models"
	"time"

	"gorm.io/gorm"
)

/*
ListPendingMonitors 获取所有待监听的支付记录
功能：查询状态为 monitoring 且未过期的支付监听记录
*/
func (d *DAO) ListPendingMonitors() ([]models.PaymentMonitor, error) {
	var monitors []models.PaymentMonitor
	if err := d.DB.Where("status = ? AND expires_at > ?", "monitoring", time.Now()).
		Order("created_at ASC").Find(&monitors).Error; err != nil {
		return nil, err
	}
	return monitors, nil
}

/*
UpdatePaymentMonitorStatus 更新支付监听状态
功能：更新支付监听记录的状态和确认次数
*/
func (d *DAO) UpdatePaymentMonitorStatus(id string, status string, confirmCount int) error {
	now := time.Now()
	return d.DB.Model(&models.PaymentMonitor{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        status,
			"confirm_count": confirmCount,
			"last_check_at": &now,
		}).Error
}

/*
CreatePaymentMonitor 创建支付监听记录
*/
func (d *DAO) CreatePaymentMonitor(monitor *models.PaymentMonitor) error {
	return d.DB.Create(monitor).Error
}

/*
GetPaymentMonitor 根据交易ID获取支付监听记录
*/
func (d *DAO) GetPaymentMonitor(transactionID string) (*models.PaymentMonitor, error) {
	var monitor models.PaymentMonitor
	if err := d.DB.Where("transaction_id = ?", transactionID).First(&monitor).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &monitor, nil
}

/*
GetTransactionByID 根据交易ID获取交易记录
功能：通过交易ID查找交易记录，用于支付确认时查询关联信息
*/
func (d *DAO) GetTransactionByID(id string) (*models.Transaction, error) {
	var tx models.Transaction
	if err := d.DB.First(&tx, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

/*
UpdateTransactionStatus 更新交易状态和余额
功能：将交易状态更新为完成/失败，并记录最终余额
*/
func (d *DAO) UpdateTransactionStatus(id string, status string, balance float64) error {
	return d.DB.Model(&models.Transaction{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":  status,
			"balance": balance,
		}).Error
}
