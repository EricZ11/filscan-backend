package main

import (
	"filscan_lotus/utils"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"time"
)

func filcoin_check_blockreward() {
	hitipset, err := lotus_api.ChainHead(ctx)
	if err != nil {
		fmt.Printf("err, message:%s\n", err.Error())
		return
	}
	blocks := hitipset.Blocks()
	miners := make(map[string]*struct {
		worker      string
		new_balance types.BigInt
		old_balance types.BigInt
		blocks      []*types.BlockHeader
	})

	for _, b := range blocks {
		s, isok := miners[b.Miner.String()]
		if !isok || s == nil {
			s = &struct {
				worker      string
				new_balance types.BigInt
				old_balance types.BigInt
				blocks      []*types.BlockHeader
			}{new_balance: types.NewInt(0), blocks: nil}
		}

		worker, _ := lotus_api.StateMinerWorker(ctx, b.Miner, hitipset)
		s.worker = worker.String()
		s.new_balance, _ = lotus_api.WalletBalance(ctx, worker)
		s.blocks = append(s.blocks, b)

		miners[b.Miner.String()] = s
	}

	lowtipset, err := lotus_api.ChainGetTipSetByHeight(ctx, hitipset.Height()-1, nil)
	if err != nil {
		fmt.Printf("err, message:%s\n", err.Error())
		return
	}
	err = lotus_api.ChainSetHead(ctx, lowtipset)
	if err != nil {
		fmt.Printf("err, message:%s\n", err.Error())
		return
	}

	for _, m := range miners {
		w, _ := address.NewFromString(m.worker)
		m.old_balance, _ = lotus_api.WalletBalance(ctx, w)
	}

	for _, m := range miners {
		fmt.Sprintf("%v\n", m)
	}
}

func filcoin_balances() {
	var balance types.BigInt
	// TODO: the spec says to also grab 'total vested filecoin' and include it as available
	// If we don't factor that in, we effectively assume all of the locked up filecoin is 'available'
	// the blocker on that right now is that its hard to tell how much filecoin is unlocked
	balance, _ = lotus_api.WalletBalance(ctx, actors.NetworkAddress)
	fmt.Printf("wallet get networkaddress new_balance=%.6f\n", utils.ToFil(balance.Int))

	if actor, err := lotus_api.StateGetActor(ctx, actors.NetworkAddress, nil); err != nil {
		fmt.Printf("get actor failed,message:%s\n", err.Error())
	} else {
		fmt.Printf("actor new_balance is :%.3f\n", utils.ToFil(actor.Balance.Int))
	}

	availableFilecoin := types.BigSub(
		types.BigMul(types.NewInt(build.TotalFilecoin), types.NewInt(build.FilecoinPrecision)), balance)
	fmt.Printf("avaliable filcoin new_balance=%.6f\n", utils.ToFil(availableFilecoin.Int))

	totalPowerCollateral := types.BigDiv(
		types.BigMul(
			availableFilecoin,
			types.NewInt(build.PowerCollateralProportion),
		),
		types.NewInt(build.CollateralPrecision),
	)
	fmt.Printf("total power collateral:%.6f\n", utils.ToFil(totalPowerCollateral.Int))

	totalPerCapitaCollateral := types.BigDiv(
		types.BigMul(
			availableFilecoin,
			types.NewInt(build.PerCapitaCollateralProportion),
		),
		types.NewInt(build.CollateralPrecision),
	)
	fmt.Printf("total per capita collateral:%.6f\n", utils.ToFil(totalPerCapitaCollateral.Int))

	// REVIEW: for bootstrapping purposes, we skip the power portion of the
	// collateral if there is no collateral in the network yet
	// powerCollateral := types.NewInt(0)
	// if types.BigCmp(totalStorage, types.NewInt(0)) != 0 {
	// 	powerCollateral = types.BigDiv(
	// 		types.BigMul(
	// 			totalPowerCollateral,
	// 			size,
	// 		),
	// 		totalStorage,
	// 	)
	// }
	//
	// perCapCollateral := types.BigDiv(
	// 	totalPerCapitaCollateral,
	// 	types.NewInt(minerCount),
	// )

	// return types.BigAdd(powerCollateral, perCapCollateral), nil
	fmt.Printf("total new_balance=%.6f\n", utils.ToFil(balance.Int))
	balance, _ = lotus_api.WalletBalance(ctx, actors.StorageMarketAddress)
	fmt.Printf("total new_balance=%.6f\n", utils.ToFil(balance.Int))
	balance, _ = lotus_api.WalletBalance(ctx, actors.StoragePowerAddress)
	fmt.Printf("total new_balance=%.6f\n", utils.ToFil(balance.Int))
	balance, _ = lotus_api.WalletBalance(ctx, actors.CronAddress)
	fmt.Printf("total new_balance=%.6f\n", utils.ToFil(balance.Int))
	balance, _ = lotus_api.WalletBalance(ctx, actors.BurntFundsAddress)
	fmt.Printf("total new_balance=%.6f\n", utils.ToFil(balance.Int))

	address, _ := address.NewFromString("t0222")

	bbb, _ := lotus_api.StateMarketBalance(ctx, address, nil)
	fmt.Printf("address:%s, avalable new_balance : %.3f, locked new_balance : %.3f\n",
		address.String(), utils.ToFil(bbb.Available.Int), utils.ToFil(bbb.Locked.Int))
}

