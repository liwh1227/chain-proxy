package service

import "time"

// SyncStatus 定义了同步状态的枚举
type SyncStatus int8

const (
	StatusPending SyncStatus = 0 // 待发送
	StatusSent    SyncStatus = 1 // 已发送/待确认
	StatusSuccess SyncStatus = 2 // 已成功
	StatusFailed  SyncStatus = 3 // 失败
	StatusIgnored SyncStatus = 4 // 已忽略
)

const (
	CarbonIntegralChangeTopic = "cic_topic"
)

type AuthRequest struct {
	Dcid   string `json:"dcid"`
	UserId string `json:"userid"`
}

type AuthResponse struct {
	Balance int64   `json:"balance"`
	Wallet  *Wallet `json:"wallet"`
}

// SyncEventLog 对应 sync_event_log 表
type SyncEventLog struct {
	ID           uint64 `gorm:"primaryKey"`
	UserID       string `gorm:"type:varchar(255);index:idx_user_version,unique"`
	Version      uint64 `gorm:"index:idx_user_version,unique"`
	BalanceAfter int64
	ChangeValue  int64
	EventType    string     `gorm:"type:varchar(50);index"`
	TxID         string     `gorm:"type:varchar(255);uniqueIndex"`
	ChainID      string     `gorm:"type:varchar(100)"`
	EventPayload string     `gorm:"type:text"`
	SyncStatus   SyncStatus `gorm:"index"`
	RetryCount   uint8
	ErrorMessage string `gorm:"type:text"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserVersionSnapshot 对应 user_version_snapshot 表
type UserVersionSnapshot struct {
	UserID         string `gorm:"primaryKey;type:varchar(255)"`
	CurrentVersion uint64
	CurrentBalance int64
	UpdatedAt      time.Time
}

// EventPayloadDetail 是 event_payload 字段的结构化表示
type EventPayloadDetail struct {
	Addr     string         `json:"addr"`
	Balance  int64          `json:"balance"`
	HashList map[string]int `json:"hashList"`
}

type WalletInfoDetail struct {
	Key         string  `json:"key"`
	Field       string  `json:"field"`
	TxId        string  `json:"txId"`
	BlockHeight int     `json:"blockHeight"`
	Total       int     `json:"total"`
	WalletInfo  *Wallet `json:"walletInfo"`
}

type Wallet struct {
	// 未拆分积分
	IntegralMap map[string]int `json:"integralMap"`
	// 已经拆分积分
	SplitIntegralMap map[string]int `json:"splitIntegralMap"`
}

type WalletResp struct {
	WalletHistoryInfo []*WalletInfoDetail `json:"walletHistoryInfo"`
}

type Event struct {
	BlockHeight  int64    `json:"block_height"`
	ChainId      string   `json:"chain_id"`
	Topic        string   `json:"topic"`
	TxId         string   `json:"tx_id"`
	ContractName string   `json:"contract_name"`
	EventData    []string `json:"event_data" gorm:"type:longtext;serializer:json"`
}

type CollectEventInfo struct {
	Address     string `json:"address"`     // 用户钱包地址
	Height      int64  `json:"height"`      // 当前区块高度
	Balance     int64  `json:"balance"`     // 当前余额
	ChangeValue int64  `json:"changeValue"` // 改变值
	TxId        string `json:"txId"`        // 交易 id
}

type GetUserAddrResp struct {
	Addr string `json:"addr"`
}
