package filscaner

import (
	"container/list"
	"filscan_lotus/models"
	"filscan_lotus/utils"
	"fmt"
	"github.com/filecoin-project/lotus/chain/types"
	"sync"
)

// use *Tipset_block_messages as list.element.Value
type Fs_tipset_cache struct {
	list     *list.List
	max_size int
	mutx     sync.Mutex
}

func new_fs_cache(max_size int) *Fs_tipset_cache {
	fsc := &Fs_tipset_cache{
		list:     list.New(),
		max_size: max_size}
	return fsc
}

func (fsc *Fs_tipset_cache) lock() {
	fsc.mutx.Lock()
}

func (fsc *Fs_tipset_cache) unlock() {
	fsc.mutx.Unlock()
}

func (fsc *Fs_tipset_cache) full() bool {
	fsc.lock()
	defer fsc.unlock()
	return fsc.list.Len() >= fsc.max_size
}

func (fsc *Fs_tipset_cache) Size() int {
	fsc.lock()
	defer fsc.unlock()
	return fsc.list.Len()
}

func (fsc *Fs_tipset_cache) push_back(in *Tipset_block_messages) {
	if in == nil {
		return
	}
	fsc.lock()
	defer fsc.unlock()

	if fsc.list.Len() > 0 {
		if in.equals(fsc.list.Back().Value.(*Tipset_block_messages)) {
			return
		}
	}

	fsc.list.PushBack(in)
}

func (fsc *Fs_tipset_cache) push_front(in *Tipset_block_messages) (blockmessage *Tipset_block_messages) {
	if in == nil || in.Tipset == nil {
		return
	}
	fsc.lock()
	defer fsc.unlock()

	if fsc.list.Len() > 0 {
		if fsc.list.Front().Value.(*Tipset_block_messages).equals(in) {
			return
		}
	}

	in_tipset := in.Tipset
	if true {
		for front := fsc.list.Front(); front != nil; front = fsc.list.Front() {
			if f := front.Value.(*Tipset_block_messages).Tipset; f != nil {
				if f.Key().String() == in_tipset.Parents().String() {
					break
				}
				fmt.Printf("this is a forked situation, income in:%d, removed in:%d\n",
					in_tipset.Height(), f.Height())
				fsc.list.Remove(front)
			}
		}
	} else {
		// check if in.Height() < fsc.list.front.height(),
		//   and sovle this chain 'forked' situation
		for fsc.list.Len() > 0 && in_tipset.Height() <= fsc.list.Front().Value.(*types.TipSet).Height() {
			fsc.list.Remove(fsc.list.Front())
		}
	}

	fsc.list.PushFront(in)

	if fsc.list.Len() > fsc.max_size {
		blockmessage = fsc.list.Remove(fsc.list.Back()).(*Tipset_block_messages)
		// child = fsc.list.Back().Value.(*Tipset_block_messages)
	}
	return
}

func (fsc *Fs_tipset_cache) Front() *types.TipSet {
	fsc.lock()
	defer fsc.unlock()
	if fsc.list.Len() > 0 {
		return fsc.list.Front().Value.(*Tipset_block_messages).Tipset
	}
	return nil
}

type Is_match_func func(*Tipset_block_messages) bool

func (fsc *Fs_tipset_cache) Loop(is_match Is_match_func) interface{} {
	fsc.lock()
	defer fsc.unlock()

	for f := fsc.list.Front(); f != nil; f = f.Next() {
		if is_match(f.Value.(*Tipset_block_messages)) {
			// TODO:!!!!!
		}
	}

	return nil
}

func (fsc *Fs_tipset_cache) FindBlock_ofst_count(ofst, count int) []*models.BlockAndMsg {
	fsc.lock()
	defer fsc.unlock()

	list_size := fsc.list.Len()
	if ofst+count > list_size {
		count = list_size - ofst
	}

	var blockmsg_arr = []*models.BlockAndMsg{}

	front := fsc.list.Front()
	blockindex := 0
exit:
	for front != nil {
		tipset_block_message := front.Value.(*Tipset_block_messages)
		front = front.Next()

		for _, blockmsg := range tipset_block_message.fs_block_message {
			if blockindex < ofst {
				blockindex++
				continue
			}

			blockmsg_arr = append(blockmsg_arr, blockmsg)

			if blockindex-ofst >= count {
				break exit
			}
			blockindex++
		}
	}
	return blockmsg_arr
}

