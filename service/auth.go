package service

import (
	"chain-proxy/db"
	"chain-proxy/db/model"
	"chain-proxy/gateway"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"sort"
)

func Auth(apiReq []byte) (*AuthResponse, error) {
	req := new(AuthRequest)
	err := json.Unmarshal(apiReq, req)
	if err != nil {
		return nil, err
	}

	var id int64
	err = db.GetGormDb().
		Table(model.TableUserAuth).
		Select("id").
		Where("user_id = ?", req.UserId).Scan(&id).Error
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	if id != 0 {
		fmt.Printf("user %v has been authenticated \n", req.UserId)
		return nil, err
	}

	addr, err := getAddrByUserId(req.UserId)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// 获取钱包历史状态数据
	wresp, err := getUserWalletInfo(addr)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	walletinfo, err := getLatestWalletInfo(wresp)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	r := &model.UserAuth{
		UserId:      req.UserId,
		Addr:        addr,
		Dcid:        req.Dcid,
		Balance:     int64(walletinfo.Total),
		BlockHeight: int64(walletinfo.BlockHeight),
	}

	err = db.GetGormDb().Table(model.TableUserAuth).Create(r).Error
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &AuthResponse{
		Balance: int64(walletinfo.Total),
		Wallet:  walletinfo.WalletInfo,
	}, nil
}

func getAddrByUserId(userId string) (string, error) {
	// 从 gateway 获取该 addr 信息
	resp, err := gateway.GetUserAddr(userId)
	if err != nil {
		return "", err
	}

	res := new(GetUserAddrResp)
	resBytes, err := json.Marshal(resp)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(resBytes, res)
	if err != nil {
		return "", err
	}

	return res.Addr, nil
}

// 获取用户钱包读写集历史状态 from gateway
func getUserWalletInfo(addr string) (*WalletResp, error) {
	resp, err := gateway.GetWalletInfo(addr)
	if err != nil {
		return nil, err
	}

	resBytes, err := json.Marshal(resp)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	res := &WalletResp{}
	err = json.Unmarshal(resBytes, res)
	if err != nil {
		return nil, err
	}

	fmt.Printf("res is %#v\n", res)

	return res, nil
}

func getLatestWalletInfo(wresp *WalletResp) (*WalletInfoDetail, error) {
	if wresp == nil {
		return nil, errors.New("walletResp is nil")
	}

	sort.Slice(wresp.WalletHistoryInfo, func(i, j int) bool {
		return wresp.WalletHistoryInfo[i].BlockHeight > wresp.WalletHistoryInfo[j].BlockHeight
	})

	return wresp.WalletHistoryInfo[0], nil
}
