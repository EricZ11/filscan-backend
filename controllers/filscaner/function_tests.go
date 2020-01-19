package filscaner

import (
	"github.com/filecoin-project/lotus/chain/types"
)

func (fs *Filscaner) ChainHeadTest() {
	tipset, err := fs.api.ChainHead(fs.ctx)
	if err!=nil {
		fs.Printf("chian head failed, message:%s\n", err.Error())
		return
	}
	fs.DisplayTipset(tipset)
}

func (fs *Filscaner) DisplayTipset(tipset *types.TipSet) {
	fs.Printf("tipset heigth=%d, tipset.block_size=%d\n",
		tipset.Height(),
		len(tipset.Blocks()))
}

func (fs *Filscaner) ChainTipsetByHeightTest() {
	var tipset *types.TipSet

	tipset, err := fs.api.ChainGetTipSetByHeight(fs.ctx, 100, nil)
	if err!=nil {
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


func (fs *Filscaner) NotifierTest(n int) {
	fs.Task_StartHandleMessage()
	fs.Task_StartHandleHeadChange()
	if notifer, err := fs.api.ChainNotify(fs.ctx); err!=nil {
		fs.Printf("notify failed, message:%s\n", err.Error())
		return
	} else {
		for i:=0; i<n; i++ {
			fs.Printf("---------------crcle index = %d\n", i)
			select {
			case headers, isok := <-notifer:
				if !isok { continue }
				for _, header := range headers {
					fs.Printf("get new header = %d\n", header.Val.Height())
					fs.head_notifier <- header
				}
			// default:
				// time.Sleep(time.Second * 5)
				// fs.Printf("sleep 1 second\n")
			}
		}
	}
}