/*
func reword_blocks() {
	rewards, _, err := fscaner.future_block_rewards(60*60*24*356, 50)
	if err != nil {
		fmt.Printf("err, message:%s\n", err.Error())
	}

	total := big.NewInt(0)
	for index, r := range rewards {
		total.Add(total, r)
		fmt.Printf("day index :%d, reward=%s\n", index, utils.ToFilString(r))
	}
	fmt.Printf("week(%d) days reward=%.3f\n", len(rewards), utils.ToFil(total))
}
// */

/*
func miner_list() {
	v := 8900000000000000000000000000000000000.134356565643423
	v_str := strconv.FormatFloat(v, 'f', -1, 64)
	fmt.Printf("%s", v_str)
	b_v, _ := big.NewInt(0).SetString(v_str, 10)
	fmt.Printf("%s", b_v.String())

	time_end := uint64(time.Now().Unix())
	time_start := uint64(0) // time_end - (60 * 60 * 24)

	miner_list := []string{"t06241", "t01475", "t01493", "t06594", "t12345"}

	mgo.SetDebug(true)
	miners, err := fscaner.models_miner_list_sort_block(nil, time_start, time_end, 0, 5, "mining_efficiency", -1)
	if err!=nil {
		fmt.Print("err:%s\n", err.Error())
	} else {
		fmt.Printf("miners:%v\n", miners)
	}
	miners, err = fscaner.models_minerlist_sort_power(miner_list, time_start, time_end, 0, 5, "power_rate", -1)
	if err!=nil {
		fmt.Print("err:%s\n", err.Error())
	} else {
		fmt.Printf("miners:%v\n", miners)
	}
	mgo.SetDebug(false)
}
// */

func block_rewards(tipset *types.TipSet) {
	actor, _ := lotus_api.StateGetActor(ctx, actors.NetworkAddress, tipset)
	rewards := vm.MiningReward(actor.Balance)
	fmt.Printf("block rewards at tipset:%d = %.8f\n", tipset.Height(), utils.ToFil(rewards.Int))
}

func check_block_rewards() {
	var tipset *types.TipSet = nil
	var err error
	if tipset, err = lotus_api.ChainHead(ctx); err != nil {
		fmt.Printf("tipset is : %s, %d\n", tipset.Key().String(), tipset.Height())
	}

	block_rewards(tipset)

	miner := tipset.Blocks()[0].Miner
	worker, _ := lotus_api.StateMinerWorker(ctx, miner, tipset)
	fmt.Printf("miner:%s, worker is :%s\n", miner.String(), worker.String())

	balance1, err := lotus_api.WalletBalance(ctx, worker)
	fmt.Printf("miner : %s, balance at tipset:%d is : %.6f\n", miner.String(), tipset.Height(), utils.ToFil(balance1.Int))

	old_height := tipset.Height()
	for {
		tipset, _ = lotus_api.ChainHead(ctx)
		if old_height < tipset.Height() {
			balance2, _ := lotus_api.WalletBalance(ctx, worker)
			fmt.Printf("miner : %s, balance at tipset:%d is : %.6f, block reward is:%.6f\n", miner.String(), tipset.Height(), utils.ToFil(balance2.Int), utils.ToFil(balance2.Sub(balance2.Int, balance1.Int)))
			break
		} else {
			fmt.Printf("head didn't increased, sleep 1 second...\n")
			time.Sleep(time.Second)
		}
	}
}

func main() {
	check_block_rewards()
	filcoin_balances()
}
