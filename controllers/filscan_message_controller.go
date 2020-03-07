package controllers

import (
	"encoding/json"
	"filscan_lotus/filscanproto"
	"filscan_lotus/models"
	"gitlab.forceup.in/dev-proto/common"
	"golang.org/x/net/context"
)

var _ filscanproto.FilscanMessagesServer = (*FilscanMessages)(nil)

type FilscanMessages struct {
}

/**
BlockMessages(context.Context, *BlockMessagesReq) (*BlockMessagesResp, error)
AllMessagesMethods(context.Context, *common.Empty) (*AllMethodsResp, error)
MessageDetails(context.Context, *MessageDetailsReq) (*MessageDetailsResp, error)
WalletIdMessage(context.Context, *WalletIdMessageReq) (*WalletIdMessageResp, error)
MessageByAddress(context.Context, *MessageByAddressReq) (*MessageByAddressResp, error)
*/

func (this *FilscanMessages) BlockMessages(ctx context.Context, input *filscanproto.BlockMessagesReq) (*filscanproto.BlockMessagesResp, error) {
	resp := new(filscanproto.BlockMessagesResp)
	count := input.GetCount()
	beginx := input.GetBegindex()
	blockCid := input.GetBlockCid()
	method := input.GetMethod()
	if !CheckArg(count) {
		resp.Res = common.NewResult(5, "Value is null")
		return resp, nil
	}
	msgList, total, err := GetMsgByBlockMethodBeginxCount(count, beginx, blockCid, method)
	if err != nil {
		log("GetMsgByBlockMethodBeginxCount err =%v", err)
		resp.Res = common.NewResult(5, "search err")
		return resp, nil
	}
	resp.Res = common.NewResult(3, "success")
	var fm []*filscanproto.FilscanMessage
	for _, value := range msgList {
		fm = append(fm, FilscanResMsg2PtotoFilscanMessage(*value))
	}
	resp.Data = &filscanproto.BlockMessagesResp_Data{Total: int64(total), Msgs: fm}
	return resp, nil
}

func (this *FilscanMessages) MessagesMethods(ctx context.Context, input *filscanproto.MessagesMethodsReq) (*filscanproto.MethodsResp, error) {
	resp := new(filscanproto.MethodsResp)
	blockCids := input.GetCids()

	res, err := models.GetMsgMethodName(blockCids) //搜索全部或者一部分
	if err != nil {
		log("MessagesMethods err = %v", err.Error())
		resp.Res = common.NewResult(5, "search err")
		return resp, nil
	}
	if len(res) < 1 { //未搜索到
		resMap := make(map[string]string)
		// blockMsg := TipsetQueue0.MsgByBlockCid(blockCids)
		blockMsg := flscaner.List().FindMesage_blocks(blockCids)
		for _, value := range blockMsg {
			resMap[value.MethodName] = value.MethodName
		}
		for key, _ := range resMap {
			res = append(res, key)
		}
	}
	resp.Data = &filscanproto.MethodsResp_Data{Method: res}
	resp.Res = common.NewResult(3, "success")
	return resp, nil
}

func (this *FilscanMessages) MessageDetails(ctx context.Context, input *filscanproto.MessageDetailsReq) (*filscanproto.MessageDetailsResp, error) {
	resp := new(filscanproto.MessageDetailsResp)
	msgCid := input.GetMsgCid()
	if !CheckArg(msgCid) {
		resp.Res = common.NewResult(5, "Value is null")
		return resp, nil
	}
	msgRes, err := models.GetMsgByMsgCid(msgCid)
	blocks, err2 := models.GetBlockByMsg(msgCid)
	if err != nil || err2 != nil {
		log("MessageDetails err = %v", err.Error())
		resp.Res = common.NewResult(5, "search err")
		return resp, nil
	}
	if len(msgRes) < 1 {
		// filscanMsg := TipsetQueue.MsgByCid(msgCid)
		filscanMsg := flscaner.List().FindMesage_id(msgCid)

		tbyte, _ := json.Marshal(filscanMsg)
		var p models.FilscanMsgResult
		json.Unmarshal(tbyte, &p)
		msgRes = append(msgRes, &p)
	}
	msgDe := FilscanResMsg2PtotoFilscanMessage(*msgRes[0])
	if msgDe != nil && len(blocks) > 0 {
		msgDe.BlockCid = nil
		for _, value := range blocks {
			msgDe.BlockCid = append(msgDe.BlockCid, value.Cid)
		}
	}
	resp.Data = &filscanproto.MessageDetailsResp_Data{Msg: msgDe}
	resp.Res = common.NewResult(3, "success")
	return resp, nil
}

func (this *FilscanMessages) MessageByAddress(ctx context.Context, input *filscanproto.MessageByAddressReq) (*filscanproto.MessageByAddressResp, error) {
	resp := new(filscanproto.MessageByAddressResp)
	method := input.GetMethod()
	address := input.GetAddress()
	fromto := input.GetFromTo()
	count := input.GetCount()
	begindex := input.GetBegindex()
	if !CheckArg(address, count) {
		resp.Res = common.NewResult(5, "Value is null")
		return resp, nil
	}
	res, total, err := GetMsgByAddressFromToMethod(address, fromto, method, int(begindex), int(count))
	if err != nil {
		log("MessageByAddress err = %v", err.Error())
		resp.Res = common.NewResult(5, "search err")
		return resp, nil
	}
	var filscanRes []*filscanproto.FilscanMessage
	for _, value := range res {
		filscanRes = append(filscanRes, FilscanResMsg2PtotoFilscanMessage(*value))
	}
	resp.Data = &filscanproto.MessageByAddressResp_Data{Total: int64(total), Data: filscanRes}
	resp.Res = common.NewResult(3, "success")
	return resp, nil
}
