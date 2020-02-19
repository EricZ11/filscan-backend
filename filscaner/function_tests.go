package filscaner

import (
	"fmt"
	"github.com/filecoin-project/lotus/chain/types"
)

func (fs *Filscaner) ChainHeadTest() {
	if tipset, err := fs.api.ChainHead(fs.ctx); err!=nil {
		panic(fmt.Sprintf("chian head failed, message:%s\n", err.Error()))
	} else {
		fs.DisplayTipset(tipset)
	}
}

func (fs *Filscaner) DisplayTipset(tipset *types.TipSet) {
	fs.Printf("tipset heigth=%d, tipset.block_size=%d\n",
		tipset.Height(),
		len(tipset.Blocks()))
}

func (fs *Filscaner) ChainTipsetByHeightTest() {
	var tipset *types.TipSet

	tipset, err := fs.api.ChainGetTipSetByHeight(fs.ctx, 100, nil)
	if err != nil {
		fs.Printf("get tipsetbyheight(10000) failed, message:%s", err.Error())
		return
	}
	fs.Printf("get tipsetbyheight, tipset.height:%d\n",
		tipset.Height())

	// if miners, err := fs.api.StateListMiners(fs.ctx, nil); err!=nil {
	// 	fs.Printf("error:%s\n", err.Error())
	// 	return
	// } else {
	// 	fs.Printf("total miner count = %d\n", len(miners))
	// 	for index, address := range miners {
	// 		fs.Printf("miner index:%d, miner address:%s\n", index, address.String())
	// 	}
	// }
	// fs.api.ChainGetBlockMessages()
	// fs.api.StateMinerSectors()
}

