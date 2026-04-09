package database

import (
	"context"

	"gorm.io/gorm"
)

// 定义一个专属的 Context Key，防止和其他包的变量冲突
type txKey struct{}

// RunInTx 是一个全局的事务调度器
// 它开启一个事务，把事务实例塞进 ctx 里，然后传给你的业务函数
func RunInTx(db *gorm.DB, ctx context.Context, fn func(txCtx context.Context) error) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// 魔法：把 tx 放进 context 中
		txCtx := context.WithValue(ctx, txKey{}, tx)
		
		// 执行你的业务闭包（内部的 service 调用都会收到这个 txCtx）
		return fn(txCtx)
	})
}

// GetDB 自动智能判断：
// 如果 ctx 里有事务，就返回事务实例；如果没有，就返回默认的全局 db。
func GetDB(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	tx, ok := ctx.Value(txKey{}).(*gorm.DB)
	if ok {
		return tx // 在事务中，返回 tx
	}
	return defaultDB // 不在事务中，返回普通 db
}