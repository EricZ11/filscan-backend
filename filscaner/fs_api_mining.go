package filscaner

import (
	"context"
	"errors"
	innererr "filscan_lotus/error"
	. "filscan_lotus/filscanproto"
	"filscan_lotus/models"
	"filscan_lotus/utils"
	"fmt"
	"github.com/globalsign/mgo"
	"github.com/ipfs-force-community/common"
	"math/big"
	"time"
)

func (fs *Filscaner) MinerList(ctx context.Context, req *MinerListReq) (*MinerListResp, error) {
	resp := &MinerListResp{}
	if req.Sort == "" {
		req.Sort = "power"
	}

	if req.Sort != "block" && req.Sort != "power" {
		resp.Res = common.NewResult(5, "invalid paramater")
		return resp, nil
	}

	var err error
	var minerlist *Models_minerlist

	start := req.TimeStart
	end := req.TimeEnd
	now := uint64(time.Now().Unix())

	if start < fs.chain_genesis_time {
		start = fs.chain_genesis_time
	}
	if end > now {
		end = now
	}

	if end < start {
		start, end = end, start
	}

	if start == end {
		resp.Res = common.NewResult(3, "success")
		return resp, nil
	}

	sort_type := -1
	if req.SortType > 0 {
		sort_type = 1
	}

	switch req.Sort {
	case "power":
		minerlist, err = fs.models_minerlist_sort_power(nil, start, end, req.Offset, req.Limit, req.Sort, sort_type)
		if err != nil {
			resp.Res = resp_search_error
			return resp, nil
		}
		tmp_minerlist, err := fs.models_miner_list_sort_block(minerlist.GetMiners(), start, end, req.Offset, req.Limit, "", sort_type)
		if err != nil {
			resp.Res = resp_search_error
			return resp, nil
		}
		minerlist.SetBlockValues(tmp_minerlist)
	case "block":
		minerlist, err = fs.models_miner_list_sort_block(nil, start, end, req.Offset, req.Limit, req.Sort, sort_type)
		if err != nil {
			resp.Res = resp_search_error
			return resp, nil
		}
		tmp_minerlist, err := fs.models_minerlist_sort_power(minerlist.GetMiners(), start, end, 0, req.Limit, "", sort_type)
		if err != nil {
			resp.Res = resp_search_error
			return resp, nil
		}
		minerlist.SetPowerValues(tmp_minerlist)
	default:
		resp.Res = resp_invalid_parama
		return resp, nil
	}

	resp.Data = minerlist.APIRespData()
	resp.Data.SortType = req.Sort
	resp.Res = resp_success
	return resp, nil
}

// 返回的是时间段内, 旷力增加最多的矿工地址,
// block也是时间段内, 矿工爆块的数量和这段时间的爆块总数.
func (fs *Filscaner) TopnPowerIncreaseMiners(ctx context.Context, req *TopnPowerIncreaseMinersReq) (*TopnPowerIncreaseMinersResp, error) {
	if req.TimeAt == 0 {
		req.TimeAt = uint64(time.Now().Unix())
	}
	resp := &TopnPowerIncreaseMinersResp{}

	records, total, err := fs.models_miner_power_increase_top_n(req.TimeStart, req.TimeAt, req.Offset, req.Limit)
	if err != nil {
		resp.Res = resp_search_error
		return resp, nil
	}

	resp.Res = resp_success

	data := &TopnPowerIncreaseMinersResp_Response{
		TotalMinerCount: total,
		Records:         make([]*TopnPowerIncreaseMinersResp_Response_Record, len(records)),
	}

	miners := make([]string, len(records))

	total_increased_power := uint64(0)
	for _, record := range records {
		total_increased_power += record.IncreasedPower
	}

	for index, record := range records {
		data.Records[index] = &TopnPowerIncreaseMinersResp_Response_Record{
			IncreasedPower:        record.IncreasedPower,
			Miner:                 record.Record.State(),
			IncreasedPowerPercent: utils.IntToPercent(record.IncreasedPower, total_increased_power)}
		miners[index] = record.Record.MinerAddr
	}
	data.TotalIncreasedPower = total_increased_power

	map_records, total_block, err := fs.models_blockcount_time_range_with_miners(miners, req.TimeStart, req.TimeAt)
	if err == nil {
		for _, record := range data.Records {
			record.BlockCount, _ = map_records[record.Miner.Address]
		}
		data.TotalBlockCount = total_block
		for _, record := range data.Records {
			record.BlockPercent = fmt.Sprintf("%.2f%%", float64(record.BlockCount*100)/float64(total_block))
		}
	} else {
		fs.Printf("models_blockcount_time_range_with_miners error, message:%s\n", err.Error())
	}

	resp.Data = data

	return resp, nil
}

