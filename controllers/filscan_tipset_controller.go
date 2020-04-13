package controllers

import (
	"encoding/json"
	"filscan_lotus/filscaner"
	filscanproto "filscan_lotus/filscanproto"
	"filscan_lotus/models"
	"gitlab.forceup.in/dev-proto/common"
	"golang.org/x/net/context"
	"strconv"
)

var _ filscanproto.FilscanTipsetServer = (*FilscanTipset)(nil)

type FilscanTipset struct {
}

func TipsetQueue() *filscaner.Fs_tipset_cache {
	return flscaner.List()
}

/**
//
BlockByHeight(context.Context, *BlockByHeightReq) (*BlockInfoResp, error)
//
BlockByMiner(context.Context, *BlockByMinerReq) (*BlockByMinerResp, error)
//
BlockByCid(context.Context, *BlockByCidReq) (*BlockInfoResp, error)
//
TipSetTree(context.Context, *TipSetTreeReq) (*TipSetTreeResp, error)

TipsetList(context.Context, *ListReq) (*TipsetListResq, error)

BlockConfirmCount(context.Context, *BlockByCidReq) (*CountResq, error)
*/

func (this *FilscanTipset) BlockByHeight(ctx context.Context, input *filscanproto.BlockByHeightReq) (*filscanproto.BlockInfoResp, error) {
	resp := new(filscanproto.BlockInfoResp)
	h := input.GetHeight()
	blocks, err := models.GetBlockByHeight(h)
	if err != nil {
		log("GetBlockByCid err=%v", err)
		resp.Res = common.NewResult(5, "GetBlockByCid err")
		return resp, nil
	}
	if len(blocks) < 1 {
		// e := fTipsetQueue.TipsetByOneHeight(h)
		e := TipsetQueue().FindTipset_height(h)
		if e != nil {
			for _, value := range e.Blocks {
				tbyte, _ := json.Marshal(value.Block)
				var p models.FilscanBlockResult
				json.Unmarshal(tbyte, &p)
				blocks = append(blocks, p)
			}
		}
	}
	var bs []*filscanproto.FilscanBlock
	for _, value := range blocks {
		b := FilscanBlockResult2PtotoFilscanBlock(value)
		bs = append(bs, b)
	}
	resp.Data = &filscanproto.BlockInfoResp_Data{Blocks: bs}
	resp.Res = common.NewResult(3, "success")
	return resp, nil
}

func (this *FilscanTipset) BlockByCid(ctx context.Context, input *filscanproto.BlockByCidReq) (*filscanproto.BlockInfoResp, error) {
	resp := new(filscanproto.BlockInfoResp)
	cid := input.GetCid()
	if !CheckArg(cid) {
		resp.Res = common.NewResult(5, "Value is null")
		return resp, nil
	}
	var cids []string
	cids = append(cids, cid)
	blocks, err := models.GetBlockByCid(cids)
	if err != nil {
		log("GetBlockByCid err=%v", err)
		resp.Res = common.NewResult(5, "GetBlockByCid err")
		return resp, nil
	}
	if len(blocks) < 1 {
		// bm := TipsetQueue.BlockByCid(cid)
		bm := TipsetQueue().FindBlock_id(cid)
		if bm != nil {
			tbyte, _ := json.Marshal(bm.Block)
			var p models.FilscanBlockResult
			json.Unmarshal(tbyte, &p)
			blocks = append(blocks, p)
		}
	}
	var bs []*filscanproto.FilscanBlock
	for _, value := range blocks {
		b := FilscanBlockResult2PtotoFilscanBlock(value)
		bs = append(bs, b)
	}
	resp.Data = &filscanproto.BlockInfoResp_Data{Blocks: bs}
	resp.Res = common.NewResult(3, "success")
	return resp, nil
}

