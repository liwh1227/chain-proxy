package service

// 监听合约中碳积分的变动事件，并同步到接收方服务
import (
	"chain-proxy/chain"
	"chain-proxy/config"
	"chain-proxy/db"
	"chain-proxy/db/model"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

func HandleCollectEvent(ctx context.Context) error {
	var (
		err error
	)

	// 监听合约事件
	start, err := getListenStart()
	if err != nil {
		return err
	}

	evCh, err := chain.ListenContractEvents(ctx, start, -1, config.GetConfigInstance().ChainClient.ContractName, CarbonIntegralChangeTopic)
	if err != nil {
		return err
	}

	for {
		select {
		case res, ok := <-evCh:
			if !ok {
				return errors.New("event channel closed")
			}

			if res == nil {
				err = errors.New("nil event received")
				fmt.Println(err)
				continue
			}

			err = handleContractEvent(res)
			if err != nil {
				fmt.Println(err)
			}

		case <-ctx.Done():
			fmt.Printf("collect events recv ctx cancel signal, listen cc event task will close\n")
			return ctx.Err()
		}
	}
}

func getListenStart() (int64, error) {
	var start sql.NullInt64
	// 监听的起始高度
	// Q: 是否需要区分 collect or exchange？
	// A: 不需要区分，因为log表中的数据记录的block height有两个作用：
	//   a1: max height 是为了减少冗余监听，始终以最新的变化为起始进行监听；
	//   a2: 每个新插入到该表中的记录都是以auth中已授权用户的高度为准进行save，即始终会记录授权后该用户的余额变动；

	err := db.GetGormDb().
		Table(model.TableSyncEventLog).
		Select("max(block_height)").
		Where("contract_name = ?", config.GetConfigInstance().ChainClient.ContractName).
		Scan(&start).Error
	if err != nil {
		fmt.Println(err)
		return 0, err
	}

	if start.Valid {
		return start.Int64, nil
	}

	start.Int64 = config.GetConfigInstance().ChainClient.DefaultHeight

	return start.Int64, nil
}

func handleContractEvent(ev interface{}) error {
	bytes, err := json.Marshal(ev)
	if err != nil {
		return errors.Wrap(err, "failed to marshal unknown event")
	}

	evInfo := new(Event)
	err = json.Unmarshal(bytes, evInfo)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal unknown event into canonical Event struct")
	}

	if len(evInfo.EventData) == 0 {
		return errors.New("event data is empty")
	}

	evData := new(CollectEventInfo)
	err = json.Unmarshal([]byte(evInfo.EventData[0]), evData)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal event data into AddCarbonIntegralBatchRequest struct")
	}

	// 查询是否已授权
	var uar = new(model.UserAuth)
	err = db.GetGormDb().
		Table(model.TableUserAuth).
		Select("*").
		Where("addr = ?", evData.Address).
		Scan(&uar).Error
	if err != nil {
		return err
	}

	if uar.ID == 0 {
		// 说明未授权
		fmt.Printf("user %s has not auth", evData.Address)
		return nil
	}

	if uar.BlockHeight > evData.Height {
		// 该事件在余额授权同步之前发生，属于无效事件
		fmt.Printf("this event info is invalid %v", evData)
		return nil
	}

	sr := &model.SyncEventLog{
		UserId:       uar.UserId,
		BlockHeight:  evData.Height,
		BalanceAfter: evData.Balance,
		ChangeValue:  evData.ChangeValue,
		Topic:        CarbonIntegralChangeTopic,
		TxId:         evData.TxId,
		ContractName: config.GetConfigInstance().ChainClient.ContractName,
		SyncStatus:   int(StatusPending),
		RetryCount:   0,
	}

	err = db.GetGormDb().
		Table(model.TableSyncEventLog).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		Create(sr).Error
	if err != nil {
		return err
	}

	err = pushEvent(sr.ID, ev)

	return nil
}

// 3种方式：
// 1. 推送的消息队列，消息使用方自行订阅使用
// 2. 主动调用某个接口方法，将数据传送过去；
// 3. 定时任务间隔获取数据库数据并传送；
// pushEvent: Combined Tactical and Strategic Retry Logic
func pushEvent(id int, ev interface{}) error {
	// 1. 标记任务开始处理 (乐观更新)
	err := db.GetGormDb().
		Table(model.TableSyncEventLog).
		Where("id = ?", id).
		Update(model.SyncStatusCol, StatusSent).Error
	if err != nil {
		return fmt.Errorf("failed to mark event as sent for id %d: %w", id, err)
	}

	// 2. 进入内部的“战术重试”循环
	var handleErr error
	for attempt := 1; attempt <= 3; attempt++ {
		handleErr = mockHandleEvent(ev)
		if handleErr == nil {
			break // 跳出循环
		}
		if attempt < 3 {
			time.Sleep(50 * time.Millisecond)
		}
	}

	// 3. 根据内部重试循环的结果，更新最终状态
	tx := db.GetGormDb().
		Table(model.TableSyncEventLog).
		Where("id = ?", id)

	if handleErr == nil {
		fmt.Printf("[Strategic] Event id %d handled successfully.", id)
		updates := map[string]interface{}{
			model.SyncStatusCol: StatusSuccess,
			model.RetryCountCol: 0,
		}
		err = tx.Updates(updates).Error
		if err != nil {
			return fmt.Errorf("event handled, but failed to mark as success for id %d: %w", id, err)
		}
		return nil
	}

	// a. 原子地增加“战略失败”计数器
	updateResult := tx.Update(model.RetryCountCol, gorm.Expr(model.RetryCountCol, " + 1"))
	if updateResult.Error != nil {
		return fmt.Errorf("failed to increment strategic failure count for id %d: %w", id, updateResult.Error)
	}

	// b. 检查是否达到战略失败的阈值
	err = tx.Where(model.RetryCountCol+" >= ?", 3).
		Update(model.SyncStatusCol, StatusFailed).Error
	if err != nil {
		return fmt.Errorf("failed to mark event as failed after reaching max strategic retries for id %d: %w", id, err)
	}

	return fmt.Errorf("all %d tactical retries failed: %w", 3, handleErr)
}

func mockHandleEvent(ev interface{}) error {
	return nil
}

// 积分拆分的事件怎么处理？
