package controllers

import (
	"context"
	"filscan_lotus/filscanproto"
	"filscan_lotus/models"
	"gitlab.forceup.in/dev-proto/common"
	"strconv"
)

var _ filscanproto.FilscanPeerServer = (*FilscanPeer)(nil)

type FilscanPeer struct {
}

/**
PeerById(context.Context, *PeerIdReq) (*PeerByIdResp, error)
ActivePeerCount(context.Context, *common.Empty) (*CountResq, error)
PeerMap(context.Context, *common.Empty) (*PeerMapResq, error)
*/

func (this *FilscanPeer) PeerById(ctx context.Context, input *filscanproto.PeerIdReq) (*filscanproto.PeerByIdResp, error) {
	resp := new(filscanproto.PeerByIdResp)
	peerId := input.GetPeerId()
	if !CheckArg(peerId) {
		resp.Res = common.NewResult(5, "Value is null")
		return resp, nil
	}
	miner, err := models.MinerByPeerId(peerId)
	if err != nil {
		log("GetMsgByBlockMethodBeginxCount err =%v", err)
		resp.Res = common.NewResult(5, "search err")
		return resp, nil
	}
	peer := new(filscanproto.Peer)
	if miner != nil {
		peer.PeerId = miner.PeerId
		peer.MinerAddress = miner.MinerAddr
		res, err := models.GetPeerByPeerId(miner.PeerId)
		if err != nil {
			log("GetPeerByPeerId err =%v", err)
			resp.Res = common.NewResult(5, "search err")
			return resp, nil
		}
		if res != nil {
			peer.Ip = res.Ip
			peer.LocationCn = res.LocationCN
			peer.LocationEn = res.LocationEN
		} else {
			peer.Ip = ""
			peer.LocationCn = ""
			peer.LocationEn = ""
		}
	} else {
		res, err := models.GetPeerByPeerId(peerId)
		if err != nil {
			log("GetPeerByPeerId err =%v", err)
			resp.Res = common.NewResult(5, "search err")
			return resp, nil
		}
		if res != nil {
			peer.PeerId = peerId
			peer.Ip = res.Ip
			peer.LocationCn = res.LocationCN
			peer.LocationEn = res.LocationEN
		}
	}
	resp.Data = &filscanproto.PeerByIdResp_Data{Peer: peer}
	resp.Res = common.NewResult(3, "sucess")
	return resp, nil
}

func (this *FilscanPeer) ActivePeerCount(ctx context.Context, input *common.Empty) (*filscanproto.CountResq, error) {
	resp := new(filscanproto.CountResq)
	miners, err := models.GetMinerstateActivateAtTime(uint64(models.TimeNow))
	//total, err := models.GetActivePeerCountByTime(models.TimeNow)
	if err != nil {
		log("GetMsgByBlockMethodBeginxCount err =%v", err)
		resp.Res = common.NewResult(5, "search err")
		return resp, nil
	}
	resp.Res = common.NewResult(3, "sucess")
	resp.Data = &filscanproto.CountResq_Data{Count: strconv.Itoa(len(miners))}
	return resp, nil
}

func (this *FilscanPeer) PeerMap(ctx context.Context, input *common.Empty) (*filscanproto.PeerMapResq, error) {
	resp := new(filscanproto.PeerMapResq)

	if peerPointCash != nil && models.TimeNow-peerPointCash.Time < peerPointCash.CashTime { //
		resp.Res = common.NewResult(3, "success")
		resp.Data = &filscanproto.PeerMapResq_Data{PeerPoint: peerPointCash.PeerPoint}
		return resp, nil
	}
	res, err := models.GetPeerGroup()
	if err != nil {
		log("GetPeerGroup err =%v", err)
		resp.Res = common.NewResult(5, "search err")
		return resp, nil
	}
	var pp []*filscanproto.PeerPoint
	for _, value := range res {
		peer := new(filscanproto.PeerPoint)
		longkey := 0
		for key, v := range value.LocationCn {
			if len(v) > len(value.LocationCn) {
				longkey = key
			}
		}
		peer.LocationCn = value.LocationCn[longkey]
		peer.LocationEn = value.LocationEn[longkey]
		peer.Latitude = value.ID.Latitude
		peer.Longitude = value.ID.Longitude
		var pppii []*filscanproto.PeerPoint_PeerIdIp
		for k, id := range value.PeerID {
			pi := new(filscanproto.PeerPoint_PeerIdIp)
			pi.PeerId = id
			if len(value.IP) > k {
				pi.Ip = value.IP[k]
			}
			pppii = append(pppii, pi)
		}
		peer.Peers = pppii
		pp = append(pp, peer)
	}
	peerPointCash.Time = models.TimeNow
	peerPointCash.PeerPoint = pp
	resp.Res = common.NewResult(3, "success")
	resp.Data = &filscanproto.PeerMapResq_Data{PeerPoint: pp}
	return resp, nil
}