func (fsc *Fs_tipset_cache) FindMesage_ofset_count(ofst, count int) []*models.FilscanMsg {
	fsc.lock()
	defer fsc.unlock()

	list_size := fsc.list.Len()
	if ofst+count > list_size {
		count = list_size - ofst
	}

	var msg_arr = []*models.FilscanMsg{}

	front := fsc.list.Front()
	blockindex := 0
exit:
	for front != nil {
		tipset_block_message := front.Value.(*Tipset_block_messages)
		front = front.Next()

		for _, msg := range tipset_block_message.fs_msgs {
			if blockindex < ofst {
				blockindex++
				continue
			}

			msg_arr = append(msg_arr, msg)

			if blockindex-ofst >= count {
				break exit
			}
			blockindex++
		}
	}
	return msg_arr
}

func (fsc *Fs_tipset_cache) FindTipset_height(height uint64) *models.Element {
	fsc.lock()
	defer fsc.unlock()

	for back := fsc.list.Back(); back != nil; back = back.Prev() {
		block_message := back.Value.(*Tipset_block_messages)

		if block_message.Tipset.Height() == height {
			return &models.Element{block_message.Tipset, block_message.fs_block_message}
		}
	}

	return nil
}

func (fsc *Fs_tipset_cache) FindTipset_in_height(start, end uint64) []*models.Element {
	fsc.lock()
	defer fsc.unlock()

	var ele_arr = []*models.Element{}

	for back := fsc.list.Back(); back != nil; back = back.Prev() {
		block_message := back.Value.(*Tipset_block_messages)

		if block_message.Tipset.Height() > end {
			break
		}

		if block_message.Tipset.Height() < start {
			continue
		}

		ele_arr = append(ele_arr, &models.Element{block_message.Tipset, block_message.fs_block_message})
	}

	return ele_arr
}

func (fsc *Fs_tipset_cache) FindMesage_block(block_id string) []*models.FilscanMsg {
	fsc.lock()
	defer fsc.unlock()

	for front := fsc.list.Front(); front != nil; front = front.Next() {
		tipset_blockmessage := front.Value.(*Tipset_block_messages)
		for _, blockmessage := range tipset_blockmessage.fs_block_message {
			if blockmessage.Block.Cid == block_id {
				return blockmessage.Msg
			}
		}
	}

	return []*models.FilscanMsg{}
}

func (fsc *Fs_tipset_cache) FindMesage_method(method string) []*models.FilscanMsg {
	fs_msg_arr := []*models.FilscanMsg{}
	if method == "" {
		return fs_msg_arr
	}

	fsc.lock()
	defer fsc.unlock()

	for front := fsc.list.Front(); front != nil; front = front.Next() {
		tipset_blockmessage := front.Value.(*Tipset_block_messages)

		for _, msg := range tipset_blockmessage.fs_msgs {
			if msg.MethodName == method {
				fs_msg_arr = append(fs_msg_arr, msg)
			}
		}
	}

	return fs_msg_arr
}

func (fsc *Fs_tipset_cache) FindMesage_id(cid string) *models.FilscanMsg {
	fsc.lock()
	defer fsc.unlock()

	for front := fsc.list.Front(); front != nil; front = front.Next() {
		tipset_blockmessage := front.Value.(*Tipset_block_messages)

		for _, msg := range tipset_blockmessage.fs_msgs {
			if msg.Cid == cid {
				return msg
			}
		}
	}
	return nil
}

func (fsc *Fs_tipset_cache) FindMesage_blocks(blocks []string) []*models.FilscanMsg {
	mblocks := make(map[string]struct{})
	for _, id := range blocks {
		mblocks[id] = struct{}{}
	}

	fsc.lock()
	defer fsc.unlock()

	var msg_arr = []*models.FilscanMsg{}
	for front := fsc.list.Front(); front != nil; front = front.Next() {
		tipset_blockmessage := front.Value.(*Tipset_block_messages)
		for _, block := range tipset_blockmessage.fs_block_message {
			if _, exist := mblocks[block.Block.Cid]; exist {
				msg_arr = append(msg_arr, block.Msg[:]...)
			}
		}
	}
	return msg_arr
}

