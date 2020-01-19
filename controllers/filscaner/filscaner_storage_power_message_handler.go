package filscaner

// import (
// 	"github.com/filecoin-project/lotus/api"
// 	"github.com/filecoin-project/lotus/chain/actors"
// 	"github.com/filecoin-project/go-address"
// 	"github.com/filecoin-project/lotus/chain/store"
// 	"github.com/filecoin-project/lotus/chain/types"
// 	core "github.com/libp2p/go-libp2p-core"
// 	"github.com/libp2p/go-libp2p-core/peer"
// 	"math/big"
// 	"time"
//
// 	"filscan_lotus/models"
// 	"github.com/filecoin-project/lotus/chain/vm"
// )
//
// // //1: spa.StoragePowerConstructor,
// // 2: spa.CreateStorageMiner,
// // 3: spa.ArbitrateConsensusFault,
// // 4: spa.UpdateStorage,
// // 5: spa.GetTotalStorage,
// // 6: spa.PowerLookup,
// // 7: spa.IsValidMiner,
// // 8: spa.PledgeCollateralForSize,
// // 9: spa.CheckProofSubmissions,
// func (fs *Filscaner) handle_storage_power_message(hctype string, tipset *types.TipSet, method *MethodCall) {
// 	if method.actor_name != "StoragePowerActor" { return }
//
// 	param := method.MethodInfo.NewParam()
// 	if err := vm.DecodeParams(method.Params, param); err != nil {
// 		fs.Printf("decode actor_name(%s), method(%s) param failed, message:%s\n",
// 			method.actor_name, method.Name, err.Error())
// 		return
// 	}
//
// 	isok := param.(actors.PledgeCollateralParams)
//
// }
