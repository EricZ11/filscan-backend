package filscaner

import (
	"context"
	. "filscan_lotus/filscanproto"
	"filscan_lotus/models"
	"fmt"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/types"
	"gitlab.forceup.in/dev-proto/common"
	"math/big"
	"time"
)

var resp_success = &common.Result{Code: 3, Msg: "success"}
var resp_search_error = &common.Result{Code: 5, Msg: "search error"}
var resp_invalid_parama = &common.Result{Code: 5, Msg: "invalid param"}

func (fs *Filscaner) error_resp(err error) *common.Result {
	return common.NewResult(3, err.Error())
}

func (fs *Filscaner) FilNetworkBlockReward(ctx context.Context, req *FutureBlockRewardReq) (*FutureBlockRewardResp, error) {
	resp := &FutureBlockRewardResp{}

	timediff := req.TimeDiff
	repeate := req.Repeate
	time_now := uint64(time.Now().Unix())

	// rewards, released, err := fs.future_block_rewards(timediff, repeate)
	rewards, _, err := fs.future_block_rewards(timediff, repeate)
	if err != nil {
		resp.Res = resp_search_error
		return nil, err
	}

	resp.Data = make([]*FutureBlockRewardResp_Data, req.Repeate)

	for index, v := range rewards {
		resp.Data[index] = &FutureBlockRewardResp_Data{
			Time:         time_now,
			BlockRewards: ToFilString(v)}
		// VestedRewards: released.String()}
		time_now += timediff
		// released.Add(released, v)
	}

	resp.Res = resp_success
	return resp, nil
}

var TOTAL_REWARDS = types.FromFil(build.MiningRewardTotal).Int
var TOTAL_FILCOIN = types.FromFil(build.TotalFilecoin).Int

// func calculate_remain_reward_at_block(height uint64) (*big.Int, *big.Int) {
// 	remaining := types.NewInt(0)
// 	remaining.SetString(TOTAL_REWARDS.String(), 10)
//
// 	for i := uint64(0); i < height; i++ {
// 		used := vm.MiningReward(remaining)
// 		used.Mul(used.Int, blocksPerEpoch)
//
// 		remaining.Sub(remaining.Int, used.Int)
// 	}
//
// 	total_used := big.NewInt(0)
// 	total_used.Sub(TOTAL_REWARDS, remaining.Int)
// 	return total_used, remaining.Int
// }

func (fs *Filscaner) vested_attime(time_at uint64) *big.Int {
	return big.NewInt(0)
}

func (fs *Filscaner) FilOutStanding(ctx context.Context, req *FilOutstandReq) (*FiloutstandResp, error) {
	start := req.TimeAt
	diff := req.TimeDiff
	repeate := req.Repeate

	time_now := uint64(time.Now().Unix())
	if start == 0 {
		start = time_now
	}

	start = start - (diff * repeate)

	resp := &FiloutstandResp{}

	var data []*FiloutstandResp_Data

	set_with_last_data := func(data []*FiloutstandResp_Data, iii *FiloutstandResp_Data) []*FiloutstandResp_Data {
		length := len(data)
		if length == 0 {
			zero_fil := ToFilString(big.NewInt(0))
			iii.Floating = zero_fil
			iii.PlegeCollateral = zero_fil
			iii.PlegeCollateral = zero_fil
		} else {
			iii = data[length-1]
		}
		return append(data, iii)
	}

	for i := uint64(0); i < repeate; i++ {
		if start < fs.chain_genisis_time {
			continue
		}
		if start > time_now {
			break
		}

		filoutresp_data := &FiloutstandResp_Data{TimeStart: start, TimeEnd: start + diff}
		_, max_height, _, err := fs.models_blockcount_time_range(start, start+diff)
		start += diff

		if err != nil {
			continue
		}

		// min_released_reward := fs.released_reward_at_height(min_height)
		max_released_reward := fs.released_reward_at_height(max_height)

		filoutresp_data.Floating = ToFilString(max_released_reward)

		tipset, err := fs.api.ChainGetTipSetByHeight(fs.ctx, max_height, nil)
		if err != nil {
			fs.Printf("chain_get_tipset_by_height(%d) failed,message;%s\n", err.Error())
			continue
		}

		pleged, err := fs.api.StatePledgeCollateral(ctx, tipset)
		if err != nil {
			set_with_last_data(data, filoutresp_data)
			fs.Printf("StatePledgeCollateral failed,message;%s\n", err.Error())
			return nil, err
		}

		filoutresp_data.PlegeCollateral = ToFilString(pleged.Int)
		filoutresp_data.Outstanding = fmt.Sprintf("%.4f", ToFil(max_released_reward)+ToFil(pleged.Int))
		data = append(data, filoutresp_data)
	}

	resp.Data = data
	resp.Res = resp_success
	return resp, nil
}

