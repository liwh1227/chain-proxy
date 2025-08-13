package main

import (
	"chain-proxy/api"
	"chain-proxy/chain"
	"chain-proxy/config"
	"chain-proxy/service"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	err = chain.InitBCClient()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	wp := service.NewWorkerPool(10, ctx, cancel)

	// api 服务
	err = wp.Submit(api.Run)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = wp.Submit(service.HandleCollectEvent)
	if err != nil {
		fmt.Println(err)
		return
	}

	wp.Start()

	// 捕捉系统quit信号
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	<-signals

	wp.Stop()
}