func (fs *Filscaner) TopnBlockMiners(ctx context.Context, req *TopnBlockMinersReq) (*TopnBlockMinersResp, error) {
	if req.TimeAt == 0 {
		req.TimeAt = uint64(time.Now().Unix())
	}

	resp := &TopnBlockMinersResp{}
	mined_block, miner_count, err := fs.models_miner_block_top_n(req.TimeStart, req.TimeAt, req.Offset, req.Limit)
	if err != nil {
		resp.Res = resp_search_error
		return resp, nil
	}

	miners := make([]string, len(mined_block))
	for index, m := range mined_block {
		miners[index] = m.Miner
	}

	miner_power_increased, err := fs.models_miner_power_increase_in_time(miners, req.TimeStart, req.TimeAt)
	if err != nil {
		resp.Res = resp_search_error
		return resp, nil
	}

	resp.Res = resp_success

	data := &TopnBlockMinersResp_Response{
		TotalMinerCount: miner_count,
		Records:         make([]*TopnBlockMinersResp_Response_Record, len(mined_block)),
	}

	for index, m := range mined_block {
		data.TotalBlockCount += m.BlockCount

		increased_power_miner, _ := miner_power_increased[m.Miner]
		increased_power := uint64(0)

		var minerstate *MinerState

		if increased_power_miner != nil {
			minerstate = increased_power_miner.Record.State()
		}

		if minerstate == nil {
			minerstate = &MinerState{Address: m.Miner}
		} else {
			increased_power = increased_power_miner.IncreasedPower
		}

		data.Records[index] = &TopnBlockMinersResp_Response_Record{
			BlockCount:     m.BlockCount,
			Miner:          minerstate,
			IncreasedPower: increased_power}
		data.TotalIncreasedPower += increased_power
	}

	for _, record := range data.Records {
		record.IncreasedPowerPercent = utils.IntToPercent(record.IncreasedPower, data.TotalIncreasedPower)
		record.BlockPercent = utils.IntToPercent(record.BlockCount, data.TotalBlockCount)
	}

	resp.Data = data
	return resp, nil
}

func (fs *Filscaner) TopnPowerMiners(ctx context.Context, req *TopnPowerMinersReq) (*TopnPowerMinerResp, error) {
	if req.Limit == 0 {
		return nil, innererr.ErrInvalidParam
	}

	resp := &TopnPowerMinerResp{
		Res:  common.NewResult(3, "success"),
		Data: &TopnPowerMinerResp_Data{},
	}

	resp.Data.Miners, _ = fs.miner_cache24h.index(0)
	resp.Res = resp_success

	resp.Data.TotalMinerCount = uint64(len(resp.Data.Miners))

	return resp, nil
}

func (fs *Filscaner) TopnPowerMiners_old(ctx context.Context, req *TopnPowerMinersReq) (*TopnPowerMinerResp, error) {
	if req.Limit == 0 {
		return nil, innererr.ErrInvalidParam
	}

	resp := &TopnPowerMinerResp{
		Res:  common.NewResult(3, "success"),
		Data: &TopnPowerMinerResp_Data{},
	}

	time_at := int64(req.TimeAt)
	if time_at == 0 {
		time_at = time.Now().Unix()
	}

	miners, total, err := models_miner_top_power(nil, time_at, int64(req.Offset), int64(req.Limit))
	if err != nil {
		fs.Printf("models_miner_top_power error, message:%s\n", err.Error())
		resp.Res = resp_search_error
		return resp, nil
	}

	resp.Res = resp_success

	if len(miners) > 0 {
		resp.Data.Miners = fs.to_resp_slice(miners)
	}

	resp.Data.TotalMinerCount = uint64(total)

	return resp, nil
}

