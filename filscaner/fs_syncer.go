package filscaner

import (
	"fmt"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"time"
)

// TODO: use fs.api.StateChangedActors(),
//  to sync miner state change information
func (fs *Filscaner) sync_to_genesis(from *types.TipSet) (*types.TipSet, error) {
	const max_size = 25

	var child = from
	var parent *types.TipSet = nil
	var tbml = &Tipset_block_message_list{}
	var tpst_blms *Tipset_block_messages
	var err error

	var cids_size = len(child.Parents().Cids())

	var begin_time = time.Now().Unix()
	var begin_height = child.Height()
	for cids_size != 0 { // genesis case 'bafy2bzaceaxm23epjsmh75yvzcecsrbavlmkcxnva66bkdebdcnyw3bjrc74u'
		parent, err = fs.api.ChainGetTipSet(fs.ctx, child.Parents())
		if err != nil {
			fs.Printf("error:sync_to_genesis will exit, message:%s\n", err.Error())
			return nil, err
		}
		fs.Printf("‹syncinfo›:start sync tipset:%d", parent.Height())

		if !fs.synced_tipset_path_list.insert_tail_parent(parent) {
			return parent, fmt.Errorf("error, sync to genesis, isn't a parents(%d), it's impossible", parent.Height())
		}

		tpst_blms, err = fs.build_persistence_data(child, parent)
		if err != nil {
			return parent, err
		}

		tbml.Tipset_block_messages = append(tbml.Tipset_block_messages, tpst_blms)

		// todo: just merge but haven't write to store, but,
		//  the chain-notify may cause a writing..
		merged_path := fs.synced_tipset_path_list.try_merge_front_with_next(true)

		cids_size = len(parent.Parents().Cids())
		if len(tbml.Tipset_block_messages) >= max_size || merged_path != nil || cids_size == 0 {

			fs.Printf("‹syncinfo›:models_upsert(cached_tipset_size=%d)", len(tbml.Tipset_block_messages))
			if merged_path != nil {
				fs.Printf("‹syncinfo›:synced range is merged:(%d, %d)\n", merged_path.Head.Height, merged_path.Tail.Height)
			}

			models_data := tbml.build_models_data()

			fs.tipset_miner_messages_notifer <- models_data.miners

			if err = models_data.models_upsert(); err != nil {
				fs.Printf("error, tipset_block_message_list upsert failed, message:%s\n", err.Error())
				return parent, err
			}

			if err := fs.synced_tipset_path_list.models_upsert_front(true); err != nil {
				fs.Printf("error, models_upsert_front failed, message:%s\n", err.Error())
				return parent, err
			}

			if merged_path != nil {
				if err := merged_path.models_del(); err != nil {
					fs.Printf("error, merged_path.models_del failed, message:%s\n", err.Error())
					return parent, err
				}

				if parent, err = fs.api_tipset(fs.synced_tipset_path_list.front_tail().Key); err != nil {
					fs.Printf("api_tipset failed, message:%s\n", err.Error())
					return nil, err
				} else {
					cids_size = len(parent.Parents().Cids())
				}
			}

			tbml.Tipset_block_messages = tbml.Tipset_block_messages[:0]
		}

		fs.Printf("‹syncinfo›:sync tipset(%d) finished\n", parent.Height())
		end_height := parent.Height()
		if begin_height-end_height > 250 {
			end_time := time.Now().Unix()
			fs.Printf("‹syncinfo›:sync from:%d to:[%d], count=%d, used time=%dm:%ds\n",
				begin_height, end_height, begin_height-end_height,
				(end_time-begin_time)/60, (end_time-begin_time)%60)
			begin_time = end_time
			begin_height = end_height
		}
		child = parent
	}

	// send a nil, ask fs.loop_handle_messages to call fs.do_upsert_miners()
	fs.tipset_miner_messages_notifer <- nil

	return child, nil
}

func (fs *Filscaner) sync_tipset_cache_fall_through(child, parent *types.TipSet) (*types.TipSet, error) {
	var err error
	var blockmessage *Tipset_block_messages
	for !fs.tipsets_cache.full() {
		if blockmessage, err = fs.build_persistence_data(child, parent); err != nil {
			return nil, err
		}
		blockmessage.build_models_data()

		fs.tipsets_cache.push_back(blockmessage)

		child = parent
		if parent, err = fs.api.ChainGetTipSet(fs.ctx, child.Parents()); err != nil {
			return nil, err
		}
	}
	return parent, nil
}

func (fs *Filscaner) sync_tipset_with_range(last_ *types.TipSet, head_height, foot_height uint64) (*types.TipSet, error) {
	fs.Printf("do_sync_lotus, from:%d, to:%d\n", head_height, foot_height)

	head_tipset, err := fs.api.ChainGetTipSetByHeight(fs.ctx, head_height, last_)
	foot_tipset, err := fs.api.ChainGetTipSetByHeight(fs.ctx, foot_height, last_)
	tipset_list, err := fs.api.ChainGetPath(fs.ctx, foot_tipset.Key(), head_tipset.Key())
	if err != nil {
		fs.Printf("error:api.chain_get_path failed, message:%s\n", err.Error())
		return nil, err
	}

	if last_ == nil {
		if last_, err = fs.api_child_tipset(tipset_list[0].Val); err != nil {
			fs.Printf("error:api_child_tipset failed, message:%s\n", err.Error())
			return nil, err
		}
	}

	tbml := &Tipset_block_message_list{}
	for _, t := range tipset_list {
		tipset := t.Val
		tbm, err := fs.build_persistence_data(last_, tipset)
		if err != nil {
			return nil, err
		}

		tbml.Tipset_block_messages = append(tbml.Tipset_block_messages, tbm)

		last_ = tipset
	}

	return last_, tbml.build_models_data().models_upsert()
}

