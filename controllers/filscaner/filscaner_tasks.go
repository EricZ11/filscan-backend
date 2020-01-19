package filscaner

import (
	"filscan_lotus/controllers/filscaner/force/factors"
	innererr "filscan_lotus/error"
	"filscan_lotus/models"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"time"
)

func (fs *Filscaner) run_syncer() error {
	re_sync:
	notifs, err := fs.api.ChainNotify(fs.ctx)
	if err != nil {
		time.Sleep(time.Second * 10)
		goto re_sync
	}

	for {
		select {
		case headers, isok := <-notifs:
			{
				if !isok {
					// notifs被关闭,可能是连接断开, 此时
					// fs.ctx.Done并没有关闭, 所以, 需要
					// 重新启动同步!
					goto re_sync
				}

				for _, header := range headers {
					if header.Type == store.HCApply {
						fs.header_height = header.Val.Height()
					}
					fs.Printf("get new header = %d\n", header.Val.Height())
					fs.head_notifier <- header
				}
			}
		case <-fs.ctx.Done():
			{
				fs.Printf("run_syncer stoped by ctx.done()")
				return fs.ctx.Err()
			}
		}

	}
	return nil
}

func (fs *Filscaner) Task_StartSyncer() {
	fs.waitgroup.Add(1)
	go func() {
	re_runsyncer:
		var err = fs.run_syncer()
		if err != nil {
			if err == innererr.ErrNotifierClosed {
				fs.Printf("who closed fs.api.ChainNotify ????? re_runsyncer\n")
				goto re_runsyncer
			}
			fs.Printf("run_syncer error, message:%s\n", err.Error())
		}

		fs.waitgroup.Done()
	}()
}

func (fs *Filscaner) ParseActorMessage(message *types.Message) (*factors.ActorInfo, *factors.MethodInfo, error) {

	// cid := actors.StorageMarketCodeCid.String()
	if message.Method == 0 {
		return nil, nil, innererr.ErrActorNotFound
	}
	actor, exist := factors.LookupByAddress(message.To)
	if !exist {
		if actor, exist = factors.Lookup(actors.StorageMinerCodeCid); !exist {
			return nil, nil, innererr.ErrActorNotFound
		}
	}

	// actor_str := message.To.String()
	// if !strings.HasPrefix(actor_str, "t0"){
	// 	fs.Printf("address(%s) not a miner address", actor_str)
	// 	return nil
	// }

	method, exist := actor.LookupMethod(message.Method)
	if !exist {
		return nil, nil, innererr.ErrMethodNotFound
	}

	return &actor, &method, nil

	// paramater := method.NewParam()
	// if err := vm.DecodeParams(message.Params, paramater); err != nil {
	// 	fs.Printf("decode block(%d) message_index(%d) failed, actor_name(%s) method(%s) parameter failed, message:%s",
	// 		b.Height, index, message.To, method.Name, err.Error())
	// 	return nil
	// }
	// sector, isok := paramater(*actors.StorageMinerConstructorParams)
	// sector, isok := paramater.(*actors.SectorProveCommitInfo);
	// sector, isok := paramater.(*actors.SectorPreCommitInfo);
}

func (fs *Filscaner) parseAndPostTipsetMessages(hc_type string, tipset *types.TipSet) error {
	blocks := tipset.Blocks()

	var miner_messages = TipsetMinerMessages{
		tipset: tipset,
		miners: make(map[string]struct{})}

	for _, b := range blocks {
		blockmessages, err := fs.api.ChainGetBlockMessages(fs.ctx, b.Cid())
		if err != nil {
			fs.Printf("get block(%d, %s) message failed, message:%s\n",
				b.Height, b.Messages.String(), err.Error())
			continue
		}

		for index, message := range blockmessages.BlsMessages {
			if message.Method == 0 {
				fs.Printf("block(%d) index(%d) message(%s) is a transform message\n",
					b.Height, index, message.Cid().String())
				continue
			}
			actor, _, err := fs.ParseActorMessage(message)
			if err != nil {
				fs.Printf("parse actor_name message failed, to:%s \n", message.To.String())
				continue
			} else if actor.Name == "StorageMinerActor" {
				miner_messages.miners[message.To.String()] = struct{}{}
			}
		}
	}

	timestart := time.Now().Unix()
	fs.tipset_miner_messages_notifer <- &miner_messages
	timeend := time.Now().Unix()
	fs.Printf("fs.tipset_miner_messages_notifer <- &messages(miner count=%d, tipset_height=%d) use time:%.3f(m)\n",
		len(miner_messages.miners), tipset.Height(), float64(timeend-timestart)/60)
	return nil
}

func (fs *Filscaner) Task_StartHandleHeadChange() {
	fs.waitgroup.Add(1)
	go func() {
		fs.loop_handle_head_change()
		fs.waitgroup.Done()
		fs.Printf("Task_OnTipsetApplied stoped")
	}()
}

func (fs *Filscaner) Task_StartSyncLostTipsets() {
	fs.waitgroup.Add(1)
	go func() {
		fs.loop_sync_lost_tipsets()
		fs.waitgroup.Done()
	}()
}

func (fs *Filscaner) Task_StartHandleMessage() {
	fs.waitgroup.Add(1)
	go func() {
		fs.loop_handle_messages()
		fs.waitgroup.Done()
	}()
}

func (fs *Filscaner) refresh_height_state(newHeight uint64) {
	if newHeight > fs.header_height {
		fs.header_height = newHeight
		if fs.header_height > CONFIRM_TICSET_SIZE {
			fs.safe_height = fs.header_height - CONFIRM_TICSET_SIZE
		}
	}
}

func (fs *Filscaner) default_handle_message(hctype string, method *MethodCall) {}

func (fs *Filscaner) Task_Init_blockrewards() {
	fs.waitgroup.Add(1)
	go func() {
		fs.loop_init_block_rewards()
		fs.waitgroup.Done()
	}()
}

func (fs *Filscaner) Task_SyncTipsetRewardsDb() {
	fs.waitgroup.Add(1)
	go func() {
		models.Loop_WalkThroughTipsetRewards(fs.ctx)
		fs.waitgroup.Done()
	}()
}
