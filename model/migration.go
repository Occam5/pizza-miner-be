package model

// Migration 执行数据迁移
func Migration() {
	// 自动迁移模式
	DB.AutoMigrate(&User{})
	DB.AutoMigrate(&Frog{})
	DB.AutoMigrate(&PrizePool{})
	DB.AutoMigrate(&PoolParticipant{})
}
