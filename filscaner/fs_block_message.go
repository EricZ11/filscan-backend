package filscaner

import (
	"filscan_lotus/models"
	"filscan_lotus/utils"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/globalsign/mgo"
	"gopkg.in/mgo.v2/bson"
	"math/big"
	"strconv"
	"time"
)

type Block_message struct {
	Block   *types.BlockHeader
	BlkMsgs *api.BlockMessages
}

type Tipset_block_messages struct {
	Tipset    *types.TipSet
	BlockRwds *big.Int
	Messages  []api.Message
	Receipts  []*types.MessageReceipt
	BlockMsgs []*Block_message

	is_cached bool

	fs_tipset *models.FilscanTipSet
	fs_blocks []*models.FilscanBlock
	fs_msgs   []*models.FilscanMsg
	fs_miners *Tipset_miner_messages

	fs_block_message []*models.BlockAndMsg
}

type Tipset_block_message_list struct {
	Tipset_block_messages []*Tipset_block_messages
}

func (self *Block_message) fs_block() *models.FilscanBlock {
	if self.Block == nil {
		return nil
	}
	now := time.Now().Unix()
	block_data, _ := self.Block.Serialize()
	fs_block := &models.FilscanBlock{
		Cid:         self.Block.Cid().String(),
		BlockHeader: self.Block,
		MsgCids:     self.BlkMsgs.Cids,
		GmtCreate:   now,
		GmtModified: now,
		Size:        int64(len(block_data)),
	}
	return fs_block
}

func (self *Block_message) fs_messages() []*models.FilscanMsg {
	if self.BlkMsgs == nil {
		return nil
	}
	var fs_msg_list []*models.FilscanMsg

	now := time.Now().Unix()
	for _, v := range self.BlkMsgs.BlsMessages {
		data, _ := v.Serialize()
		fs_msg := &models.FilscanMsg{
			Message:       *v,
			Cid:           v.Cid().String(),
			BlockCid:      self.Block.Cid().String(),
			RequiredFunds: v.RequiredFunds(),
			Size:          int64(len(data)),
			Height:        self.Block.Height,
			MsgCreate:     self.Block.Timestamp,
			GmtCreate:     now,
			GmtModified:   now}

		if v.Method == 0 {
			fs_msg.MethodName = "Transfer"
		} else {
			if actor, method, err := ParseActorMessage(v); err == nil {
				fs_msg.ActorName = actor.Name
				fs_msg.MethodName = method.Name
			}
		}
		fs_msg_list = append(fs_msg_list, fs_msg)
	}
	for _, secp := range self.BlkMsgs.SecpkMessages {
		data, _ := secp.Message.Serialize()
		fs_msg := &models.FilscanMsg{
			Message:       secp.Message,
			Cid:           secp.Cid().String(),
			BlockCid:      self.Block.Cid().String(),
			RequiredFunds: secp.Message.RequiredFunds(),
			Size:          int64(len(data)),
			Height:        self.Block.Height,
			MsgCreate:     self.Block.Timestamp,
			GmtCreate:     now,
			GmtModified:   now,
			Signature:     secp.Signature}

		if secp.Message.Method == 0 {
			fs_msg.MethodName = "Transfer"
		} else {
			if actor, method, err := ParseActorMessage(&secp.Message); err == nil {
				fs_msg.ActorName = actor.Name
				fs_msg.MethodName = method.Name
			}
		}
		fs_msg_list = append(fs_msg_list, fs_msg)
	}
	return fs_msg_list
}

func (self *Tipset_block_messages) equals(tps_blms *Tipset_block_messages) bool {
	if self == tps_blms {
		return true
	}
	if self == nil || tps_blms == nil {
		return false
	}
	return self.Tipset.Equals(tps_blms.Tipset)
}

