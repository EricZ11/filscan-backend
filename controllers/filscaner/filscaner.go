package filscaner

import (
	"context"
	"filscan_lotus/controllers/filscaner/force/factors"
	"filscan_lotus/utils"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/globalsign/mgo"
	"math/big"
	"sync"

	// "github.com/filecoin-project/lotus/chain/types"
	internalerr "filscan_lotus/error"
)

const CONFIRM_TICSET_SIZE = 500
const MINER_CACHE_SIZE = 200

type BlockMessages struct {
	// HCRevert  = "revert"
	// HCApply   = "apply"
	// HCCurrent = "current"
	hc_type string
	tipset  *types.TipSet
	methods []*MethodCall
}

type TipsetMinerMessages struct {
	miners map[string]struct{}
	tipset *types.TipSet
}

type MethodCall struct {
	actor_name string
	*types.Message
	*factors.MethodInfo
}

type Filscaner struct {
	api api.FullNode

	ctx    context.Context
	cancel context.CancelFunc

	head_notifier                 chan *store.HeadChange
	tipset_miner_messages_notifer chan *TipsetMinerMessages

	// 已经同步到的tipset高度,当程序重启时,
	// 需要从此高度同步到first_notifiedTipsetHeight
	synced_height uint64
	safe_height   uint64
	header_height uint64

	miners             *Miners
	chain_genisis_time uint64

	waitgroup sync.WaitGroup

	colation *mgo.Collation

	to_upsert_miners      []interface{}
	to_update_miner_size  uint64
	to_update_miner_index uint64

	latest_total_power *big.Int

	dispaly_tracs bool
}

var FilscanerInst = &Filscaner{}

// func Init(ctx context.Context, lotusApi api.FullNode) error {
// 	return FilscanerInst.Init(ctx, lotusApi)
// }

func NewInstance(ctx context.Context, lotusApi api.FullNode) (error, *Filscaner) {
	filscaner := &Filscaner{}
	if err := filscaner.Init(ctx, lotusApi); err != nil {
		return err, nil
	}
	return nil, filscaner
}

func (fs *Filscaner) Init(ctx context.Context, lotusApi api.FullNode) error {
	if lotusApi == nil {
		return internalerr.ErrInvalidParam
	}

	fs.ctx, fs.cancel = context.WithCancel(ctx)

	fs.head_notifier = make(chan *store.HeadChange)
	fs.tipset_miner_messages_notifer = make(chan *TipsetMinerMessages)

	fs.miners = new_sorted_miners(MINER_CACHE_SIZE)

	fs.to_update_miner_size = 256
	fs.to_update_miner_index = 0
	fs.to_upsert_miners = make([]interface{}, fs.to_update_miner_size*2)

	fs.api = lotusApi

	fs.dispaly_tracs = true

	fs.colation = &mgo.Collation{Locale: "zh", NumericOrdering: true}

	if err := fs.ini_ChainGenisisTime(); err!=nil {
		return err
	}

	if tipset, err := fs.api.ChainHead(ctx); err != nil {
		return err
	} else {
		fs.refresh_height_state(tipset.Height())
	}

	// if err := fs.init_synced_heigth(); err != nil {
	// 	return err
	// }

	return nil
}

func (fs *Filscaner) Printf(fmts string, args ...interface{}) {
	if !fs.dispaly_tracs {
		return
	}
	utils.Printf("filscaner", fmts, args[:]...)
}

func (fs *Filscaner) Run() {
	for i := 0; i < 4; i++ {
		fs.Task_StartHandleMessage()
	}
	fs.Task_StartHandleHeadChange()
	fs.Task_StartSyncer()

	fs.Task_StartSyncLostTipsets()

	fs.Task_Init_blockrewards()

	fs.Task_SyncTipsetRewardsDb()
}

func (fs *Filscaner) ini_ChainGenisisTime() error {
	if fs.chain_genisis_time != 0 {
		return nil
	}

	genesis, err := fs.api.ChainGetGenesis(fs.ctx)
	if err != nil {
		return err
	}

	fs.chain_genisis_time = genesis.MinTimestamp()
	return nil
}
