package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

// SyncServiceSimplified 封装了碳积分同步服务的核心逻辑
type SyncServiceSimplified struct {
	db *gorm.DB
}

// NewSyncServiceSimplified 创建一个新的同步服务实例
func NewSyncServiceSimplified(db *gorm.DB) *SyncServiceSimplified {
	return &SyncServiceSimplified{db: db}
}

// HandleInitialSync 也需要设置 SyncStatus 为 Pending
func (s *SyncServiceSimplified) HandleInitialSync(ctx context.Context, userID string) error {
	snapshot, err := s.getBalanceSnapshotFromChain(userID)
	if err != nil {
		return errors.Wrap(err, "failed to get balance snapshot from chain")
	}
	// 3. 在数据库事务中创建记录
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		snapshotEntry := UserVersionSnapshot{
			UserID:         userID,
			CurrentVersion: 0,
			CurrentBalance: snapshot.Balance,
			UpdatedAt:      time.Now(),
		}
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&snapshotEntry)
		if result.Error != nil {
			return errors.Wrap(result.Error, "failed to create user version snapshot")
		}

		if result.RowsAffected == 0 {
			fmt.Printf("User snapshot for %s already exists. Initial sync is considered complete.\n", userID)
			return nil
		}

		// 2.2 只有在快照是新创建的情况下，才创建 version=0 的事件日志
		eventPayloadDetail := EventPayloadDetail{
			Addr:     snapshot.Addr,
			Balance:  snapshot.Balance,
			HashList: snapshot.HashList,
		}
		payloadBytes, err := json.Marshal(eventPayloadDetail)
		if err != nil {
			return errors.Wrap(err, "failed to marshal eventPayloadDetail")
		}

		initTxID := fmt.Sprintf("init_sync_tx_%s", uuid.New().String())

		logEntry := SyncEventLog{
			UserID:       userID,
			Version:      0,
			BalanceAfter: snapshot.Balance,
			ChangeValue:  0,
			EventType:    "INIT",
			TxID:         initTxID,
			ChainID:      "carbon_puhui_chain_v1",
			EventPayload: string(payloadBytes),
			SyncStatus:   StatusPending,
		}

		if err := tx.Create(&logEntry).Error; err != nil {
			return errors.Wrap(err, "failed to create initial sync event log")
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to handle initial sync event")
	}
	return err
}

// SendPendingEvents 是一个独立的服务，负责将 Pending/Failed 的事件按顺序发送给目标系统。
func (s *SyncServiceSimplified) SendPendingEvents(ctx context.Context) {
	fmt.Println("Starting SendPendingEvents task...")

	// 轮询查找需要发送的事件
	// 轮询间隔可以根据需求调整，例如每隔几秒检查一次
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("SendPendingEvents task received context done. Shutting down.")
			return
		case <-ticker.C:
			// 查找所有状态为 Pending 或 Failed (且重试次数未超限) 的事件
			var eventsToSend []SyncEventLog
			if err := s.db.WithContext(ctx).Model(&SyncEventLog{}).
				Where("sync_status = ? OR (sync_status = ? AND retry_count < ?)",
					StatusPending, StatusFailed, 5). // 假设最多重试5次
				Order("user_id ASC, version ASC").
				Find(&eventsToSend).Error; err != nil {
				fmt.Printf("Error finding pending events: %v\n", err)
				continue // 继续下次轮询
			}

			for _, event := range eventsToSend {
				fmt.Printf("Processing event for user %s, version %d, status %d\n", event.UserID, event.Version, event.SyncStatus)

				// 模拟发送给目标系统（例如调用数币app的API）
				// 这里的 CallTargetAPISuccess 通常会返回一个布尔值，表示目标是否成功处理
				// 实际场景中，你需要一个真正的 HTTP/RPC 调用
				callSuccess := s.simulateTargetAPICall(ctx, event)

				if callSuccess {
					// 如果目标系统成功处理，更新状态为 Success
					if err := s.updateEventStatus(ctx, event.ID, StatusSuccess, 0, ""); err != nil {
						fmt.Printf("Failed to update status for event %d to Success: %v\n", event.ID, err)
						// 记录错误，但继续处理下一个事件
					}
				} else {
					// 如果目标系统处理失败，增加重试次数，并更新状态为 Failed
					newRetryCount := event.RetryCount + 1
					errMsg := "Target API returned error or failed to process" // 模拟错误信息
					if err := s.updateEventStatus(ctx, event.ID, StatusFailed, newRetryCount, errMsg); err != nil {
						fmt.Printf("Failed to update status for event %d to Failed: %v\n", event.ID, err)
					}
					// 如果重试次数用尽，可以考虑标记为 Ignored 或报警
					if newRetryCount >= 5 {
						fmt.Printf("Event %d reached max retries, marking as Ignored.\n", event.ID)
						if err := s.updateEventStatus(ctx, event.ID, StatusIgnored, newRetryCount, "Max retries reached"); err != nil {
							fmt.Printf("Failed to update status for event %d to Ignored: %v\n", event.ID, err)
						}
					}
				}
			}
		}
	}
}

// updateEventStatus 是一个辅助方法，用于原子地更新事件的状态
func (s *SyncServiceSimplified) updateEventStatus(ctx context.Context, eventID uint64, status SyncStatus, retryCount uint8, errorMessage string) error {
	updateData := map[string]interface{}{
		"sync_status": status,
		"updated_at":  time.Now(),
	}
	if status == StatusFailed || status == StatusIgnored {
		updateData["retry_count"] = retryCount
		updateData["error_message"] = errorMessage
	} else {
		// 成功时，清空错误信息和重试计数
		updateData["retry_count"] = 0
		updateData["error_message"] = ""
	}

	return s.db.WithContext(ctx).Model(&SyncEventLog{}).Where("id = ?", eventID).Updates(updateData).Error
}

// simulateTargetAPICall 是一个模拟函数，模拟调用数币app的API
// 返回 true 表示目标成功处理，false 表示失败
func (s *SyncServiceSimplified) simulateTargetAPICall(ctx context.Context, event SyncEventLog) bool {
	fmt.Printf("Simulating API call to target system for Event ID %d (User: %s, Version: %d, Type: %s)\n", event.ID, event.UserID, event.Version, event.EventType)
	// 模拟随机的成功/失败
	if event.EventType == "EXCHANGE" && event.Version == 2 { // 假设 Version 2 的 EXCHANGE 偶尔会失败
		if time.Now().Nanosecond()%3 == 0 { // 模拟 1/3 的失败率
			fmt.Printf("   -> Simulation: Target API call FAILED for event %d\n", event.ID)
			return false
		}
	}
	fmt.Printf("   -> Simulation: Target API call SUCCESSFUL for event %d\n", event.ID)
	return true
}

func (s *SyncServiceSimplified) getBalanceSnapshotFromChain(userId string) (*EventPayloadDetail, error) {
	// 1. 通过 userId 获取 addr

	// 2. 从 chain 上获取余额快照

	// --- 实际的链交互逻辑在这里 ---
	return &EventPayloadDetail{
		Addr:    "xxx",
		Balance: 1250,
		HashList: map[string]int{
			"c00e42e3-c540-4501-8b89-668d41131cbb": 890,
		},
	}, nil
}
