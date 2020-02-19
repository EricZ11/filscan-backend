package filscaner

import (
	"context"
	errs "filscan_lotus/error"
	"filscan_lotus/filscaner/force/factors"
	"filscan_lotus/models"
	"filscan_lotus/utils"
	"github.com/astaxie/beego/config"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/globalsign/mgo"
	"math/big"
	"sync"
)

type Tipset_miner_messages struct {
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

	conf config.Configer

	head_notifier                 chan *store.HeadChange
	tipset_miner_messages_notifer chan []*Tipset_miner_messages

	// 已经同步到的tipset高度,当程序重启时,
	// 需要从此高度同步到first_notifiedTipsetHeight
	tipset_cache_size     uint64
	to_sync_header_height uint64
	safe_height           uint64
	header_height         uint64
	mutx_for_numbers      sync.Mutex

	chain_genesis_time uint64

	waitgroup sync.WaitGroup

	colation *mgo.Collation

	to_upsert_miners      []interface{}
	to_update_miner_size  uint64
	to_update_miner_index uint64

	latest_total_power *big.Int

	dispaly_tracs bool

	////////////////////////////////////////////////////
	synced_tipset_path_list *fs_synced_tipset_path_list // tipset synced status loaded from database
	tipsets_cache           *Fs_tipset_cache            // un-confrimed tipses in front of chain head
	safe_tipset_channel     chan *types.TipSet          //

	last_safe_tipset          *types.TipSet
	last_appl_tipset          *types.TipSet
	is_sync_to_genesis_runing bool
	handle_appl_tipset        func(child, parent *types.TipSet)
	handle_safe_tipset        func(blockmessage *Tipset_block_messages)
	////////////////////////////////////////////////////
	miner_cache24h   *fs_miner_cache
	miner_cache1day  *fs_miner_cache
	// miner_cache1mon  *fs_miner_cache
	miner_state_chan chan *models.MinerStateAtTipset
}

var FilscanerInst = &Filscaner{}

func NewInstance(ctx context.Context, config_path string, lotusApi api.FullNode) (*Filscaner, error) {
	filscaner := &Filscaner{}
	if err := filscaner.Init(ctx, config_path, lotusApi); err != nil {
		return nil, err
	}
	return filscaner, nil
}

func (fs *Filscaner) init_configuration(filepath string) error {
	var err error
	var cache_size int64
	if fs.conf, err = config.NewConfig("ini", filepath); err != nil {
		return err
	}
	if cache_size, err = fs.conf.Int64("tipset_cache_size"); err != nil || cache_size < 0 {
		return err
	}
	fs.tipset_cache_size = uint64(cache_size)
	return nil
}

func (fs *Filscaner) List() *Fs_tipset_cache {
	return fs.tipsets_cache
}

func (fs *Filscaner) init_lotus_client(lotus_api api.FullNode) error {
	if lotus_api == nil {
		return errs.ErrInvalidParam
	}
	fs.api = lotus_api
	if err := fs.ini_ChaingenesisTime(); err != nil {
		return err
	}

	tipset, err := fs.api.ChainHead(context.TODO())
	if err != nil {
		return err
	}
	fs.refresh_height_state(tipset.Height())
	return nil
}

func (fs *Filscaner) Init(ctx context.Context, config_path string, lotusApi api.FullNode) error {
	fs.ctx, fs.cancel = context.WithCancel(ctx)

	var err error
	if err = fs.init_configuration(config_path); err != nil {
		return err
	}

	if err = fs.init_lotus_client(lotusApi); err != nil {
		return err
	}

	fs.head_notifier = make(chan *store.HeadChange)
	fs.tipset_miner_messages_notifer = make(chan []*Tipset_miner_messages)

	fs.to_update_miner_size = 512
	fs.to_update_miner_index = 0
	fs.to_upsert_miners = make([]interface{}, fs.to_update_miner_size*2)

	fs.dispaly_tracs = true

	fs.colation = &mgo.Collation{Locale: "zh", NumericOrdering: true}

	if fs.synced_tipset_path_list, err = models_new_synced_tipset_list(); err != nil {
		return err
	}

	fs.tipsets_cache = new_fs_cache(int(fs.tipset_cache_size))
	fs.safe_tipset_channel = make(chan *types.TipSet, 100)

	fs.handle_safe_tipset = fs.handle_first_safe_tipset
	fs.handle_appl_tipset = fs.handle_first_appl_tipset

	if err := fs.init_miners_caches(); err != nil {
		return err
	}

	return nil
}

func (fs *Filscaner) Printf(fmts string, args ...interface{}) {
	if !fs.dispaly_tracs {
		return
	}
	utils.Printf("filscaner", fmts, args[:]...)
}

func (fs *Filscaner) Error(fmts string, args ...interface{}) {
	if !fs.dispaly_tracs {
		return
	}
	utils.Printf("filscaner", fmts, args[:]...)

}

func (fs *Filscaner) Run() {
	fs.Task_StartHandleMinerState()
	fs.Task_StartHandleMessage()
	fs.Task_StartSyncer()
	// fs.Task_StartSyncLostTipsets()
}

func (fs *Filscaner) ini_ChaingenesisTime() error {
	if fs.chain_genesis_time != 0 {
		return nil
	}

	genesis, err := fs.api.ChainGetGenesis(fs.ctx)
	if err != nil {
		return err
	}

	fs.chain_genesis_time = genesis.MinTimestamp()
	return nil
}
