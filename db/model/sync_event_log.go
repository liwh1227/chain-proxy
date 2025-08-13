package model

// 同步事件信息表
// 1. 同步每次从区块链捕获的用户事件信息
// 2. 该表记录了同步信息的结构、同步结果（状态）、重试次数、错误信息、 block height
// save：
// 1. 当用户首次授权后，获取用户钱包历史状态信息，
// 2. 当监听到已授权用户的变化动态时，解析区块：
//   2.1 若change height >= 用户 init 的 height，说明当前变化是发生在 init 之后，则存储；
//   2.2 若change height < 用户 init 的 height，说明当前变化发生在 init 之前，不存储；
// 用户余额同步后还存在的问题【极低概率】
// 用户 balance 在保存到该表之前，发生了变动（除非该用户在做该操作时，同步是进行收集或兑换操作）
// 如果要防止该情况的出现，可以加一步同步完后的校验接口（获取 gateway 余额？）

const TableSyncEventLog = "sync_event_log"

type SyncEventLog struct {
	CommonField
	UserId       string
	BlockHeight  int64
	BalanceAfter int64
	ChangeValue  int64
	Topic        string
	TxId         string
	ContractName string
	SyncStatus   int
	RetryCount   int
	ErrorMessage string
}
