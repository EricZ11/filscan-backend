package controllers

import (
	"encoding/json"
	filscanproto "filscan_lotus/filscanproto"
	"filscan_lotus/models"
	"filscan_lotus/utils"
	"github.com/filecoin-project/lotus/chain/types"
	"math"
	"strconv"
)

/**
get unification data from tipsetqueue and db
*/
func GetBlockByIndex(start, count int) (res []*models.FilscanBlockResult, err error) {
	// head := TipsetQueue.BlockByIndex(start, count) //get block by cash
	head := flscaner.List().FindBlock_ofst_count(start, count)

	var v []*models.FilscanBlockResult
	if count-start-len(head) > 0 {
		v, err = models.GetLatestBlockList(count - start - len(head)) // get block by db
		if err != nil {
			return nil, err
		}
	}
	var blockheaders []*models.FilscanBlockResult

	for _, value := range head {
		tbyte, _ := json.Marshal(value.Block)
		var p *models.FilscanBlockResult
		err := json.Unmarshal(tbyte, &p)
		if err != nil {
			return nil, err
		}
		blockheaders = append(blockheaders, p)
	}
	if len(v) > 0 {
		blockheaders = append(blockheaders, v...)
	}
	return blockheaders, nil
}

func GetMsgByIndex(start, end int) (res []*models.FilscanMsgResult, err error) {
	// head := TipsetQueue.MsgByIndex(start, end) //get block by cash
	head := flscaner.List().FindMesage_ofset_count(start, end) //get block by cash
	var v []*models.FilscanMsgResult
	if end-start-len(head) > 0 {
		v, err = models.GetMsgLatestList(end - start - len(head)) // get block by db
		if err != nil {
			return nil, err
		}
	}
	var msgList []*models.FilscanMsgResult

	for _, value := range head {
		tbyte, _ := json.Marshal(value)
		var p models.FilscanMsgResult
		err := json.Unmarshal(tbyte, &p)
		if err != nil {
			return nil, err
		}
		msgList = append(msgList, &p)
	}
	if len(v) > 0 {
		msgList = append(msgList, v...)
	}
	return msgList, nil
}

/*func GetTipsetByHeight(startHeight , endHeight uint64)(res []*models.FilscanTipSetResult ,err error){
	var tipsetList []*models.FilscanTipSetResult

	tipset := TipsetQueue.TipsetByHeight(startHeight , endHeight)
	var queueStart uint64
	if len(tipset) > 0  {
		queueStart = tipset[0].Height()
		if tipset[0].Height() >= startHeight && tipset[len(tipset)-1].Height()>= endHeight { // all data in cash
			return TipSet2FilscanTipSet(tipset),nil
		}
		tipsetList = TipSet2FilscanTipSet(tipset)
	}else {
		queueStart = endHeight
	}
	dbRes, err := models.GetTipSetByHeight(startHeight, queueStart)
	if err != nil {
		return nil, err
	}
	tipsetList = append(tipsetList, dbRes...)
	return tipsetList,nil
}*/

func GetfilscanprotoTipsetByHeight(startHeight, endHeight uint64) (res []*filscanproto.TipSet, err error) {
	var queueTipElement []*models.Element
	tipset := flscaner.List().FindTipset_in_height(startHeight, endHeight)
	var queueStart uint64
	if len(tipset) > 0 {
		queueStart = tipset[0].Tipset.Height()
		//if tipset[0].tipset.Height() >= startHeight && tipset[len(tipset)-1].tipset.Height()>= endHeight { // all data in cash
		//	queueTipElement = tipset
		//}else { //part data in cash
		//	queueTipElement =
		//}
		queueTipElement = tipset
	} else {
		queueStart = endHeight
	}
	var dbRes []*models.FilscanTipSetResult
	if queueStart >= startHeight {
		dbRes, err = models.GetTipSetByHeight(startHeight, queueStart)
		if err != nil {
			return nil, err
		}
	}

	if len(queueTipElement) > 0 {
		for _, value := range queueTipElement { //get tipset  blocks info
			tip := new(filscanproto.TipSet)
			tip.MinTicketBlock = value.Tipset.MinTicketBlock().Cid().String()
			var filscanblocks []*filscanproto.FilscanBlock
			for _, value := range value.Blocks {
				tbyte, _ := json.Marshal(value.Block)
				var p models.FilscanBlockResult
				json.Unmarshal(tbyte, &p)
				b := FilscanBlockResult2PtotoFilscanBlock(p)
				filscanblocks = append(filscanblocks, b)
			}
			tip.Tipset = filscanblocks
			res = append(res, tip)
		}
	}
	for _, value := range dbRes {
		tip := new(filscanproto.TipSet)
		tip.MinTicketBlock = value.MinTicketCId.Str
		var cids []string
		for _, value := range value.Cids {
			cids = append(cids, value.Str)
		}
		blocks, err := models.GetBlockByCid(cids)
		if err != nil {
			return res, err
		}
		var filscanblocks []*filscanproto.FilscanBlock
		for _, value := range blocks { //tipset内所有blocks
			b := FilscanBlockResult2PtotoFilscanBlock(value)
			flag := true
			for _, fvalue := range filscanblocks { //防止 block 已存在
				if fvalue.Cid == b.Cid {
					flag = !flag
					break
				}
			}
			if flag {
				filscanblocks = append(filscanblocks, b)
			}
		}
		tip.Tipset = filscanblocks
		res = append(res, tip)
	}

	return res, nil
}

