package filscaner

import (
	errs "filscan_lotus/error"
	"filscan_lotus/models"
	// "github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"time"
)

// idea of syncing:
// head -> parent -> parent...-> genesis
// tipset cache for un-confrimed tipseds, there is a comfirm number
// save synced state to database
func (fs *Filscaner) run_syncer() error {
re_sync:
	notifs, err := fs.api.ChainNotify(fs.ctx)
	if err != nil {
		time.Sleep(time.Second * 10)
		goto re_sync
	}

	ping := time.NewTicker(time.Second * 30)

	for {
		select {
		case headers, isok := <-notifs:
			{
				if !isok {
					goto re_sync
				}
				fs.handle_new_headers(headers)
			}
		case <-ping.C:
			{
				// health check
				if _, err := fs.api.ID(fs.ctx); err != nil {
					fs.Printf("error, lotus api 'ping' failed, message:%s\n", err.Error())
				}
			}
		case <-fs.ctx.Done():
			{
				ping.Stop()
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
		if err := fs.run_syncer(); err != nil {
			if err == errs.ErrNotifierClosed {
				fs.Printf("who closed fs.api.ChainNotify ????? re_runsyncer\n")
				goto re_runsyncer
			}
			fs.Printf("run_syncer error, message:%s\n", err.Error())
		}

		fs.waitgroup.Done()
	}()
}

func (fs *Filscaner) task_sync_to_genesis(tipset *types.TipSet) {
	fs.synced_tipset_path_list.push_new_path(tipset)
	fs.waitgroup.Add(1)
	go func() {
		latest, err := fs.sync_to_genesis(tipset)
		if latest != nil {
			fs.Printf("‹‹‹‹‹‹‹‹‹‹‹‹‹synced_to_tipst finished: (%d, %s)››››››››››››\n",
				latest.Height(), latest.Key().String())
			fs.Printf("‹‹‹‹‹‹‹‹‹‹‹‹‹synced_to_tipst finished: (%d, %s)››››››››››››\n",
				latest.Height(), latest.Key().String())
		}
		if err != nil {
			fs.Printf("sync to genesis failed,message:%s\n", err.Error())
		}
		fs.waitgroup.Done()
	}()
}

// func (fs *Filscaner) parseAndPostTipsetMessages(hctype string, tipset *types.TipSet) error {
// 	blocks := tipset.Blocks()
//
// 	tipset.Blocks()
// 	var miner_messages = Tipset_miner_messages{
// 		tipset: tipset,
// 		miners: make(map[string]struct{})}
//
// 	for _, b := range blocks {
// 		blockmessages, err := fs.api.ChainGetBlockMessages(fs.ctx, b.Cid())
// 		if err != nil {
// 			fs.Printf("get block(%d, %s) message failed, message:%s\n",
// 				b.Height, b.Messages.String(), err.Error())
// 			continue
// 		}
//
// 		for index, message := range blockmessages.BlsMessages {
// 			if message.Method == 0 {
// 				fs.Printf("block(%d) index(%d) message(%s) is a transform message\n",
// 					b.Height, index, message.Cid().String())
// 				continue
// 			}
// 			actor, _, err := ParseActorMessage(message)
// 			if err != nil {
// 				fs.Printf("parse actor_name message failed, to:%s \n", message.To.String())
// 				continue
// 			} else if actor.Name == "StorageMinerActor" {
// 				miner_messages.miners[message.To.String()] = struct{}{}
// 			}
// 		}
// 	}
//
// 	timestart := time.Now().Unix()
// 	fs.tipset_miner_messages_notifer <- []*Tipset_miner_messages{&miner_messages}
// 	timeend := time.Now().Unix()
// 	fs.Printf("fs.tipset_miner_messages_notifer <- &messages(miner count=%d, tipset_height=%d) use time:%.3f(m)\n",
// 		len(miner_messages.miners), tipset.Height(), float64(timeend-timestart)/60)
// 	return nil
// }

func (fs *Filscaner) Task_StartHandleMinerState() {
	fs.waitgroup.Add(1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				fs.Printf("%v", err)
			}
		}()
		fs.loop_handle_refresh_miner_state()
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

func (fs *Filscaner) refresh_height_state(header_height uint64) {
	fs.mutx_for_numbers.Lock()
	defer fs.mutx_for_numbers.Unlock()

	fs.header_height = header_height

	if fs.header_height > fs.tipset_cache_size {
		fs.safe_height = fs.header_height - fs.tipset_cache_size
		fs.to_sync_header_height = fs.safe_height
	}
}

func (fs *Filscaner) default_handle_message(hctype string, method *MethodCall) {}

func (fs *Filscaner) Task_InitBlockRewards() {
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
