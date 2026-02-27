package dao

import (
	"time"

	"gkipass/plane/internal/db/models"

	"gorm.io/gorm"
)

/* ==================== 用户 CRUD ==================== */

/*
GetUser 根据ID获取用户
*/
func (d *DAO) GetUser(id string) (*models.User, error) {
	var user models.User
	if err := d.DB.First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

/*
GetUserByUsername 根据用户名获取用户
*/
func (d *DAO) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	if err := d.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

/*
GetUserByEmail 根据邮箱获取用户
*/
func (d *DAO) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := d.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

/*
CreateUser 创建用户
*/
func (d *DAO) CreateUser(user *models.User) error {
	return d.DB.Create(user).Error
}

/*
UpdateUser 更新用户
*/
func (d *DAO) UpdateUser(user *models.User) error {
	return d.DB.Save(user).Error
}

/*
UpdateUserLastLogin 更新最后登录时间
*/
func (d *DAO) UpdateUserLastLogin(id string) error {
	return d.DB.Model(&models.User{}).Where("id = ?", id).Update("last_login", time.Now()).Error
}

/*
ListUsers 列出用户（分页）
*/
func (d *DAO) ListUsers(page, pageSize int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	d.DB.Model(&models.User{}).Count(&total)

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if err := d.DB.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

/*
DeleteUser 删除用户（软删除）
*/
func (d *DAO) DeleteUser(id string) error {
	return d.DB.Delete(&models.User{}, "id = ?", id).Error
}

/*
GetUserCount 获取用户总数
*/
func (d *DAO) GetUserCount() (int64, error) {
	var count int64
	err := d.DB.Model(&models.User{}).Count(&count).Error
	return count, err
}

/*
GetUserByProvider 根据 OAuth 提供商和提供商用户ID获取用户
功能：用于 OAuth 登录时查找已绑定的用户
*/
func (d *DAO) GetUserByProvider(provider, providerID string) (*models.User, error) {
	var user models.User
	if err := d.DB.Where("provider = ? AND provider_id = ?", provider, providerID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

/*
GetUserWithRelations 获取用户及关联数据（订阅+钱包）
*/
func (d *DAO) GetUserWithRelations(id string) (*models.User, error) {
	var user models.User
	if err := d.DB.
		Preload("Wallet").
		Preload("Subscriptions", "status = 'active'").
		Preload("Subscriptions.Plan").
		First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