func TipSet2FilscanTipSet(tipsetArr []*types.TipSet) (res []*models.FilscanTipSetResult) {
	if len(tipsetArr) < 1 {
		return res
	}
	for _, t := range tipsetArr {
		var tips models.FilscanTipSet
		tips.Cids = t.Cids()
		tips.Height = t.Height()
		tips.Parents = t.Parents().Cids()
		tips.MinTicketCId = t.MinTicketBlock().Cid()
		tbyte, _ := json.Marshal(tips)
		var p models.FilscanTipSetResult
		json.Unmarshal(tbyte, &p)
		res = append(res, &p)
	}
	return res
}

func GetMsgByBlockMethodBeginxCount(count, beginx int64, blockCid, method string) (res []*models.FilscanMsgResult, total int, err error) {
	var queueRes []*models.FilscanMsg
	if len(blockCid) < 1 && len(method) < 1 {
		queueRes = flscaner.List().MesageAll()
	} else {
		// queueRes = TipsetQueue.MsgByBlockCidMethodName(blockCid, method)
		queueRes = flscaner.List().FindMesage_block_method(blockCid, method)
	}
	var cashRes []*models.FilscanMsg
	var diff int
	total = len(queueRes)
	if total > 0 && int64(total) > beginx {
		if len(queueRes) > int(beginx+count) {
			cashRes = queueRes[beginx : beginx+count]
		} else {
			cashRes = queueRes[beginx:]
		}
	} else {
		diff = total
	}

	res = append(res, FilscanMsg2FilscanMsgResult(cashRes)...)
	b := 0
	if int(beginx)-len(cashRes)-diff < 0 {
		b = -1 * (int(beginx) - len(cashRes) - diff)
	} else {
		b = int(beginx) - len(cashRes) - diff
	}
	msgList, total2, err := models.GetMsgByBlockMethodNameLimit(blockCid, method, b, int(count)-len(cashRes))
	if err != nil {
		return nil, 0, err
	}
	if len(msgList) > 0 {
		res = append(res, msgList...)
	}
	total += total2
	return res, total, nil
}

func GetBlockByMiner(minerArr []string, start, count int) (res []*models.FilscanBlockResult, total int, err error) {
	// blockList := TipsetQueue.SortBlockByMinerArr(minerArr)
	blockList := flscaner.List().FindBlock_miners(minerArr)

	total = len(blockList)
	//var cashRes []*models.FilscanBlock
	var diff int
	if total > 0 && total > start {
		if len(blockList) > start+count {
			blockList = blockList[start : start+count]
		} else {
			blockList = blockList[start:]
		}
	} else {
		diff = total
	}
	b := 0
	if int(start)-len(blockList)-diff < 0 {
		b = -1 * (int(start) - len(blockList) - diff)
	} else {
		b = int(start) - len(blockList) - diff
	}
	//result, total2, err := models.GetBlockListByMiner(minerArr, start-len(blockList), end-start-len(blockList)) // get block by db
	result, total2, err := models.GetBlockListByMiner(minerArr, b, count-len(blockList)) // get block by db

	fbr := FilscanBlock2FilscanBlockResult(blockList)
	if len(result) > 0 {
		fbr = append(fbr, result...)
	}
	total += total2
	return fbr, total, nil
}

