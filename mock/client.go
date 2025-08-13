package mock

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"time"
)

// MockClient 模拟你的 Client.cmClient
type MockClient struct{}

type Event struct {
	Topic string
	Data  []byte
}

// SubscribeContractEvent 模拟实际的订阅方法
func (c *MockClient) SubscribeContractEvent(ctx context.Context, start, end int64, contract, topic string) (<-chan interface{}, error) {
	// 模拟一个可能发生的初始错误
	if contract == "error_contract" {
		return nil, errors.New("无效的合约地址")
	}

	// 创建一个用于返回事件的 channel
	eventChan := make(chan interface{})

	// 启动一个 goroutine 来模拟事件的产生和发送
	go func() {
		// 确保在 goroutine 结束时关闭 channel，这是至关重要的
		defer close(eventChan)

		// 模拟发送5个事件
		for i := 0; i < 5; i++ {
			event := &Event{
				Topic: "collect",
				Data:  []byte(fmt.Sprintf("事件 %d 来自合约 %s", i+1, contract)),
			}
			select {
			case <-ctx.Done(): // 如果外部上下文被取消，立即停止发送
				return
			case eventChan <- event: // 发送事件
				time.Sleep(5 * time.Second) // 模拟事件之间的时间间隔
			}
		}
	}()

	return eventChan, nil
}

// 这是你提供的函数
func ListenContractEvents(ctx context.Context, start, end int64, contract, topic string) (<-chan interface{}, error) {
	// 假设 Client.cmClient 是 MockClient 的一个实例
	var client = &MockClient{}
	return client.SubscribeContractEvent(ctx, start, end, contract, topic)
}
