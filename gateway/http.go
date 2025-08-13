package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const basicAddr = "http://127.0.0.1:30004/carbonIntegral"

const (
	getWalletMethod = "getWalletHistoryInfo"
	getAddrMethod   = "getUserAddr"
)

type GetWalletInfoReq struct {
	Address string `json:"address"`
}

type GetAddrInfoReq struct {
	UserId string `json:"userId"`
}

type Resp struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
}

func GetUserAddr(userId string) (interface{}, error) {
	req := &GetAddrInfoReq{
		UserId: userId,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	respBytes, err := post(fmt.Sprintf("%s/%s", basicAddr, getAddrMethod), reqBytes)
	if err != nil {
		return nil, err
	}

	resp := new(Resp)
	err = json.Unmarshal(respBytes, resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func GetWalletInfo(addr string) (interface{}, error) {
	req := &GetWalletInfoReq{
		Address: addr,
	}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	respBytes, err := post(fmt.Sprintf("%s/%s", basicAddr, getWalletMethod), reqBytes)
	if err != nil {
		return nil, err
	}

	resp := new(Resp)
	err = json.Unmarshal(respBytes, resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func post(url string, payload []byte) ([]byte, error) {
	client := &http.Client{
		Timeout: time.Second * 120,
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status code %d, err msg is %v\n", resp.StatusCode, resp.Status)
	}

	return io.ReadAll(resp.Body)
}
