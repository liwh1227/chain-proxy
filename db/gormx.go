package db

import (
	"chain-proxy/config"
	"database/sql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"sync"
	"time"
)

type Option func(gormdb *GormDbOnce)

type GormDbOnce struct {
	gormdb  *gorm.DB
	onceCli sync.Once
	sqlDb   *sql.DB
	err     error
}

var dbIns = new(GormDbOnce)

// NewGormDB 初始化并获取gorm db实例
func NewGormDB(DSN string, opts ...Option) (*GormDbOnce, error) {
	dbIns.initGormDB(DSN, opts...)
	return dbIns, dbIns.err
}

// GetGormDb
func GetGormDb() *gorm.DB {
	if dbIns != nil && dbIns.gormdb != nil {
		return dbIns.gormdb
	}

	var err error
	conf := config.GetConfigInstance()
	dbIns, err = NewGormDB(conf.MySQL.DSN(),
		WithMaxIdleConns(conf.Gorm.MaxIdleConns),
		WithMaxLifetime(conf.Gorm.MaxLifetime),
		WithMaxOpenConns(conf.Gorm.MaxOpenConns))
	if err != nil {
		panic(err)
	}

	return dbIns.gormdb
}

// initGormDB 初始化gorm db相关
func (db *GormDbOnce) initGormDB(DSN string, opts ...Option) {
	db.onceCli.Do(func() {
		// 连接数据库
		gormDb, err := gorm.Open(mysql.Open(DSN), &gorm.Config{
			SkipDefaultTransaction: true,
		})
		if err != nil {
			db.err = err
			return
		}

		db.gormdb = gormDb
		sqlDb, err := gormDb.DB()
		if err != nil {
			db.err = err
			return
		}

		db.sqlDb = sqlDb

		// 校验是否可以ping通mysql service
		err = db.sqlDb.Ping()
		if err != nil {
			db.err = err
			return
		}

		// 执行option中的操作
		for _, opt := range opts {
			opt(dbIns)
		}
	})
}

// 连接池中空闲连接的最大数量 MaxIdleConns应该<= MaxOpenConns
func WithMaxIdleConns(maxIdleConns int) Option {
	return func(gormIns *GormDbOnce) {
		gormIns.sqlDb.SetMaxIdleConns(maxIdleConns)
	}
}

// 数据库最大连接数
func WithMaxOpenConns(maxOpenConns int) Option {
	return func(gormIns *GormDbOnce) {
		gormIns.sqlDb.SetMaxOpenConns(maxOpenConns)
	}
}

// 连接可复用最长时间
func WithMaxLifetime(maxLifetime int) Option {
	return func(gormIns *GormDbOnce) {
		gormIns.sqlDb.SetConnMaxLifetime(time.Duration(maxLifetime) * time.Millisecond)
	}
}
