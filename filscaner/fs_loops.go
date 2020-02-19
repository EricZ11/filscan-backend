package filscaner

import (
	"filscan_lotus/models"
	"github.com/filecoin-project/lotus/chain/store"
	"math/big"
	"time"
)

func (fs *Filscaner) display_notifi_(header *store.HeadChange) {
	parent, _ := fs.api.ChainGetTipSet(fs.ctx, header.Val.Parents())
	head, _ := fs.api.ChainHead(fs.ctx)

	fs.Printf("new notify:>>>>%s(%d)<<<< : %s\nparent(%d,%s)\nchain head(%d, %s)",
		header.Type, header.Val.Height(), header.Val.Key().String(),
		parent.Height(), parent.Key().String(),
		head.Height(), head.Key())
}

func (fs *Filscaner) handle_new_headers(headers []*store.HeadChange) {

	for _, header := range headers {
		if header == nil {
			continue
		}

		fs.display_notifi_(header)

		if header.Type == store.HCApply {
			fs.handle_appl_tipset(header.Val, nil)
			fs.last_appl_tipset = header.Val
		}
	}
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
		// 如果出现跳块的情况, 会得到他的parent的tipset, 这里不需要关注
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

func (fs *Filscaner) loop_handle_messages() {
	ticker := time.NewTicker(time.Second * 90)
	for {
		select {
		case miner_messages_list, isok := <-fs.tipset_miner_messages_notifer:
			if !isok {
				fs.Printf("messge notifier is closed stop handle message")
				return
			}

			isNil := false
			if miner_messages_list == nil { // if syncor reached genesis, it send a 'nil' message
				isNil = true
				if tipset_miner_messages, err := fs.list_genesis_miners(); err != nil {
					fs.Printf("list_genesis_miners failed, message:%s\n", err.Error())
				} else {
					miner_messages_list = []*Tipset_miner_messages{tipset_miner_messages}
				}
			}

			for _, miner_messages := range miner_messages_list {
				fs.Printf("handle storage_miner_actor messages at tipset:%d, miner count:%d",
					miner_messages.tipset.Height(), len(miner_messages.miners))
				for address, _ := range miner_messages.miners {
					fs.handle_storage_miner_message(miner_messages.tipset, address)
				}
			}

			if isNil {
				fs.do_upsert_miners()
				if synced_path := fs.synced_tipset_path_list.front_synced_path(); synced_path != nil {
					fs.Printf(`
|‹‹‹‹‹‹‹‹‹‹‹‹‹‹‹‹‹‹‹‹ * successed handled genesis tipset messages * ›››››››››››››››››››››››››|
|      current synced state : head.height:%d, tail.height:%d
|‹‹‹‹‹‹‹‹‹‹‹‹‹‹‹‹‹‹‹‹ * successed handled genesis tipset messages * ›››››››››››››››››››››››››|`,
						synced_path["head.height"], synced_path["tail.height"])
				}
			}
		case <-ticker.C:
			fs.do_upsert_miners()
		case <-fs.ctx.Done():
			fs.Printf("ctx.done, exit loop_handle_messages")
			return
		}
	}
}

func (fs *Filscaner) loop_handle_refresh_miner_state() {
	// 5 分钟触发一次刷新最新状态..
	// ticker := time.NewTicker(time.Second * 300)
	level2_cache := make(map[string]*models.MinerStateAtTipset)
	for {
		select {
		case miner_state, isok := <-fs.miner_state_chan:
			if !isok {
				fs.Printf("messge notifier is closed stop handle message")
				return
			}
			if true {
				fs.handle_miner_state(miner_state)
			} else {
				if miner, exist := level2_cache[miner_state.MinerAddr]; !exist {
					level2_cache[miner_state.MinerAddr] = miner
				}
				if len(level2_cache) > 20 {
					// todo: 批量更新, havn't implemented yet
					fs.handle_miner_state(miner_state)
					for k, _ := range level2_cache {
						delete(level2_cache, k)
					}
				}
			}
		// case <-ticker.C:
		// fs.refresh_miner_caches()
		case <-fs.ctx.Done():
			fs.Printf("ctx.done, exit loop_handle_messages")
			return
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