func (fs *Filscaner) build_persistence_data(child, parent *types.TipSet) (*Tipset_block_messages, error) {
	if child.Parents().String() != parent.Key().String() {
		return nil, fmt.Errorf("child(%d, %s).parentkey(%s)!=tipset(%d).key(%s)",
			child.Height(), child.Key().String(), child.Parents().String(),
			parent.Height(), parent.Key().String())
	}

	child_keys := child.Key().Cids()
	if len(child_keys) == 0 {
		return nil, fmt.Errorf("tipset(%d, %s) have no blocks????",
			child.Height(), child.Key().String())
	}

	return fs.api_tipset_blockmessages_and_receipts(parent, child_keys[0])
}

func (fs *Filscaner) handle_first_appl_tipset(child, parent *types.TipSet) {
	if child == nil {
		return
	}
	var err error
	if parent == nil || child.Parents().String() != parent.Key().String() {
		if parent, err = fs.api.ChainGetTipSet(fs.ctx, child.Parents()); err != nil {
			fs.Printf("error, get tipset(%d,%s) failed, message:%s\n",
				parent.Height()-1, parent.Parents().String(), err.Error())
			return
		}
	}
	fs.sync_tipset_cache_fall_through(child, parent)
	fs.handle_appl_tipset = fs.handle_secod_appl_tipset
}

func (fs *Filscaner) handle_secod_appl_tipset(child, this_is_nil_value_do_not_use *types.TipSet) {
	if fs.last_appl_tipset.Height() == child.Height() {
		// p1, _ := fs.api.ChainGetTipSet(fs.ctx, child.Parents())
		// p2, _ := fs.api.ChainGetTipSet(fs.ctx, fs.last_appl_tipset.Parents())
		// fs.Printf("p1 equals p2 = %v\n", p1.Equals(p2))
		return
	}

	parent, err := fs.api.ChainGetTipSet(fs.ctx, child.Parents())
	if err != nil {
		fs.Printf("error, get child(%d,%s) failed, message:%s\n",
			child.Height()-1, child.Parents().String(), err.Error())
		return
	}

	blockmessage, err := fs.build_persistence_data(child, parent)
	if err != nil {
		fs.Printf("error, build_persistence_data(child:%d, parent:%d) failed, message:%s\n",
			child.Height(), parent.Height(), err.Error())
		return
	}

	if ftsp := fs.tipsets_cache.Front(); ftsp != nil && child.Height() <= ftsp.Height() {
		fs.Printf("‹forked›‹›‹›at child:‹%d›, current head is:‹%d›\n",
			child.Height(), ftsp.Height())
	}

	if blockmessage = fs.tipsets_cache.push_front(blockmessage); blockmessage != nil {
		fs.handle_safe_tipset(blockmessage)
	}
}

func (fs *Filscaner) handle_first_safe_tipset(blockmessage *Tipset_block_messages) {
	if err := blockmessage.models_upsert(); err != nil {
		fs.Printf("error, Tipset_block_messages.models_upsert failed, message:%s\n",
			err.Error())
		return
	}
	fs.task_sync_to_genesis(blockmessage.Tipset)
	fs.handle_safe_tipset = fs.handle_secod_safe_tipset
}

func (fs *Filscaner) handle_secod_safe_tipset(in *Tipset_block_messages) {
	if in == nil {
		return
	}
	var tbml = &Tipset_block_message_list{
		Tipset_block_messages: []*Tipset_block_messages{in}}

	var err error

	if fs.synced_tipset_path_list.insert_head_child(in.Tipset) {
		if err = fs.synced_tipset_path_list.models_upsert_front(true); err != nil {
			fs.Printf("error, models_upsert_front failed, message:%s\n", err.Error())
			return
		}

		models_data := tbml.build_models_data()

		fs.tipset_miner_messages_notifer <- models_data.miners

		if err = models_data.models_upsert(); err != nil {
			fs.Printf("error, Tipset_block_message_list.upsert failed, message:%s\n",
				err.Error())
			return
		}
	}
}

func (fs *Filscaner) api_tipset_blockmessages_and_receipts(tipset *types.TipSet, child_cid cid.Cid) (*Tipset_block_messages, error) {
	var tpst_blms = &Tipset_block_messages{}
	var err error

	var blocks = tipset.Blocks()
	for _, block := range blocks {
		if message, err := fs.api.ChainGetBlockMessages(fs.ctx, block.Cid()); err == nil {
			blmsg := &Block_message{block, message}
			tpst_blms.BlockMsgs = append(tpst_blms.BlockMsgs, blmsg)

		} else {
			return nil, err
		}
	}

	tpst_blms.Tipset = tipset
	tpst_blms.BlockRwds = fs.API_block_rewards(tipset)
	// todo: for each block, it's block rewards shold multi it's epostprof.candidates.length
	//   that's real 'block rewards' geting by miner
	/* for _, blm := range tpst_blms.BlockMsgs {
			len(blm.Block.EPostProof.Candidates)
		}*/

	tpst_blms.Messages, err = fs.api.ChainGetParentMessages(fs.ctx, child_cid)
	if err != nil {
		return nil, err
	}
	tpst_blms.Receipts, err = fs.api.ChainGetParentReceipts(fs.ctx, child_cid)
	if err != nil {
		return nil, err
	}

	tpst_blms.build_models_data()

	return tpst_blms, nil

}
