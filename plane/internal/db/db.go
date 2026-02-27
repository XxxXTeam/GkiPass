package db

import (
	"gkipass/plane/internal/db/sqlite"
)

/*
Deprecated: DB 旧 SQLite 专用数据库封装，已由 GORM DAO 层完全替代。
Manager 不再引用此类型。保留仅为避免破坏 db/sqlite 包的编译。
后续可连同 db/sqlite/ 目录一起安全删除。
*/
type DB struct {
	SQLite *sqlite.SQLiteDB
}

// Deprecated: NewDB 已废弃，请使用 dao.New(gormDB)
func NewDB(sqliteDB *sqlite.SQLiteDB) *DB {
	return &DB{
		SQLite: sqliteDB,
	}
}

// Deprecated: Close 已废弃
func (db *DB) Close() error {
	if db.SQLite != nil {
		return db.SQLite.Close()
	}
	return nil
}
