package chain

import (
	"chain-proxy/config"
	cmsdk "chainmaker.org/chainmaker/sdk-go/v2"
	"context"
	"fmt"
)

type BCClient struct {
	cmClient      *cmsdk.ChainClient
	chainId       string
	sdkConfigPath string
}

var Client = new(BCClient)

func newBCClient(chainId string, sdkConfigPath string) (*BCClient, error) {
	cli, err := cmsdk.NewChainClient(
		cmsdk.WithConfPath(sdkConfigPath),
	)
	if err != nil {
		return nil, err
	}

	return &BCClient{
		cmClient:      cli,
		chainId:       chainId,
		sdkConfigPath: sdkConfigPath,
	}, nil
}

func InitBCClient() error {
	var (
		err      error
		chainId  = config.GetConfigInstance().ChainClient.ChainId
		confPath = config.GetConfigInstance().ChainClient.SdkConfigPath
	)

	if Client.cmClient != nil {
		return nil
	}

	Client, err = newBCClient(chainId, confPath)
	if err != nil {
		return err
	}

	_, err = Client.cmClient.GetPoolStatus()
	if err != nil {
		return err
	}

	fmt.Println("init bc client success")

	return err
}

// ListenContractEvents 监听合约信息
func ListenContractEvents(ctx context.Context, start, end int64, contract, topic string) (<-chan interface{}, error) {
	return Client.cmClient.SubscribeContractEvent(ctx, start, end, contract, topic)
}
