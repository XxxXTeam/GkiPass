package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"gkipass/plane/internal/db/models"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

/*
GormUserService 基于 GORM 的用户服务
功能：管理用户的完整生命周期（注册/登录验证/CRUD/密码/角色/状态），
自动创建关联钱包，支持首用户自动设为管理员
*/
type GormUserService struct {
	db     *gorm.DB
	logger *zap.Logger
}

/*
NewGormUserService 创建基于 GORM 的用户服务
*/
func NewGormUserService(db *gorm.DB) *GormUserService {
	return &GormUserService{
		db:     db,
		logger: zap.L().Named("gorm-user-service"),
	}
}

/* ==================== 密码安全工具 ==================== */

/*
ValidatePasswordStrength 校验密码强度
规则：至少8位，包含大写字母、小写字母、数字，可选特殊字符
*/
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("密码长度不能少于8位")
	}
	if len(password) > 72 {
		return fmt.Errorf("密码长度不能超过72位")
	}

	var hasUpper, hasLower, hasDigit bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("密码必须包含至少一个大写字母")
	}
	if !hasLower {
		return fmt.Errorf("密码必须包含至少一个小写字母")
	}
	if !hasDigit {
		return fmt.Errorf("密码必须包含至少一个数字")
	}
	return nil
}

/*
bcryptCost bcrypt 哈希成本因子
OWASP 推荐生产环境至少 12，兼顾安全性和性能。
DefaultCost=10 对现代硬件偏低，暴力破解成本不够。
*/
const bcryptCost = 12

/*
HashPassword 使用 bcrypt 对密码进行哈希
*/
func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("密码加密失败: %w", err)
	}
	return string(hashed), nil
}

/*
CheckPassword 验证密码是否匹配
*/
func CheckPassword(hashedPassword, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}

/* ==================== 用户名/邮箱校验 ==================== */

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

/*
ValidateUsername 校验用户名格式
规则：3-32位，仅允许字母、数字、下划线、连字符
*/
func ValidateUsername(username string) error {
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("用户名必须为3-32位，仅允许字母、数字、下划线、连字符")
	}

	/* 禁止保留用户名 */
	reserved := []string{"admin", "root", "system", "api", "www", "mail", "support"}
	lower := strings.ToLower(username)
	for _, r := range reserved {
		if lower == r {
			return fmt.Errorf("用户名 '%s' 为系统保留名称", username)
		}
	}
	return nil
}

/*
ValidateEmail 校验邮箱格式
*/
func ValidateEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("邮箱格式无效")
	}
	return nil
}

/* ==================== 注册 ==================== */

/*
RegisterRequest GORM 注册请求
*/
type GormRegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required"`
}

/*
RegisterResult 注册结果
*/
type RegisterResult struct {
	User        *models.User `json:"user"`
	IsFirstUser bool         `json:"is_first_user"`
}

/*
Register 用户注册
功能：校验输入 → 检查重复 → 首用户自动管理员 → 创建用户 + 钱包（事务）
*/
func (s *GormUserService) Register(req *GormRegisterRequest) (*RegisterResult, error) {
	/* 校验用户名 */
	if err := ValidateUsername(req.Username); err != nil {
		return nil, err
	}

	/* 校验邮箱 */
	if err := ValidateEmail(req.Email); err != nil {
		return nil, err
	}

	/* 校验密码强度 */
	if err := ValidatePasswordStrength(req.Password); err != nil {
		return nil, err
	}

	/* 检查用户名是否已存在 */
	var existCount int64
	if err := s.db.Model(&models.User{}).Where("username = ?", req.Username).Count(&existCount).Error; err != nil {
		s.logger.Error("检查用户名失败", zap.String("username", req.Username), zap.Error(err))
		return nil, fmt.Errorf("注册服务暂时不可用，请稍后重试")
	}
	if existCount > 0 {
		return nil, fmt.Errorf("用户名已存在")
	}

	/* 检查邮箱是否已存在 */
	if err := s.db.Model(&models.User{}).Where("email = ?", req.Email).Count(&existCount).Error; err != nil {
		s.logger.Error("检查邮箱失败", zap.String("email", req.Email), zap.Error(err))
		return nil, fmt.Errorf("注册服务暂时不可用，请稍后重试")
	}
	if existCount > 0 {
		return nil, fmt.Errorf("邮箱已被注册")
	}

	/* 检查是否为首个用户 */
	var totalUsers int64
	s.db.Model(&models.User{}).Count(&totalUsers)
	isFirstUser := totalUsers == 0

	role := models.RoleUser
	if isFirstUser {
		role = models.RoleAdmin
		s.logger.Info("首个用户注册，自动设置为管理员", zap.String("username", req.Username))
	}

	/* 加密密码 */
	hashedPwd, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	/* 事务：创建用户 + 钱包 */
	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPwd,
		Role:     role,
		Enabled:  true,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return fmt.Errorf("创建用户失败: %w", err)
		}

		/* 自动创建钱包 */
		wallet := &models.Wallet{
			UserID:       user.ID,
			Balance:      0,
			FrozenAmount: 0,
		}
		if err := tx.Create(wallet).Error; err != nil {
			return fmt.Errorf("创建钱包失败: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	s.logger.Info("用户注册成功",
		zap.String("userID", user.ID),
		zap.String("username", user.Username),
		zap.String("role", string(user.Role)))

	return &RegisterResult{User: user, IsFirstUser: isFirstUser}, nil
}

