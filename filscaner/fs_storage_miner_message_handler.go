package filscaner

import (
	"filscan_lotus/models"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
)

func (fs *Filscaner) handle_storage_miner_message(tipset *types.TipSet, miner string) {
	address, err := address.NewFromString(miner)
	if err != nil {
		fs.Printf("handle miner(%s) message failed, message:%s\n", miner)
	}

	miner_state, err := fs.api_miner_state_at_tipset(address, tipset)
	if err != nil {
		fs.Printf("api_get_miner_state(%s) at tipset(%d) message failed, message:%s\n",
			miner, tipset.Height(), err.Error())
		return
	}

	if miner_state == nil {
		return
	}

	fs.models_update_miner(miner_state)
	fs.miner_state_chan <- miner_state
}

func (fs *Filscaner) handle_miner_state(miner *models.MinerStateAtTipset) {
	fs.miner_cache24h.update(miner)
	fs.miner_cache1day.update(miner)
}