func (self *Tipset_block_messages) receipts_ref() map[string]*types.MessageReceipt {
	receipt_ref := make(map[string]*types.MessageReceipt)
	for index, receipt := range self.Receipts {
		receipt_ref[self.Messages[index].Cid.String()] = receipt
	}
	return receipt_ref
}

func (self *Tipset_block_messages) models_upsert() error {
	return (&Tipset_block_message_list{[]*Tipset_block_messages{self}}).build_models_data().models_upsert()
}

func (self *Tipset_block_messages) build_models_data() (*models.FilscanTipSet, []*models.FilscanBlock, []*models.FilscanMsg, *Tipset_miner_messages) {
	if self.is_cached {
		return self.fs_tipset, self.fs_blocks, self.fs_msgs, self.fs_miners
	}

	var model_tipset *models.FilscanTipSet
	var model_blocks []*models.FilscanBlock
	var model_msgs []*models.FilscanMsg

	var model_block_msgs []*models.BlockAndMsg

	var receipt_ref = self.receipts_ref()
	var miner_messages = &Tipset_miner_messages{
		tipset: self.Tipset,
		miners: make(map[string]struct{})}

	model_tipset = to_fs_tipset(self.Tipset)
	for _, blmsg := range self.BlockMsgs {
		fsblock := blmsg.fs_block()
		fsmesages := blmsg.fs_messages()

		fsblock_message := &models.BlockAndMsg{
			Block: fsblock}

		fsblock.BlockReward = utils.ToFilStr(self.BlockRwds)

		for _, fsmsg := range fsmesages {
			// 设置message相关的receipt
			if receipt, exist := receipt_ref[fsmsg.Cid]; exist && receipt != nil {
				fsmsg.ExitCode = strconv.Itoa(int(receipt.ExitCode))
				fsmsg.GasUsed = receipt.GasUsed.String()
				fsmsg.Return = string(receipt.Return)
			}
			model_msgs = append(model_msgs, fsmsg)

			fsblock_message.Msg = append(fsblock_message.Msg, fsmsg)

			if fsmsg.ActorName == "StorageMinerActor" {
				miner_messages.miners[fsmsg.Message.To.String()] = struct{}{}
			}
		}

		model_blocks = append(model_blocks, fsblock)
		model_block_msgs = append(model_block_msgs, fsblock_message)
	}

	self.fs_tipset = model_tipset
	self.fs_blocks = model_blocks
	self.fs_msgs = model_msgs
	self.fs_miners = miner_messages
	self.fs_block_message = model_block_msgs

	self.is_cached = true

	return model_tipset, model_blocks, model_msgs, miner_messages
}

type fs_models_data struct {
	tipsets  []*models.FilscanTipSet
	blocks   []*models.FilscanBlock
	messages []*models.FilscanMsg
	miners   []*Tipset_miner_messages
}

func (self *fs_models_data) models_upsert() error {
	return models_bulk_upsert_block_message_tipset(self.messages, self.blocks, self.tipsets)
}

func (self *Tipset_block_message_list) build_models_data() *fs_models_data {
	models_data := &fs_models_data{
		miners: make([]*Tipset_miner_messages, len(self.Tipset_block_messages)),
	}

	for index, tpst_blms := range self.Tipset_block_messages {

		tmp_fs_tipset, tmp_fs_blocks, tmp_fs_msgs, tmp_miner_msgs := tpst_blms.build_models_data()

		models_data.tipsets = append(models_data.tipsets, tmp_fs_tipset)
		models_data.blocks = append(models_data.blocks, tmp_fs_blocks[:]...)
		models_data.messages = append(models_data.messages, tmp_fs_msgs[:]...)
		models_data.miners[index] = tmp_miner_msgs
	}

	return models_data
}