/* ==================== 登录验证 ==================== */

/*
Authenticate 验证用户凭据
功能：根据用户名查找用户 → 验证启用状态 → 验证密码 → 更新登录时间
返回认证通过的用户信息
*/
func (s *GormUserService) Authenticate(username, password string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户名或密码错误")
		}
		s.logger.Error("认证查询用户失败", zap.String("username", username), zap.Error(err))
		return nil, fmt.Errorf("用户名或密码错误")
	}

	if !user.Enabled {
		return nil, fmt.Errorf("账户已被禁用")
	}

	if !CheckPassword(user.Password, password) {
		return nil, fmt.Errorf("用户名或密码错误")
	}

	/* 更新最后登录时间 */
	s.db.Model(&user).Update("last_login", time.Now())

	return &user, nil
}

/* ==================== 用户 CRUD ==================== */

/*
GetUser 根据ID获取用户
*/
func (s *GormUserService) GetUser(id string) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, err
	}
	return &user, nil
}

/*
GetUserWithRelations 获取用户及其关联数据（订阅+钱包）
*/
func (s *GormUserService) GetUserWithRelations(id string) (*models.User, error) {
	var user models.User
	if err := s.db.
		Preload("Wallet").
		Preload("Subscriptions", "status = 'active'").
		Preload("Subscriptions.Plan").
		First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, err
	}
	return &user, nil
}

/*
ListUsers 列出所有用户（管理员用）
功能：返回用户列表，隐藏密码字段（通过json:"-"已处理）
*/
func (s *GormUserService) ListUsers(page, pageSize int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	s.db.Model(&models.User{}).Count(&total)

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	if err := s.db.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

/*
UpdatePassword 修改密码
功能：验证旧密码 → 校验新密码强度 → 更新
*/
func (s *GormUserService) UpdatePassword(userID, oldPassword, newPassword string) error {
	user, err := s.GetUser(userID)
	if err != nil {
		return err
	}

	if !CheckPassword(user.Password, oldPassword) {
		return fmt.Errorf("旧密码不正确")
	}

	if err := ValidatePasswordStrength(newPassword); err != nil {
		return err
	}

	hashedPwd, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	if err := s.db.Model(user).Update("password", hashedPwd).Error; err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}

	s.logger.Info("用户密码已更新", zap.String("userID", userID))
	return nil
}

/*
ToggleUserStatus 切换用户启用/禁用状态
*/
func (s *GormUserService) ToggleUserStatus(targetID, operatorID string) (bool, error) {
	if targetID == operatorID {
		return false, fmt.Errorf("不能禁用自己的账户")
	}

	user, err := s.GetUser(targetID)
	if err != nil {
		return false, err
	}

	newStatus := !user.Enabled
	if err := s.db.Model(user).Update("enabled", newStatus).Error; err != nil {
		return false, fmt.Errorf("更新用户状态失败: %w", err)
	}

	s.logger.Info("用户状态已更新",
		zap.String("userID", targetID),
		zap.Bool("enabled", newStatus))

	return newStatus, nil
}

/*
UpdateUserRole 更新用户角色
*/
func (s *GormUserService) UpdateUserRole(targetID, operatorID, newRole string) error {
	if targetID == operatorID {
		return fmt.Errorf("不能修改自己的角色")
	}

	/* 校验角色值 */
	role := models.UserRole(newRole)
	if role != models.RoleAdmin && role != models.RoleUser {
		return fmt.Errorf("无效的角色: %s，仅支持 admin 或 user", newRole)
	}

	user, err := s.GetUser(targetID)
	if err != nil {
		return err
	}

	if err := s.db.Model(user).Update("role", role).Error; err != nil {
		return fmt.Errorf("更新角色失败: %w", err)
	}

	s.logger.Info("用户角色已更新",
		zap.String("userID", targetID),
		zap.String("newRole", newRole))

	return nil
}

/*
DeleteUser 删除用户（软删除）
功能：不允许删除自己，关联数据通过 GORM 软删除保留
*/
func (s *GormUserService) DeleteUser(targetID, operatorID string) error {
	if targetID == operatorID {
		return fmt.Errorf("不能删除自己的账户")
	}

	if _, err := s.GetUser(targetID); err != nil {
		return err
	}

	if err := s.db.Delete(&models.User{}, "id = ?", targetID).Error; err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}

	s.logger.Info("用户已删除", zap.String("userID", targetID))
	return nil
}

/*
GetUserCount 获取用户总数
*/
func (s *GormUserService) GetUserCount() (int64, error) {
	var count int64
	err := s.db.Model(&models.User{}).Count(&count).Error
	return count, err
}