func GetBlockTotalFilByMiner(minerArr []string) (total string, err error) {
	// blockList := TipsetQueue.SortBlockByMinerArr(minerArr) //缓存中的block
	blockList := flscaner.List().FindBlock_miners(minerArr)
	var totalFil float64
	for _, value := range blockList {
		totalFil += utils.StringToFloat(value.BlockReward)
	}
	dbTotal, err := models.GetBlockTotalRewardFilByMiner(minerArr) // get blockfil by db
	totalFil += utils.StringToFloat(dbTotal)
	return utils.FloatToString(totalFil), nil
}

func GetMsgByAddressFromToMethod(address, fromTo, method string, start, count int) (res []*models.FilscanMsgResult, total int, err error) {
	// queueMsgList := TipsetQueue.MsgByAddressFromToMethodName(address, fromTo, method)
	queueMsgList := flscaner.List().FindMesage_address(address, fromTo, method)

	var cashRes []*models.FilscanMsg
	total = len(queueMsgList)
	var diff int
	if total > 0 && total > start {
		if len(queueMsgList) > start+count {
			cashRes = queueMsgList[start : start+count]
		} else {
			cashRes = queueMsgList[start:]
		}
	} else {
		diff = total
	}
	b := 0
	if start-len(cashRes)-diff < 0 {
		b = -1 * (start - len(queueMsgList) - diff)
	} else {
		b = int(start) - len(queueMsgList) - diff
		bfl := math.Abs(float64(b))
		bstring := strconv.FormatFloat(bfl, 'f', -1, 64)
		b, _ = strconv.Atoi(bstring)
	}
	res = append(res, FilscanMsg2FilscanMsgResult(cashRes)...)
	//msgList, total2, err := models.GetMsgByAddressFromMethodLimit(address, fromTo, method, start-len(cashRes), end-len(cashRes))
	msgList, total2, err := models.GetMsgByAddressFromMethodLimit(address, fromTo, method, b, count-len(cashRes))
	if err != nil {
		return nil, 0, err
	}
	if len(msgList) > 0 {
		res = append(res, msgList...)
	}
	total += total2
	return res, total, nil
}

//从DB  CASH 获取 时间区间内的 block
func GetBlockNumByTime(startTime, endTime int64) (bms []*models.FilscanBlockResult, err error) {
	// allBlock := TipsetQueue.AllBlock()
	allBlock := flscaner.List().Blocks()
	for _, value := range allBlock {
		if value.Block.BlockHeader.Timestamp > uint64(startTime) && value.Block.BlockHeader.Timestamp < uint64(endTime) {
			tbyte, _ := json.Marshal(value.Block)
			var p *models.FilscanBlockResult
			err := json.Unmarshal(tbyte, &p)
			if err != nil {
				return nil, err
			}
			bms = append(bms, p)
		}
	}
	resBlock, err := models.GetBlockByTime(startTime, endTime)
	if len(resBlock) > 0 {
		bms = append(bms, resBlock...)
	}
	return
}

//从DB  CASH 获取 时间区间内的 tipset数量
func GetTipsetNumByTime(startTime, endTime int64) (num int, err error) {
	//allBlock := TipsetQueue.AllBlock()
	// for _, value := range TipsetQueue.element {
	// 	if int64(value.Tipset.MinTimestamp()) >= startTime && int64(value.Tipset.MinTimestamp()) < endTime {
	// 		num += 1
	// 	}
	// }
	num = int(flscaner.List().TipsetCountInTime(startTime, endTime))
	resTipset, err := models.GetTipsetCountByStartEndTime(startTime, endTime)
	num = num + resTipset
	return
}

//获取时间区间内合适的 tipset height
// func GetProperTipsetHeightByTime(time uint64) (uint64, error) {
// 	var tipset *types.TipSet
// 	for _, value := range TipsetQueue.element {
// 		if value.Tipset.MinTimestamp() <= time {
// 			if tipset != nil {
// 				if value.Tipset.MinTimestamp() > tipset.MinTimestamp() {
// 					tipset = value.Tipset
// 				}
// 			} else {
// 				tipset = value.Tipset
// 			}
// 		}
// 	}
//
// 	//if l == 1 {
// 	//	return tipsets[0].Height(), nil
// 	//}
// 	if tipset != nil {
// 		return tipset.Height(), nil
// 	}
// 	t, err := models.GetTipsetByTime(int64(time))
// 	if err != nil {
// 		return 0, err
// 	}
// 	if t != nil {
// 		return t.Height, nil
// 	} else {
// 		return 0, nil
// 	}
// }

func GetLatestBlockReward() (string, error) {
	fil := flscaner.List().LatestBlockrewards()
	if fil == "" || fil == "0" {
		return models.GetLatestReward()
	}
	return fil, nil

}