func (this *FilscanTipset) TipSetTree(ctx context.Context, input *filscanproto.TipSetTreeReq) (*filscanproto.TipSetTreeResp, error) {
	resp := new(filscanproto.TipSetTreeResp)
	count := uint64(input.GetCount())
	end := uint64(input.GetEndHeight())

	tipsets, err := GetfilscanprotoTipsetByHeight(end-count+1, end)
	if err != nil {
		resp.Res = common.NewResult(5, "Search err")
		return resp, nil
	}
	//set null block
	newTipset := make([]*filscanproto.TipSet, count)
	for i := uint64(0); i < count; i++ {
		for _, value := range tipsets {
			if value.Tipset[0].BlockHeader.Height == int64(end-count+1+i) {
				newTipset[i] = value
				break
			} else {
				newTipset[i] = &filscanproto.TipSet{}
			}
		}
	}
	resp.Data = &filscanproto.TipSetTreeResp_Data{Tipsets: newTipset}
	resp.Res = common.NewResult(3, "success")
	return resp, nil
}

func (this *FilscanTipset) TipsetList(ctx context.Context, input *filscanproto.ListReq) (*filscanproto.TipsetListResq, error) {
	resp := new(filscanproto.TipsetListResq)
	//begindex := input.GetBegindex()
	//count := input.GetCount()
	//if !CheckArg(count) {
	//	resp.Res = common.NewResult(5, "Value is null")
	//	return resp, nil
	//}

	return resp, nil
}

func (this *FilscanTipset) BlockByMiner(ctx context.Context, input *filscanproto.BlockByMinerReq) (*filscanproto.BlockByMinerResp, error) {
	resp := new(filscanproto.BlockByMinerResp)
	minerArr := input.GetMiners()
	begindex := input.GetBegindex()
	count := input.GetCount()
	if !CheckArg(count) || len(minerArr) < 1 {
		resp.Res = common.NewResult(5, "Value is null")
		return resp, nil
	}
	res, total, err := GetBlockByMiner(minerArr, int(begindex), int(count))
	if err != nil {
		resp.Res = common.NewResult(5, "Search err")
		return resp, nil
	}
	var fbList []*filscanproto.FilscanBlock

	for _, value := range res {
		fbList = append(fbList, FilscanBlockResult2PtotoFilscanBlock(*value))
	}
	resp.Res = common.NewResult(3, "success")
	t, _ := GetBlockTotalFilByMiner(minerArr)

	bs := &filscanproto.BlockByMinerResp_Data{Blocks: fbList, Total: int64(total), TotalFil: t}
	resp.Data = bs
	return resp, nil
}

func (this *FilscanTipset) BlockConfirmCount(ctx context.Context, input *filscanproto.BlockByCidReq) (*filscanproto.CountResq, error) {
	resp := new(filscanproto.CountResq)
	blockCid := input.GetCid()
	if !CheckArg(blockCid) {
		resp.Res = common.NewResult(5, "Value is null")
		return resp, nil
	}
	tipset, err := models.GetTipSetByBlockCid(blockCid)
	if err != nil {
		resp.Res = common.NewResult(5, "Search err")
		return resp, nil
	}
	var count uint64

	if tipset == nil { //db 中不存在
		// bm := TipsetQueue.BlockByCid(blockCid) //cash  block
		bm := TipsetQueue().FindBlock_id(blockCid) //cash  block
		if bm == nil {
			resp.Res = common.NewResult(3, "success")
			resp.Data = &filscanproto.CountResq_Data{Count: "fail"}
			return resp, nil
		} else {
			// count += uint64(len(TipsetQueue.TipsetByHeight(bm.Block.BlockHeader.Height, TipsetQueue.element[len(TipsetQueue.element)-1].Tipset.Height()))) //cash中 tipset高度 > bm高度 的数量
			count += uint64(len(TipsetQueue().FindTipset_in_height(bm.Block.BlockHeader.Height, TipsetQueue().Front().Height())))
		}
	} else {
		than, err := models.ThanHeightCount(tipset.Height)
		if err != nil {
			resp.Res = common.NewResult(5, "Search err")
			return resp, nil
		}
		// count = uint64(than + TipsetQueue.Size())
		count = uint64(than + TipsetQueue().Size())
	}
	resp.Res = common.NewResult(3, "success")
	c := strconv.FormatUint(count, 10)
	resp.Data = &filscanproto.CountResq_Data{Count: c}
	return resp, nil
}