func (fs *Filscaner) MinerSearch(ctx context.Context, req *MinerSearchReq) (*MinerSearchResp, error) {
	resp := &MinerSearchResp{}

	if req.Miner == "" {
		resp.Res = common.NewResult(5, "invalid param")
		return resp, nil
	}

	miners, err := fs.models_search_miner(req.Miner)
	if err != nil {
		resp.Res = resp_search_error
		return resp, nil
	}

	var start = uint64(0)
	var end = uint64(time.Now().Unix())
	var limit = uint64(len(miners))
	minerlist, err := fs.models_minerlist_sort_power(miners, start, end, 0, limit, "power", -1)
	if err != nil {
		resp.Res = resp_search_error
		return resp, nil
	}

	block_miner_list, err := fs.models_miner_list_sort_block(minerlist.GetMiners(), start, end, 0, limit, "", -1)
	if err != nil {
		resp.Res = resp_search_error
		return resp, nil
	}

	minerlist.SetBlockValues(block_miner_list)

	resp.Res = resp_success

	api_res := minerlist.APIRespData()

	resp.Data = &MinerSearchResp_Response{
		Miners:     api_res.Miners,
		MinerCount: uint64(len(api_res.Miners))}

	return resp, nil
}

func (fs *Filscaner) MinerPowerAtTime(ctx context.Context, req *MinerPowerAtTimeReq) (*MinerPowerAtTimeResp, error) {
	resp := &MinerPowerAtTimeResp{
		Res:  common.NewResult(3, "success"),
		Data: make(map[string]*MinerPowerAtTimeResp_Resdata),
	}

	res_datas := resp.Data

	time_diff := req.TimeDiff

	time_at := req.TimeAt
	time_tmp := time_at

	var cache *fs_miner_cache
	if time_diff == 3600 {
		cache = fs.miner_cache24h
	} else if time_diff == 86400 {
		cache = fs.miner_cache1day
	} else {
		resp.Res = common.NewResult(4, fmt.Sprintf("invalid time_duration:%d", time_diff))
	}

	repeats := int(utils.Min(int64(req.RepeateTime), cache.max_cached_size))

	for i := 0; i < repeats; i++ {
		// is_nil 为true表示所有的数据都是伪造的
		stats, is_nil := cache.index(i)
		if stats == nil {
			break
		}
		for _, stat := range stats {
			res_data := res_datas[stat.Address]
			if res_data == nil {
				res_data = &MinerPowerAtTimeResp_Resdata{}
				res_datas[stat.Address] = res_data
			}
			x := &MinerPowerAtTimeResp_X{
				AtTime:      time_tmp,
				MinerStates: stat}
			res_data.Data = append(res_data.Data, x)
		}

		time_tmp = time_at - (time_at % time_diff) - (uint64(i) * time_diff)
		if is_nil {
			break
		}
	}

	return resp, nil
}

func (fs *Filscaner) MinerPowerAtTime_old(ctx context.Context, req *MinerPowerAtTimeReq) (*MinerPowerAtTimeResp, error) {

	if req.RepeateTime > 256 {
		return nil, errors.New("repeate must time less than 36")
	}

	var time_at = req.TimeAt
	if time_at == 0 {
		time_at = uint64(time.Now().Unix())
	}

	resp := &MinerPowerAtTimeResp{
		Res:  common.NewResult(3, "success"),
		Data: map[string]*MinerPowerAtTimeResp_Resdata{},
	}

	for i := uint64(0); i < req.RepeateTime && time_at > 0; i++ {

		func_begin := time.Now()
		minerStates, err := fs.models_miner_power_increase_in_time(req.Miners, 0, time_at)
		diff := time.Since(func_begin)
		fs.Printf("get_miner_power_increased_within_time_range, use time = %.3f(second)\n", diff.Seconds())

		if err != nil {
			resp.Res = resp_search_error
			return resp, nil
		}

		if len(minerStates) == 0 || minerStates == nil {
			break
		}

		var minerState *MinerState
		otherPower := big.NewInt(0)
		maxTotalPower := big.NewInt(0)

		for _, addr := range req.Miners {
			ms, isok := minerStates[addr]
			if !isok || ms == nil {
				minerState = &MinerState{
					Address:      addr,
					Power:        "0",
					PowerPercent: "0%"}
			} else {
				minerState = ms.Record.State()
				otherPower.Add(otherPower, ms.Record.Power.Int)
				if maxTotalPower.Cmp(ms.Record.TotalPower.Int) < 0 {
					maxTotalPower = ms.Record.TotalPower.Int
				}
			}

			tmp, isok := resp.Data[addr]
			if !isok || tmp == nil {
				tmp = &MinerPowerAtTimeResp_Resdata{
					Data: []*MinerPowerAtTimeResp_X{},
				}
			}

			// fmt.Print(otherPower.String(), "\n", maxTotalPower.String(), "\n-------\n")
			tmp.Data = append(tmp.Data, &MinerPowerAtTimeResp_X{MinerStates: minerState, AtTime: time_at})

			resp.Data[addr] = tmp
		}

		otherPower.Sub(maxTotalPower, otherPower)

		tmp, isok := resp.Data["other"]
		if !isok || tmp == nil {
			tmp = &MinerPowerAtTimeResp_Resdata{}
		}

		tmp.Data = append(tmp.Data, &MinerPowerAtTimeResp_X{
			AtTime: time_at,
			MinerStates: &MinerState{
				Address:      "other",
				Power:        utils.XSizeString(otherPower),
				PowerPercent: utils.BigToPercent(otherPower, maxTotalPower),
			}})
		resp.Data["other"] = tmp

		time_at = time_at - req.TimeDiff
	}

	return resp, nil
}

