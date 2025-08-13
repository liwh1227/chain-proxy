package model

import "time"

// 该表记录授权用户的信息：
// 1. 用户 id 完成主要逻辑；
// 2. addr 完成用户的链上信息查询；
// 3. 数币 dcid 标识；

const TableUserAuth = "user_auth"

type UserAuth struct {
	CommonField
	UserId      string `gorm:"unique"`
	Addr        string `gorm:"unique"`
	Dcid        string `gorm:"unique"` // 数币唯一标识
	BlockHeight int64
	Balance     int64
}

type CommonField struct {
	ID        int       `column:"primarykey"` // 主键ID
	CreatedAt time.Time // 创建时间
	UpdatedAt time.Time // 更新时间
}
