package models

import (
	"context"
	"filscan_lotus/utils"
	"fmt"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/types"
	"math/big"

	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/globalsign/mgo"
	"gopkg.in/mgo.v2/bson"
	"time"
)

const TipsetRewardsCollection = "tipset_rewards"

type miners_blocks_rewards struct {
	Miner           string      `bson:"miner"`
	MinedBlockCount uint64      `bson:"mined_block_count"`
	Rewards         *BsonBigint `bson:"rewards"`
}

type TipsetBlockRewards struct {
	TipsetHeight          uint64                            `bson:"tipset_height"`
	TotalBlockCount       uint64                            `bson:"total_block_count"`
	TipsetBlockCount      uint64                            `bson:"tipset_block_count"`
	TimeStamp             uint64                            `bson:"time_stamp"`
	TipsetReward          *BsonBigint                       `bson:"current_tipset_rewards"`
	TotalRealeasedRewards *BsonBigint                       `bson:"chain_released_rewards"`
	Miners                map[string]*miners_blocks_rewards `bson"miners"`
}

func Create_Tipset_Rewards_Index() {
	ms, c := Connect(TipsetRewardsCollection)
	defer ms.Close()
	ms.SetMode(mgo.Monotonic, true)

	indexs := []mgo.Index{
		{Key: []string{"time_stamp"}, Unique: true, Background: true},
		{Key: []string{"miners"}, Unique: false, Background: true},
	}
	for _, index := range indexs {
		if err := c.EnsureIndex(index); err != nil {
			panic(err)
		}
	}
}

func (tbr *TipsetBlockRewards) AddMinedBlock(reward *big.Int, miner_addr string) {
	tbr.TotalBlockCount++
	tbr.TipsetBlockCount++
	tbr.TipsetReward.Add(tbr.TipsetReward.Int, reward)
	tbr.TotalRealeasedRewards.Add(tbr.TotalRealeasedRewards.Int, reward)

	if tbr.Miners == nil {
		tbr.Miners = make(map[string]*miners_blocks_rewards)
	}
	miner, exist := tbr.Miners[miner_addr]
	if !exist || miner == nil {
		miner = &miners_blocks_rewards{
			Rewards: &BsonBigint{Int: big.NewInt(0)},
			Miner:   miner_addr,
		}
		tbr.Miners[miner_addr] = miner
	}
	miner.Rewards.Add(miner.Rewards.Int, reward)
	miner.MinedBlockCount++
}

func last_block_chain_rewards() (*TipsetBlockRewards, error) {
	ms, c := Connect(TipsetRewardsCollection)
	defer ms.Close()

	last_tipset_rewards := &TipsetBlockRewards{}
	err := c.Find(nil).Sort("-tipset_height").Limit(1).One(last_tipset_rewards)
	return last_tipset_rewards, err
}

func blocks_at_height(offset, count uint64) ([]FilscanBlockResult, error) {
	ms, c := connect(BlocksCollection)
	defer ms.Close()

	var res []FilscanBlockResult

	q_find := bson.M{"block_header.Height": bson.M{"$gte": offset, "$lt": offset + count}}
	q_sort := "block_header.Height"

	err := c.Find(q_find).Sort(q_sort).All(&res)
	return res, err
}

func Loop_WalkThroughTipsetRewards(ctx context.Context) error {
	var blocks []FilscanBlockResult
	var err error
	var last_tipset_rewards *TipsetBlockRewards

	last_tipset_rewards, err = last_block_chain_rewards()
	if err != nil {
		if err == mgo.ErrNotFound {
			last_tipset_rewards = &TipsetBlockRewards{
				Miners:                map[string]*miners_blocks_rewards{},
				TipsetHeight:          0,
				TotalBlockCount:       0,
				TipsetReward:          &BsonBigint{Int: big.NewInt(0)},
				TotalRealeasedRewards: &BsonBigint{Int: big.NewInt(0)}}
		} else {
			return err
		}
	}

	remaining_filcoin := types.FromFil(build.TotalFilecoin)
	remaining_filcoin.Sub(remaining_filcoin.Int, last_tipset_rewards.TotalRealeasedRewards.Int)

	for {
		blocks, err = blocks_at_height(last_tipset_rewards.TipsetHeight, 201)

		select {
		case <-ctx.Done():
			return nil
		default:
			if err != nil || len(blocks) == 0 {
				time.Sleep(time.Second * 60)
				continue
			}
		}

		var block_reward types.BigInt
		tipset_rewards_map := make(map[uint64]*TipsetBlockRewards)

		for _, block := range blocks {
			height := block.BlockHeader.Height
			miner_addr := block.BlockHeader.Miner

			tipset_rewards, exist := tipset_rewards_map[height]

			// 由于blocks 是按正序排序的
			// 所以, 只要!exist, 则表明, 块高发生了变化, 需要重新计算爆块奖励
			if !exist || tipset_rewards == nil {
				previouse_tipset_rewards, exist := tipset_rewards_map[height-1]
				if !exist {
					previouse_tipset_rewards = last_tipset_rewards
				}

				block_reward = vm.MiningReward(remaining_filcoin)

				// block.BlockHeader.Timestamp
				tipset_rewards = &TipsetBlockRewards{
					TipsetHeight:          height,
					TotalBlockCount:       previouse_tipset_rewards.TotalBlockCount,
					TipsetBlockCount:      0,
					Miners:                make(map[string]*miners_blocks_rewards),
					TimeStamp:             block.BlockHeader.Timestamp,
					TipsetReward:          &BsonBigint{Int: big.NewInt(0)},
					TotalRealeasedRewards: &BsonBigint{Int: big.NewInt(0).Set(previouse_tipset_rewards.TotalRealeasedRewards.Int)},
				}

				tipset_rewards_map[height] = tipset_rewards
				last_tipset_rewards = tipset_rewards
			}

			// 取最大block.timestamp作为tipset的timestamp
			if tipset_rewards.TimeStamp < block.BlockHeader.Timestamp {
				tipset_rewards.TimeStamp = block.BlockHeader.Timestamp
			}
			tipset_rewards.AddMinedBlock(block_reward.Int, miner_addr)
			remaining_filcoin.Sub(remaining_filcoin.Int, block_reward.Int)
		}

		bulkUpsertTipsetRewards(tipset_rewards_map)
	}
}

