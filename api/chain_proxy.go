package api

import (
	"chain-proxy/service"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func AuthGroup(g *gin.Engine) {
	cg := g.Group("/chainProxy")
	{
		cg.POST("auth", Auth)
	}
}

func Auth(ctx *gin.Context) {
	req, err := ctx.GetRawData()
	if err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
		})
		return
	}

	if len(req) == 0 {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
		})
		return
	}

	resp, err := service.Auth(req)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": resp,
	})
}
