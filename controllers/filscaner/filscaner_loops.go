package filscaner

import (
	"filscan_lotus/models"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"math/big"
	"time"
)

const HCSafe = "safe"

func (fs *Filscaner) NotifyHeaderChanged_deprecated(chtype string, tipset *types.TipSet) {
	TipsetTime(tipset.MinTimestamp())

	starttime := time.Now().Unix()

	fs.Printf("<<<<<<<<<start NotifyHeaderChanged, chtype:%s, tipset.height:%d\n", chtype, tipset.Height())
	fs.head_notifier <- &store.HeadChange{Val: tipset, Type: chtype}
	endtime := time.Now().Unix()

	timediff := float64(endtime - starttime)
	if timediff > 30 {
		fs.Printf("warning tipset=%d notify very slow..............", tipset.Height())
	}
	fs.Printf("<<<<<<<<<<< successed!!!!!!NotifyHeaderChanged, chtype:%s, tipset.height:%d, used time=%.3f(m)\n",
		chtype, tipset.Height(), (timediff)/60)
}

func (fs *Filscaner) loop_handle_head_change() {
label_for:
	for {
		select {
		case header, isok := <-fs.head_notifier:
			{
				if !isok {
					fs.Printf("head_notifier was closed")
					break label_for
				}
				if header == nil {
					continue
				}

				// 如果是tipset被revert了, 则它和它之后的所有节点应该删除???
				if header.Type == store.HCRevert {
					fs.delete_minerstate_at(header.Val.Height())
					continue
				}

				fs.Printf("on head changed: information:type:%s, tipset.height:%d\n",
					header.Type,
					header.Val.Height())

				fs.parseAndPostTipsetMessages(header.Type, header.Val)
			}
		case <-fs.ctx.Done():
			{
				fs.Printf("run_syncer stoped by ctx.done()")
				break label_for
			}
		}
	}
}

// 每次检查'blocksize'个tipset, 看看是否已经同步!
func (fs *Filscaner) loop_sync_lost_tipsets() error {
	blocksize := uint64(50)

	min_synced_height, _ := models_get_synced_min_height()
	max_synced_height := fs.header_height

	if min_synced_height==0 {
		fs.init_miners()
		min_synced_height = 1
	}

	for index := min_synced_height; index < max_synced_height; index += blocksize {

		if index+blocksize > max_synced_height {
			blocksize = max_synced_height - index
		}

		heights, err := models_miner_heights(index, index+blocksize)
		if err != nil {
			fs.Printf("models_miner_heights faild, message:%s\n", err.Error())
			return err
		}

		unsynced_heights := make([]uint64, blocksize-uint64(len(heights)))
		tmpindex := 0
		for i := uint64(0); i < blocksize; i++ {
			_, exist := heights[index+i]
			if !exist {
				unsynced_heights[tmpindex] = index + i
				tmpindex++
			}
		}
		select {
		case <-fs.ctx.Done():
			return nil
		default:
			if err := fs.sync_tipset_with_heights(unsynced_heights); err == nil {
				if err = models_upsert_synced_height(index + blocksize); err != nil {
					fs.Printf("upsert syned height(%d) failed, message:%s\n", index+blocksize, err.Error())
				}
			} else {
				fs.Printf("sync tipset with height(%d) failed, message:%s\n", index+blocksize, err.Error())
			}
		}
	}
	return nil
}

func (fs *Filscaner) sync_tipset_with_heights(heights []uint64) error {
	for _, height := range heights {
		tipset, err := fs.api.ChainGetTipSetByHeight(fs.ctx, height, nil)
		if err != nil {
			//TODO:判断是否为lotus节点crush掉, 或者是网络问题,
			// 如果是, 应该过一段时间后, 重新尝试获取
			fs.Printf("chain_get_tipset_by_height failed, message:%s\n", err.Error())
			continue
		}
		// 如果出现跳块的情况, 会得到他的parant的tipset, 这里不需要关注
		// if tipset.Height() < height { }
		select {
		case <-fs.ctx.Done():
			return fs.ctx.Err()
		default:
			fs.head_notifier <- &store.HeadChange{Val: tipset, Type: store.HCApply}
		}
		// time.Sleep(time.Millisecond * 200)
	}
	return nil
}

// StorageMinerActor
// 	1: sma.StorageMinerConstructor,
// 	2: sma.PreCommitSector,
// 	3: sma.ProveCommitSector,
// StoragePowerActor
// 	2: spa.CreateStorageMiner,
func (fs *Filscaner) loop_handle_messages() {
	for {
		select {
		case miner_messages, isok := <-fs.tipset_miner_messages_notifer:
			if !isok {
				fs.Printf("messge notifier is closed stop handle message")
				return
			}

			// 统计矿工算力的情况
			for address, _ := range miner_messages.miners {
				fs.handle_storage_miner_message(store.HCApply, miner_messages.tipset, address)
			}
		case <-fs.ctx.Done():
			{
				fs.Printf("ctx.done, exit loop_handle_messages")
				return
			}
		}
	}
}

func (fs *Filscaner) loop_init_block_rewards() {
	tipset, err := fs.api.ChainHead(fs.ctx)
	if err != nil {
		return
	}

	head_block_rewards, err := models_block_reward_head()
	if err != nil {
		return
	}

	head_height := tipset.Height()
	total_rewards := big.NewInt(0).Sub(TOTAL_REWARDS, head_block_rewards.ReleasedRewards.Int)

	bulk_size := 20
	upsert_rewards := make([]*Models_Block_reward, bulk_size)
	offset := 0

	released_rewards := head_block_rewards.ReleasedRewards.Int

	// 每25个height间隔一个数据库记录
	// 每20 * 25个height间隔保存一次
	for i := head_block_rewards.Height; i < head_height; i++ {
		rewards := SelfTipsetRewards(total_rewards)

		total_rewards.Sub(total_rewards, rewards)
		released_rewards.Add(released_rewards, rewards)

		if i%25 == 0 {
			upsert_rewards[offset] = &Models_Block_reward{
				Height:          i,
				ReleasedRewards: &models.BsonBigint{Int: released_rewards}}
			offset++

			if offset == bulk_size {
				err := models_bulk_upsert_block_reward(upsert_rewards, offset-1)
				if err != nil {
					// TODO: handle error
				}
				offset = 0
				time.Sleep(time.Millisecond * 500)
			}
		}
	}

	if offset != 0 {
		err := models_bulk_upsert_block_reward(upsert_rewards, offset-1)
		if err != nil {
		}
	}
}
