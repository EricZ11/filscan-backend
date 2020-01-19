package main

import (
	"context"
	"filscan_lotus/controllers"
	"filscan_lotus/controllers/filscaner"
	"filscan_lotus/models"
	"filscan_lotus/utils"
	"fmt"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/actors"
	address2 "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/globalsign/mgo"
)

var fscaner *filscaner.Filscaner
var ctx context.Context
var cancel context.CancelFunc
var lotusApi api.FullNode

var sm_API api.StorageMiner

// func NewStorageMinerRPC(addr string, requestHeader http.Header) (api.StorageMiner, jsonrpc.ClientCloser, error) {

var rpc_url = "127.0.0.1:1234"

// type MongoLog struct{}
//
// func (MongoLog) Output(calldepth int, s string) error {
// 	log.SetFlags(log.Lshortfile)
// 	return log.Output(calldepth, s)
// }
//

func InitMongoLogs() {
	mgo.SetLogger(new(MongoLog))
}

func init() {
	ctx, cancel = context.WithCancel(context.TODO())

	controllers.BeegoInit()
	controllers.LotusInit()
	controllers.MongoDBInit()
	models.TimenowInit()
	controllers.LoggerInit()
	// controllers.FirstSynLotus()
	controllers.ArgInit()

	lotusApi = controllers.LotusApi

	var err error
	sm_API, _, err = client.NewStorageMinerRPC("ws://"+rpc_url+"/rpc/v0", nil)
	if err != nil {
		fmt.Printf("new storage miner failed, message:%s\n", err.Error())
	}

	if address, err := sm_API.ActorAddress(ctx); err != nil {
		fmt.Printf("actor address failed, message:%s\n", err.Error())
	} else {
		fmt.Printf("actor address is :%s\n", address.String())
	}

	// address, _ := address.NewFromString("t02718")
	err, fscaner = filscaner.NewInstance(ctx, controllers.LotusApi)
	if err != nil {
		utils.Printf("test error", "create filscaner faild, message:%s\n", err.Error())
		return
	}
	InitMongoLogs()
}

func filcoin_check_blockreward() {
	hitipset, err := lotusApi.ChainHead(ctx)
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

		worker, _ := lotusApi.StateMinerWorker(ctx, b.Miner, hitipset)
		s.worker = worker.String()
		s.new_balance, _ = lotusApi.WalletBalance(ctx, worker)
		s.blocks = append(s.blocks, b)

		miners[b.Miner.String()] = s
	}

	lowtipset, err := lotusApi.ChainGetTipSetByHeight(ctx, hitipset.Height()-1, nil)
	if err != nil {
		fmt.Printf("err, message:%s\n", err.Error())
		return
	}
	err = lotusApi.ChainSetHead(ctx, lowtipset)
	if err != nil {
		fmt.Printf("err, message:%s\n", err.Error())
		return
	}

	for _, m := range miners {
		w, _ := address2.NewFromString(m.worker)
		m.old_balance, _ = lotusApi.WalletBalance(ctx, w)
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
	balance, _ = lotusApi.WalletBalance(ctx, actors.NetworkAddress)
	fmt.Printf("wallet get networkaddress new_balance=%.6f\n", filscaner.ToFil(balance.Int))

	if actor, err := lotusApi.StateGetActor(ctx, actors.NetworkAddress, nil); err != nil {
		fmt.Printf("get actor failed,message:%s\n", err.Error())
	} else {
		fmt.Printf("actor new_balance is :%.3f\n", filscaner.ToFil(actor.Balance.Int))
	}

	availableFilecoin := types.BigSub(
		types.BigMul(types.NewInt(build.TotalFilecoin), types.NewInt(build.FilecoinPrecision)), balance)
	fmt.Printf("avaliable filcoin new_balance=%.6f\n", filscaner.ToFil(availableFilecoin.Int))

	totalPowerCollateral := types.BigDiv(
		types.BigMul(
			availableFilecoin,
			types.NewInt(build.PowerCollateralProportion),
		),
		types.NewInt(build.CollateralPrecision),
	)
	fmt.Printf("total power collateral:%.6f\n", filscaner.ToFil(totalPowerCollateral.Int))

	totalPerCapitaCollateral := types.BigDiv(
		types.BigMul(
			availableFilecoin,
			types.NewInt(build.PerCapitaCollateralProportion),
		),
		types.NewInt(build.CollateralPrecision),
	)
	fmt.Printf("total per capita collateral:%.6f\n", filscaner.ToFil(totalPerCapitaCollateral.Int))

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
	fmt.Printf("total new_balance=%.6f\n", filscaner.ToFil(balance.Int))
	balance, _ = lotusApi.WalletBalance(ctx, actors.StorageMarketAddress)
	fmt.Printf("total new_balance=%.6f\n", filscaner.ToFil(balance.Int))
	balance, _ = lotusApi.WalletBalance(ctx, actors.StoragePowerAddress)
	fmt.Printf("total new_balance=%.6f\n", filscaner.ToFil(balance.Int))
	balance, _ = lotusApi.WalletBalance(ctx, actors.CronAddress)
	fmt.Printf("total new_balance=%.6f\n", filscaner.ToFil(balance.Int))
	balance, _ = lotusApi.WalletBalance(ctx, actors.BurntFundsAddress)
	fmt.Printf("total new_balance=%.6f\n", filscaner.ToFil(balance.Int))

	address, _ := address2.NewFromString("t0222")

	bbb, _ := lotusApi.StateMarketBalance(ctx, address, nil)
	fmt.Printf("address:%s, avalable new_balance : %.3f, locked new_balance : %.3f\n",
		address.String(), filscaner.ToFil(bbb.Available.Int), filscaner.ToFil(bbb.Locked.Int))
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
		fmt.Printf("day index :%d, reward=%s\n", index, filscaner.ToFilString(r))
	}
	fmt.Printf("week(%d) days reward=%.3f\n", len(rewards), filscaner.ToFil(total))
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

func main() {
	// miner_list()
	// return
	// reword_blocks()
	// return

	filcoin_balances()
	return

	tipset, err := lotusApi.ChainHead(ctx)
	if err != nil {
		fmt.Printf("api chain head failed, message:%s\n", err.Error())
		return
	}

	tipset_height := tipset.Height()
	for i := 0; i < 100; i++ {
		tipset_height--
		tipset, err := lotusApi.ChainGetTipSetByHeight(ctx, tipset_height, nil)
		if err != nil {
			fmt.Printf("api chain head failed, message:%s\n", err.Error())
			return
		}

		conllateral, err := lotusApi.StatePledgeCollateral(ctx, tipset)

		if err != nil {
			fmt.Printf("api chain head failed, message:%s\n", err.Error())
			return
		}

		fmt.Printf("tipset_height : %d, conllateral value:%s\n", tipset.Height(), conllateral.String())
		filvalue := filscaner.ToFil(conllateral.Int)
		fmt.Printf("conllateral value:%.6f\n", filvalue)
	}
}
