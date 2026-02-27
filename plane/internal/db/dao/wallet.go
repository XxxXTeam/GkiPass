package dao

import (
	"gkipass/plane/internal/db/models"

	"gorm.io/gorm"
)

/* ==================== 钱包管理 ==================== */

/*
GetWalletByUserID 根据用户ID获取钱包
*/
func (d *DAO) GetWalletByUserID(userID string) (*models.Wallet, error) {
	var wallet models.Wallet
	if err := d.DB.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &wallet, nil
}

/*
GetOrCreateWallet 获取或创建用户钱包
*/
func (d *DAO) GetOrCreateWallet(userID string) (*models.Wallet, error) {
	wallet, err := d.GetWalletByUserID(userID)
	if err != nil {
		return nil, err
	}
	if wallet != nil {
		return wallet, nil
	}

	wallet = &models.Wallet{
		UserID:       userID,
		Balance:      0,
		FrozenAmount: 0,
	}
	if err := d.DB.Create(wallet).Error; err != nil {
		return nil, err
	}
	return wallet, nil
}

/*
UpdateWalletBalance 更新钱包余额
*/
func (d *DAO) UpdateWalletBalance(walletID string, balance, frozen float64) error {
	return d.DB.Model(&models.Wallet{}).Where("id = ?", walletID).
		Updates(map[string]interface{}{"balance": balance, "frozen_amount": frozen}).Error
}

/*
GetWalletByID 根据钱包ID获取钱包
*/
func (d *DAO) GetWalletByID(walletID string) (*models.Wallet, error) {
	var wallet models.Wallet
	if err := d.DB.First(&wallet, "id = ?", walletID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &wallet, nil
}

/*
CreateTransaction 创建交易记录
*/
func (d *DAO) CreateTransaction(tx *models.Transaction) error {
	return d.DB.Create(tx).Error
}

/*
ListTransactions 获取用户的交易记录
*/
func (d *DAO) ListTransactions(walletID string, page, pageSize int) ([]models.Transaction, int64, error) {
	var txs []models.Transaction
	var total int64

	q := d.DB.Model(&models.Transaction{}).Where("wallet_id = ?", walletID)
	q.Count(&total)

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&txs).Error; err != nil {
		return nil, 0, err
	}
	return txs, total, nil
}
