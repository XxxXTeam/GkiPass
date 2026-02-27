package dao

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

/*
DAO 统一 GORM 数据访问对象
功能：替代旧 SQLite 层，提供所有数据库操作的 GORM 实现。
所有 handler 通过 app.DAO 访问数据库。
*/
type DAO struct {
	DB     *gorm.DB
	logger *zap.Logger
}

/*
New 创建 DAO 实例
*/
func New(db *gorm.DB) *DAO {
	return &DAO{
		DB:     db,
		logger: zap.L().Named("dao"),
	}
}

/*
Transaction 在事务中执行多个数据库操作
功能：自动提交成功的事务，自动回滚失败的事务。
fn 内通过 txDAO 执行的所有操作共享同一事务。

	用法示例：
	err := d.Transaction(func(txDAO *DAO) error {
	    if err := txDAO.UpdateWalletBalance(...); err != nil { return err }
	    if err := txDAO.CreateTransaction(...); err != nil { return err }
	    return nil
	})
*/
/*
SanitizePagination 校正分页参数
功能：防止负值、零值和超大值导致的异常查询。
limit 范围 [1, maxLimit]，offset 最小为 0。
*/
func SanitizePagination(limit, offset, maxLimit int) (int, int) {
	if maxLimit <= 0 {
		maxLimit = 200
	}
	if limit <= 0 {
		limit = 20
	} else if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func (d *DAO) Transaction(fn func(txDAO *DAO) error) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		txDAO := &DAO{
			DB:     tx,
			logger: d.logger,
		}
		return fn(txDAO)
	})
}