func bulkUpsertTipsetRewards(tipset_rewards map[uint64]*TipsetBlockRewards) error {
	size := len(tipset_rewards)
	bulk_elements := make([]interface{}, size*2)
	index := 0

	for k, v := range tipset_rewards {
		bulk_elements[index*2] = bson.M{"tipset_height": k}
		bulk_elements[index*2+1] = v
		index++
	}

	_, err := BulkUpsert(TipsetRewardsCollection, bulk_elements)
	return err
}

type blockAndRewards struct {
	Height uint64
	Reward *big.Int
}

func (br *blockAndRewards) RewardFil() float64 {
	return utils.ToFil(br.Reward)
}

type Models_miner_block_rewards struct {
	Miner           string
	TotalReward     *big.Int
	MinedBlcokCount uint64
	BlockRewards    []*blockAndRewards
}

func (mmbr *Models_miner_block_rewards) AddOneBlockReward(height uint64, reward *big.Int) {
	mmbr.MinedBlcokCount++
	mmbr.TotalReward.Add(mmbr.TotalReward, reward)
	mmbr.BlockRewards = append(mmbr.BlockRewards, &blockAndRewards{
		Height: height,
		Reward: big.NewInt(0).Set(reward)})
}

func MinerRewardInTimeRange(start, diff uint64, miners []string, is_height bool) (map[string]*Models_miner_block_rewards, error) {
	ms, c := connect(TipsetRewardsCollection)
	defer ms.Close()

	var trs []*TipsetBlockRewards

	var field_name string = "time_stamp"
	if is_height {
		field_name = "tipset_height"
	}
	q_match := bson.M{field_name: bson.M{"$gte": start, "$lt": start + diff}}
	q_find := []bson.M{{"$match": q_match}}

	minersize := len(miners)
	if minersize > 0 {
		q_match_or := make([]bson.M, minersize)
		for index, miner := range miners {
			q_match_or[index] = bson.M{fmt.Sprintf(`bson"miners".%s`, miner): bson.M{"$exists": true}}
		}
		q_match["$or"] = q_match_or
	}

	// mgo.SetDebug(true)
	// fmt.Printf("%v\n", q_find)
	err := c.Pipe(q_find).AllowDiskUse().All(&trs)
	// mgo.SetDebug(false)

	mbrm := make(map[string]*Models_miner_block_rewards)
	mm := map[string]struct{}{}

	for _, m := range miners {
		mm[m] = struct{}{}
	}

	for _, trw := range trs { // 遍历tipset
		for miner_addr, mb := range trw.Miners { // 遍历tipset中的矿工奖励
			// 过滤不需要的miner地址.
			if _, exist := mm[miner_addr]; !exist {
				continue
			}
			miner, exist := mbrm[miner_addr]
			if !exist {
				miner = &Models_miner_block_rewards{
					Miner:           miner_addr,
					MinedBlcokCount: 0,
					TotalReward:     big.NewInt(0),
					BlockRewards:    []*blockAndRewards{},
				}
				mbrm[miner_addr] = miner
			}
			miner.AddOneBlockReward(trw.TipsetHeight, mb.Rewards.Int)
		}
	}
	return mbrm, err
}

func GetLatestReward() (string, error) {
	ms, c := connect(TipsetRewardsCollection)
	defer ms.Close()
	var trs []*TipsetBlockRewards
	err := c.Find(nil).Sort("-tipset_height").Limit(1).All(&trs)
	if err != nil {
		return "", err
	}
	if len(trs) == 0 {
		return "", nil
	}
	tipsetReward, err := types.BigFromString(trs[0].TipsetReward.String())
	count := types.NewInt(trs[0].TipsetBlockCount)
	reward := types.FIL(types.BigDiv(tipsetReward, count)).String()
	return reward, nil
}