// 计算历史时间周期内的区块奖励
func (fs *Filscaner) CumulativeBlockRewardsOverTime(ctx context.Context, req *CBROReq) (*CBROResp, error) {
	start := req.TimeStart
	diff := req.TimeDiff
	repeate := req.Repeate

	if start < fs.chain_genisis_time {
		start = fs.chain_genisis_time
	}

	resp := &CBROResp{}

	// 这个数据也是大致计算的,并不完全准确, 完全准确的数据应该是:
	// current_reward_remain - vm.blockreward(rewards_remain * blocks_count_in_tipset)
	// vm.MiningReward()

	time_now := uint64(time.Now().Unix())
	// TODO:需要检查时间合法性!!!
	// rewards := make([]*big.Int, repeate)

	var data []*CBROResp_Data
	var max_released *big.Int

	offset := 0

	for i := uint64(0); i < repeate; i++ {
		s := start
		e := start + diff
		start += diff
		if start > time_now {
			break
		}

		cbrresp_data := &CBROResp_Data{
			TimeStart: start,
			TimeEnd:   start + diff}

		// 从数据库读取时间周期内的块高变化
		_, max_height, miner_count, err := fs.models_blockcount_time_range(s, e)
		if err != nil || max_height == 0 {
			if offset > 0 {
				cbrresp_data.BlocksReward = data[offset-1].BlocksReward
			} else {
				continue
			}
		} else {
			max_released = fs.released_reward_at_height(max_height)
			cbrresp_data.BlocksReward = ToFilString(max_released)
			cbrresp_data.MinerCount = miner_count
		}

		data = append(data, cbrresp_data)
		offset++
	}

	resp.Data = data
	resp.Res = resp_success
	return resp, nil
}


func (fs *Filscaner) MinerRewards(ctx context.Context, req *MinerRewardsReq) (*MinerRewardsResp, error) {
	resp := &MinerRewardsResp{}

	var start, count uint64
	var is_height bool

	if req.HeightCount!= 0 {
		is_height = true
		start = req.HeightStart
		count = req.HeightCount
	} else {
		is_height = false
		start = req.TimeStart
		count = req.TimeDiff
		// if start < fs.chain_genisis_time {
		// 	start = fs.chain_genisis_time
		// }
	}

	if count == 0 {
		resp.Res = resp_invalid_parama
		return resp, nil
	}

	// convert t3 address to t0 address
	var miners = req.Miners
	var worker_map map[string]string // t0 -> t3
	if len(miners)==0 && len(req.Workers)!=0 {
		var err error
		if worker_map, err = models.GetMinersByT3(req.Workers); err!=nil {
			resp.Res = resp_search_error
			return resp, nil
		} else {
			miners = make([]string, len(worker_map))
			index := 0
			for t0, _ := range worker_map {
				miners[index] = t0
			}
		}
	}

	miner_rewards_map, err := models.MinerRewardInTimeRange(start, count, miners, is_height)
	if err != nil {
		resp.Res = resp_search_error
		return resp, nil
	}

	// resp.Data = &MinerRewardsResp_Data { }
	miners_rewards := make(map[string]*MinerRewards)

	for addr, re := range miner_rewards_map {

		mrds, exist := miners_rewards[addr]
		if mrds == nil || !exist {
			mrds = &MinerRewards{
				Miner: addr, TotalRewards: 0}
			miners_rewards[addr] = mrds
		}

		if worker_map!=nil {
			if worker, exist := worker_map[addr]; worker!="" && exist {
				mrds.Woker = worker
			}
		}

		for _, xxx := range re.BlockRewards {
			reward_fil := float32(xxx.RewardFil())
			mrds.Items = append(mrds.Items, &MinerRewards_Item{
				Rewards: reward_fil,
				Height:  xxx.Height})
			mrds.TotalRewards = float32(TruncateNaive(float64(mrds.TotalRewards+reward_fil), PrecisionDefault))
		}
	}

	resp.Res = resp_success
	if len(miners_rewards) != 0 {
		resp.Data = &MinerRewardsResp_Data{
			MinerRewards: miners_rewards,
		}
	}
	return resp, nil
}