func (fsc *Fs_tipset_cache) MesageAll() []*models.FilscanMsg {
	fsc.lock()
	defer fsc.unlock()

	fs_msg_arr := []*models.FilscanMsg{}
	for front := fsc.list.Front(); front != nil; front = front.Next() {
		tipset_blockmessage := front.Value.(*Tipset_block_messages)
		fs_msg_arr = append(fs_msg_arr, tipset_blockmessage.fs_msgs[:]...)
	}
	return fs_msg_arr
}

func (fsc *Fs_tipset_cache) FindMesage_block_method(block_cid, method string) []*models.FilscanMsg {
	if block_cid != "" {
		blockmsg := fsc.FindBlock_id(block_cid)
		if blockmsg == nil {
			return nil
		}
		if method == "" {
			return blockmsg.Msg
		} else {
			var flmsgs []*models.FilscanMsg
			for _, msg := range blockmsg.Msg {
				if msg.MethodName == method {
					flmsgs = append(flmsgs, msg)
				}
			}
			return flmsgs
		}
	} else if method != "" {
		return fsc.FindMesage_method(method)
	} else {
		return fsc.MesageAll()
	}
}

func (fsc *Fs_tipset_cache) FindBlock_id(id string) *models.BlockAndMsg {
	fsc.lock()
	defer fsc.unlock()

	for front := fsc.list.Front(); front != nil; front = front.Next() {
		block_msg_arr := front.Value.(*Tipset_block_messages).fs_block_message
		for _, block_msg := range block_msg_arr {
			if block_msg.Block.Cid == id {
				return block_msg
			}
		}
	}
	return nil
}

func (fsc *Fs_tipset_cache) FindBlock_miners(miners []string) []*models.FilscanBlock {
	fsc.lock()
	defer fsc.unlock()

	blocks := []*models.FilscanBlock{}

	mminers := utils.SlcToMap(miners, "", false).(map[string]struct{})

	for front := fsc.list.Front(); front != nil; front = front.Next() {
		block_msg_arr := front.Value.(*Tipset_block_messages).fs_block_message
		for _, block_msg := range block_msg_arr {
			if _, exist := mminers[block_msg.Block.BlockHeader.Miner.String()]; exist {
				blocks = append(blocks, block_msg.Block)
			}
		}
	}

	return blocks
}

func (fsc *Fs_tipset_cache) FindMesage_address(address, fromto, method string) []*models.FilscanMsg {
	fsc.lock()
	defer fsc.unlock()

	msg_arr := []*models.FilscanMsg{}
	for front := fsc.list.Front(); front != nil; front = front.Next() {
		tipset_block_message := front.Value.(*Tipset_block_messages)
		for _, msg := range tipset_block_message.fs_msgs {
			if fromto == "from" && msg.Message.From.String() != address {
				break
			} else if fromto == "to" && msg.Message.To.String() != address {
				break
			} else if msg.Message.From.String()!=address && msg.Message.To.String()!=address {
				break
			}
			if method != "" && msg.MethodName != method {
				break
			}
			msg_arr = append(msg_arr, msg)
		}
	}
	return msg_arr
}

func (fsc *Fs_tipset_cache) Blocks() []*models.BlockAndMsg {
	fsc.lock()
	defer fsc.unlock()

	var block_messages []*models.BlockAndMsg
	for front := fsc.list.Front(); front != nil; front = front.Next() {
		block_messages = append(block_messages, front.Value.(*Tipset_block_messages).fs_block_message[:]...)
	}
	return block_messages
}

func (fsc *Fs_tipset_cache) TipsetCountInTime(start, end int64) int64 {
	fsc.lock()
	defer fsc.unlock()

	count := int64(0)
	for front := fsc.list.Front(); front != nil; front = front.Next() {
		at := int64(front.Value.(*Tipset_block_messages).Tipset.MinTimestamp())
		if at >= start && at < end {
			count++
		}
	}
	return count
}

func (fsc *Fs_tipset_cache) LatestBlockrewards() string {
	fsc.lock()
	defer fsc.unlock()

	if front := fsc.list.Front(); front != nil {
		return utils.ToFilStr(front.Value.(*Tipset_block_messages).BlockRwds)
	}
	return ""
}
