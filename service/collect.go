package service

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
	"gorm.io/gorm/clause"
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

	evCh, err := chain.ListenContractEvents(ctx, start, -1, config.GetConfigInstance().ChainClient.ContractName, CarbonIntegralChangeEvType)
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
		Topic:        CollectEvType,
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

	return nil
}

// 3种方式：
// 1. 推送的消息队列，消息使用方自行订阅使用
// 2. 主动调用某个接口方法，将数据传送过去；
// 3. 定时任务间隔获取数据库数据并传送；
func pushEvent(ev []byte) error {
	fmt.Println("push event:", string(ev))
	return nil
}
