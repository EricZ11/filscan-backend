package controllers

import (
	"context"
	"filscan_lotus/filscanproto"
	"filscan_lotus/models"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs-force-community/common"
	"strconv"
)

var _ filscanproto.FilscanAccountServer = (*FilscanAccount)(nil)

type FilscanAccount struct {
}

/**
WalletById(context.Context, *WalletByIdReq) (*WalletByIdResp, error)
MinerById(context.Context, *MinerByIdReq) (*MinerByIdResp, error)
ActorById(context.Context, *ActorByIdReq) (*ActorByIdResp, error)
AddressById(context.Context, *AddressByIdReq) (*AddressByIdResp, error)
AccountList(context.Context, *ListReq) (*AccountResq, error)
*/
func (this *FilscanAccount) WalletById(ctx context.Context, input *filscanproto.WalletByIdReq) (*filscanproto.WalletByIdResp, error) {
	resp := new(filscanproto.WalletByIdResp)
	return resp, nil
}

func (this *FilscanAccount) MinerById(ctx context.Context, input *filscanproto.MinerByIdReq) (*filscanproto.MinerByIdResp, error) {
	resp := new(filscanproto.MinerByIdResp)
	return resp, nil
}

func (this *FilscanAccount) ActorById(ctx context.Context, input *filscanproto.ActorByIdReq) (*filscanproto.ActorByIdResp, error) {
	resp := new(filscanproto.ActorByIdResp)
	address := input.GetActorId()
	res, err := models.GetActorByAddress(address)
	if err != nil {
		log("err=%v", err)
		resp.Res = common.NewResult(5, "search err")
		return resp, nil
	}
	if res == nil || len(res.Address) < 1 || len(res.Actor.Code.Str) < 1 {
		resp.Res = common.NewResult(3, "success")
		return resp, nil
	}
	actor := AccountResult2ProtoAccount(res)

	wList := []string{}
	if res.IsOwner {
		wList, err = models.MinerListByWalletAddr(address)
		if err != nil {
			log("err=%v", err)
			resp.Res = common.NewResult(5, "search err")
			return resp, nil
		}
	}
	miner := new(filscanproto.FilscanMiner)
	if res.IsMiner { //miner info
		m, err := models.MinerByAddress(address)
		if err != nil {
			log("err=%v", err)
			resp.Res = common.NewResult(5, "search err")
			return resp, nil
		}
		if m != nil {
			miner.OwnerAddress = m.WalletAddr
			miner.PeerId = m.PeerId
			miner.SectorSize = m.SectorSize
			if m.Power != nil {
				miner.Power = m.Power.Int64()
			}
			miner.SectorNum = m.SectorCount
			if m.ProvingSectorSize != nil && m.ProvingSectorSize.Int64() > 0 && m.SectorSize > 0 {
				miner.ProvingSectorNum = int64(float64(m.ProvingSectorSize.Uint64() / m.SectorSize))
			}
		}
		//miner.FaultNum
	}
	resp.Data = &filscanproto.ActorByIdResp_Data{Data: actor, WorkList: wList, Miner: miner}
	resp.Res = common.NewResult(3, "success")
	return resp, nil
}

func (this *FilscanAccount) AddressById(ctx context.Context, input *filscanproto.AddressByIdReq) (*filscanproto.AddressByIdResp, error) {
	resp := new(filscanproto.AddressByIdResp)
	return resp, nil
}

func (this *FilscanAccount) AccountList(ctx context.Context, input *filscanproto.ListReq) (*filscanproto.AccountResq, error) {
	resp := new(filscanproto.AccountResq)
	begindex := input.GetBegindex()
	count := input.GetCount()
	if !CheckArg(count) {
		res := &common.Result{Code: 5, Msg: "Missing required parameters"}
		resp.Res = res
		return resp, nil
	}
	res, total, err := models.GetAccountBySort(int(begindex), int(count))
	if err != nil {
		log("err=%v", err)
		resp.Res = common.NewResult(5, "search err")
		return resp, nil
	}
	var acountList []*filscanproto.FilscanActor
	for _, value := range res {
		total, err := models.GetMsgByAddressFromToMethodNameCount(value.Address, "", "")
		// queueMsgList := TipsetQueue.MsgByAddressFromToMethodName(value.Address, "", "")
		queueMsgList := flscaner.List().FindMesage_address(value.Address, "", "")
		if err != nil {
			resp.Res = common.NewResult(5, "search err")
			return resp, nil
		}
		ac := AccountResult2ProtoAccount(value)
		ac.Messages = uint64(total + len(queueMsgList))
		acountList = append(acountList, ac)
	}
	sumBalance, err := models.GetAccountSumBalance()
	if err != nil {
		log("err=%v", err)
		resp.Res = common.NewResult(5, "search err")
		return resp, nil
	}
	s1 := strconv.FormatFloat(sumBalance, 'f', -1, 64)
	bigI, err := types.BigFromString(s1)
	resp.Res = common.NewResult(3, "success")
	resp.Data = &filscanproto.AccountResq_Data{Accounts: acountList, Total: uint64(total), TotalFil: types.FIL(bigI).String()}
	return resp, nil
}

func (this *FilscanAccount) WorkListByAddress(ctx context.Context, input *filscanproto.AddressReq) (*filscanproto.AddressResq, error) {
	resp := new(filscanproto.AddressResq)
	address := input.GetAddress()
	if !CheckArg(address) {
		res := &common.Result{Code: 5, Msg: "Missing required parameters"}
		resp.Res = res
		return resp, nil
	}

	wList, err := models.MinerListByWalletAddr(address)
	if err != nil {
		log("err=%v", err)
		resp.Res = common.NewResult(5, "search err")
		return resp, nil
	}
	resp.Data = &filscanproto.AddressResq_Data{Address: wList}
	resp.Res = common.NewResult(3, "success")
	return resp, nil
}

//message null
func AccountResult2ProtoAccount(account *models.AccountResult) *filscanproto.FilscanActor {
	res := new(filscanproto.FilscanActor)
	actor := new(filscanproto.Actor)
	actor.Code = account.Actor.Code.Str
	actor.Head = account.Actor.Head.Str

	balanceBigInt, _ := types.BigFromString(account.Actor.Balance)
	actor.Balance = types.FIL(balanceBigInt).String()
	actor.Nonce = account.Actor.Nonce
	res.Actor = actor
	res.Address = account.Address
	res.IsWallet = account.IsWallet
	res.IsMiner = account.IsMiner
	res.IsOwner = account.IsOwner
	res.IsStorageMiner = account.IsStorageMiner
	return res

}
