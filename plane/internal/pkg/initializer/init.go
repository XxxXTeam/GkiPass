package initializer

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"gkipass/plane/internal/config"
	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/pkg/logger"
	"gkipass/plane/internal/service"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// IsFirstRun 检查是否首次运行
func IsFirstRun(configPath string) bool {
	_, err := os.Stat(configPath)
	return os.IsNotExist(err)
}

// InitConfig 初始化配置文件
func InitConfig(configPath string) error {
	//	logger.Info("首次启动，初始化配置文件...", zap.String("path", configPath))

	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 生成默认配置
	cfg := config.DefaultConfig()

	// 生成随机 JWT Secret
	cfg.Auth.JWTSecret = generateRandomSecret()

	// 保存配置文件
	if err := config.SaveConfig(cfg, configPath); err != nil {
		return fmt.Errorf("保存配置文件失败: %w", err)
	}

	logger.Info("✓ 配置文件已生成", zap.String("path", configPath))
	return nil
}

// InitDirectories 初始化必要的目录
func InitDirectories() error {
	dirs := []string{
		"./data",
		"./logs",
		"./certs",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录失败 %s: %w", dir, err)
		}
		//	logger.Info("✓ 目录已创建", zap.String("path", dir))
	}

	return nil
}

/* generateRandomSecret 生成 32 字节（256 位）随机密钥 */
func generateRandomSecret() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		/* 极端情况下回退到时间戳+PID（不应发生） */
		return fmt.Sprintf("gkipass-fallback-%d-%d", os.Getpid(), os.Getppid())
	}
	return hex.EncodeToString(bytes)
}

/*
InitAdmin 初始化默认管理员
功能：检查数据库中是否已有用户，若无则创建默认管理员账户，
并在控制台打印凭据。仅在首次启动（空数据库）时执行。
*/
func InitAdmin(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("查询用户数量失败: %w", err)
	}
	if count > 0 {
		return nil /* 已有用户，跳过 */
	}

	/* 生成随机密码（8 字节 = 16 位十六进制） */
	pwdBytes := make([]byte, 8)
	if _, err := rand.Read(pwdBytes); err != nil {
		return fmt.Errorf("生成随机密码失败: %w", err)
	}
	defaultPassword := hex.EncodeToString(pwdBytes)

	hashedPwd, err := service.HashPassword(defaultPassword)
	if err != nil {
		return fmt.Errorf("哈希密码失败: %w", err)
	}

	admin := &models.User{
		Username: "admin",
		Email:    "admin@localhost",
		Password: hashedPwd,
		Role:     models.RoleAdmin,
		Enabled:  true,
	}

	/* 事务：创建用户 + 钱包 */
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(admin).Error; err != nil {
			return err
		}
		wallet := &models.Wallet{UserID: admin.ID}
		return tx.Create(wallet).Error
	})
	if err != nil {
		return fmt.Errorf("创建管理员失败: %w", err)
	}

	/* 在控制台醒目打印凭据 */
	fmt.Println("")
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║           默认管理员账户已创建                   ║")
	fmt.Println("╠══════════════════════════════════════════════════╣")
	fmt.Printf("║  用户名: %-39s║\n", "admin")
	fmt.Printf("║  密  码: %-39s║\n", defaultPassword)
	fmt.Println("╠══════════════════════════════════════════════════╣")
	fmt.Println("║  ⚠ 请登录后立即修改密码！                       ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println("")

	logger.Info("✓ 默认管理员已创建", zap.String("username", "admin"))
	return nil
}

// PrintWelcome 打印欢迎信息
func PrintWelcome() {
	welcome := `
╔═══════════════════════════════════════════════════════╗
║                                                       ║
║   ██████╗ ██╗  ██╗██╗    ██████╗  █████╗ ███████╗███╗
║  ██╔════╝ ██║ ██╔╝██║    ██╔══██╗██╔══██╗██╔════╝████║
║  ██║  ███╗█████╔╝ ██║    ██████╔╝███████║███████╗╚═██║
║  ██║   ██║██╔═██╗ ██║    ██╔═══╝ ██╔══██║╚════██║  ██║
║  ╚██████╔╝██║  ██╗██║    ██║     ██║  ██║███████║  ██║
║   ╚═════╝ ╚═╝  ╚═╝╚═╝    ╚═╝     ╚═╝  ╚═╝╚══════╝  ╚═╝
║                                                       ║
║           Bidirectional Tunnel Control Plane         ║
║                      v2.0.0                           ║
║                                                       ║
╚═══════════════════════════════════════════════════════╝
`
	fmt.Println(welcome)
}
