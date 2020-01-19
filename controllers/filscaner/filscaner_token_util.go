package filscaner

import (
	"filscan_lotus/models"
	"fmt"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/globalsign/mgo/bson"
	"math/big"
)

const PrecisionDefault = float64(0.00001)

var blocksPerEpoch = big.NewInt(build.BlocksPerEpoch)

func ToFil(v *big.Int) float64 {
	fbig, _ := big.NewFloat(0).SetString(v.String())
	fv, _ := fbig.Float64()
	return TruncateNaive(fv/build.FilecoinPrecision, PrecisionDefault)
}

func ToFilString(v *big.Int) string {
	value := ToFil(v)
	return fmt.Sprintf("%.4f", value)
}

// 返回每个周期中的奖励filcoin数量和释放的奖励数量
func (fs *Filscaner) future_block_rewards(timediff, repeate uint64) ([]*big.Int, *big.Int, error) {
	coffer, err := fs.api.WalletBalance(fs.ctx, actors.NetworkAddress)
	if err != nil {
		return nil, nil, err
	}
	fmt.Printf("\n!!!!!!!net work balance=%.3f\n", ToFil(coffer.Int))

	released := big.NewInt(0).Set(coffer.Int)

	// halving := (start + (timediff * (repeate + 1))) / 30 // 预测时间内的总出块数量
	// coffer = types.FromFil(build.MiningRewardTotal)
	// fmt.Printf("total balance=%.3f\n", ToFil(coffer.Int))

	block_daliy := big.NewInt(2 * 60 * 24) // 每日预计出块数量
	reward_daliy := big.NewInt(0)
	block_diff := timediff / 30

	sums := make([]*big.Int, repeate)
	sum := new(big.Int)

	for i := uint64(0); i < repeate; i++ {
		sums[i] = big.NewInt(0)

		for c := uint64(0); c < block_diff; c += block_daliy.Uint64() {
			a := vm.MiningReward(coffer)
			a.Mul(a.Int, blocksPerEpoch)

			reward_daliy.Mul(a.Int, block_daliy)

			// fmt.Printf("block reward=%.3f, daliy reward=%.3f\n", ToFil(a.Int), ToFil(reward_daliy))

			sum.Add(sum, reward_daliy)

			sums[i].Add(sums[i], reward_daliy)
			coffer.Sub(coffer.Int, reward_daliy)
		}
	}

	released.Add(released, sum)
	return sums, released, nil
}

func SelfTipsetRewards(remainingReward *big.Int) *big.Int {
	remaining := types.NewInt(0)
	remaining.Set(remainingReward)
	rewards := vm.MiningReward(remaining)
	return rewards.Mul(rewards.Int, blocksPerEpoch)
}

func (fs *Filscaner) released_reward_at_height(height uint64) *big.Int {
	release_rewards, err := models_block_released_rewards_at_height(height)
	if err != nil {
		release_rewards = &Models_Block_reward{
			Height:          0,
			ReleasedRewards: &models.BsonBigint{Int: big.NewInt(0)},
		}
	}

	remain_rewards := big.NewInt(0).Sub(TOTAL_REWARDS, release_rewards.ReleasedRewards.Int)
	skiped := height - release_rewards.Height

	rewards := SelfTipsetRewards(remain_rewards)
	rewards.Mul(rewards, big.NewInt(int64(skiped)))

	return rewards.Add(rewards, release_rewards.ReleasedRewards.Int)
}

func (fs *Filscaner) block_reward_at_height(height uint64) (*big.Int, *big.Int) {
	total := big.NewInt(0).Set(TOTAL_REWARDS)
	released := fs.released_reward_at_height(height)
	return released, total.Sub(total, released)
}


func (fs *Filscaner) init_miners() error {
	tipset, err := fs.api.ChainGetGenesis(fs.ctx)
	if err != nil {
		return err
	}
	miners, err := fs.api.StateListMiners(fs.ctx, tipset)
	if err != nil {
		return err
	}

	var size = len(miners)
	var to_updates = make([]interface{}, size*2)
	for index, miner := range miners {
		if m, err := fs.api_get_miner_state_at_tipset(miner, tipset); err != nil {
			continue
		} else {
			to_updates[index*2] = bson.M{"mine_addr": m.MinerAddr, "tipset_height": tipset.Height()}
			to_updates[index*2+1] = m
		}
	}
	return models.BulkUpsertMiners(to_updates)
}
