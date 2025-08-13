package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Option func(*gin.Engine)

var options = make([]Option, 0)

// register 注册app的路由配置
func register(opts ...Option) {
	options = append(options, opts...)
}

// Init 初始化
func groupInit() *gin.Engine {
	r := gin.Default()

	// http router engine
	register(AuthGroup)

	// 实例化http server
	for _, opt := range options {
		opt(r)
	}

	return r
}

func Run(ctx context.Context) error {
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", "127.0.0.1", 10086),
		Handler: groupInit(),
	}

	// 启动http server
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Println(err)
			return
		}
	}()

	<-ctx.Done()

	return server.Shutdown(context.Background())
}