// func (self *Tipset_block_message_list) models_upsert() error {
// 	var fs_blocks []*models.FilscanBlock
// 	var fs_messages []*models.FilscanMsg
// 	var fs_tipsets []*models.FilscanTipSet
// 	var miner_msg_arr = make([]*Tipset_miner_messages, len(self.Tipset_block_messages))
//
// 	for index, tipset_bm := range self.Tipset_block_messages {
// 		fs_tipsets = append(fs_tipsets, to_fs_tipset(tipset_bm.Tipset))
//
// 		tmp_fs_blocks, tmp_fs_msgs, tmp_miner_msgs := tipset_bm.build_models_data()
//
// 		fs_blocks = append(fs_blocks, tmp_fs_blocks[:]...)
// 		fs_messages = append(fs_messages, tmp_fs_msgs[:]...)
//
// 		miner_msg_arr[index] = tmp_miner_msgs
// 	}
//
// 	return models_bulk_upsert_block_message_tipset(fs_messages, fs_blocks, fs_tipsets)
// }

func models_bulk_upsert_tispet(col *mgo.Collection, tipset_list []*models.FilscanTipSet) error {
	size := len(tipset_list)
	if size == 0 {
		return nil
	}

	bulk_items := make([]interface{}, size*2)

	for index, tipset := range tipset_list {
		i := index * 2
		bulk_items[i] = bson.M{"height": tipset.Height}
		bulk_items[i+1] = utils.ToInterface(tipset)
	}

	_, err := models.BulkUpsert(col, "tipset", bulk_items)
	return err
}

func models_bulk_upsert_message(col *mgo.Collection, msg_list []*models.FilscanMsg) error {
	size := len(msg_list)
	if size == 0 {
		return nil
	}

	const max_size = 256
	bulk_items := make([]interface{}, max_size*2)

	for size > 0 {
		real_size := size

		if size > max_size {
			real_size = max_size
		}

		var index = 0
		for ; index < real_size; index++ {
			i := index * 2
			bulk_items[i] = bson.M{"cid": msg_list[index].Cid}
			bulk_items[i+1] = utils.ToInterface(msg_list[index])
		}

		if _, err := models.BulkUpsert(col, models.MsgCollection, bulk_items[:index*2]); err != nil {
			return err
		}

		msg_list = msg_list[index:]
		size = len(msg_list)
	}

	return nil
}

func models_bulk_upsert_block(col *mgo.Collection, block_list []*models.FilscanBlock) error {
	size := len(block_list)
	if size == 0 {
		return nil
	}

	bulk_items := make([]interface{}, size*2)
	for index, block := range block_list {
		i := index * 2
		bulk_items[i] = bson.M{"cid": block.Cid}
		bulk_items[i+1] = utils.ToInterface(block)
	}
	_, err := models.BulkUpsert(col, models.BlocksCollection, bulk_items)
	return err
}

func models_bulk_upsert_block_message_tipset(fs_messages []*models.FilscanMsg, fs_blocks []*models.FilscanBlock, fs_tipsets []*models.FilscanTipSet) error {
	ms, db := models.Copy()
	defer ms.Close()

	if err := models_bulk_upsert_message(db.C(models.MsgCollection), fs_messages); err != nil {
		return err
	}
	if err := models_bulk_upsert_block(db.C(models.BlocksCollection), fs_blocks); err != nil {
		return err
	}
	if err := models_bulk_upsert_tispet(db.C("tipset"), fs_tipsets); err != nil {
		return err
	}
	return nil
}

func to_fs_tipset(tipset *types.TipSet) *models.FilscanTipSet {
	now := time.Now().Unix()
	fs_tipset := &models.FilscanTipSet{
		Key:          tipset.Key().String(),
		ParentKey:    tipset.Parents().String(),
		Cids:         tipset.Cids(),
		Height:       tipset.Height(),
		Mintime:      tipset.MinTimestamp(),
		Parents:      tipset.Parents().Cids(),
		GmtCreate:    now,
		GmtModified:  now,
		MinTicketCId: tipset.MinTicketBlock().Cid(),
	}
	return fs_tipset
}