func (fs *Filscaner) TotalPowerGraphical(ctx context.Context, req *TotalPowerGraphicalReq) (*TotalPowerGraphicalResp, error) {
	resp := new(TotalPowerGraphicalResp)

	var time_at = uint64(req.GetTime())
	if time_at == 0 {
		time_at = uint64(time.Now().Unix())
	}
	//time_at  = uint64(1576763685)
	RepeateTime := uint64(24)
	TimeDiff := uint64(60 * 60)

	resp.Data = &TotalPowerGraphicalResp_Data{
		Data:            []*TotalPowerGraphical{},
		StorageCapacity: 0.00,
	}
	for i := uint64(0); i < RepeateTime && time_at > 0; i++ {
		powerStates, err := models.GetTotalpowerAtTime(time_at)
		if err != nil {
			fmt.Println(fmt.Sprintf("GetMsgByBlockMethodBeginxCount err =%v", err))
			resp.Res = resp_search_error
			return nil, err
		}
		t := new(TotalPowerGraphical)
		if powerStates != nil && powerStates.TotalPower != nil {
			t.Power = powerStates.TotalPower.Int64()
		} else {
			t.Power = 0
		}
		t.Time = int64(time_at)
		resp.Data.Data = append(resp.Data.Data, t)
		time_at = time_at - TimeDiff
	}
	for i := 0; i < len(resp.Data.Data)/2; i++ {
		resp.Data.Data[i], resp.Data.Data[len(resp.Data.Data)-i-1] = resp.Data.Data[len(resp.Data.Data)-i-1], resp.Data.Data[i]
	}
	resp.Data.StorageCapacity = float64(resp.Data.Data[len(resp.Data.Data)-1].Power)
	return resp, nil
}

func (fs *Filscaner) ActiveStorageMinerCountAtTime(ctx context.Context, req *ActiveStorageMinerReq) (*ActiveStorageMinerResp, error) {
	var time_at = req.TimeAt
	time_now := uint64(time.Now().Unix())

	if time_at == 0 || time_at > time_now {
		time_at = time_now
	}

	if req.RepeateTime > 64 {
		req.RepeateTime = 64
	}

	res := &ActiveStorageMinerResp{
		Res:  common.NewResult(3, "success"),
		Data: []*ActiveStorageMinerResp_Data{},
	}

	time_diff := req.TimeDiff
	for i := uint64(0); i < req.RepeateTime && time_at > 0; i++ {
		count, err := fs.model_active_miner_count_at_time(time_at, time_diff)

		if err != nil && err != mgo.ErrNotFound {
			fs.Printf("get activate miner count at time failed, message:%s\n",
				err.Error())
			break
		}

		res.Data = append(res.Data, &ActiveStorageMinerResp_Data{
			TimeAt: time_at,
			Count:  count})

		if i == 0 {
			time_at = time_at - (time_at % req.TimeDiff)
		} else {
			time_at = time_at - req.TimeDiff
		}
		if time_at < fs.chain_genesis_time {
			break
		}
	}

	return res, nil
}
