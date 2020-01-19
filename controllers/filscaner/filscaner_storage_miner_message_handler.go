package filscaner

import (
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"math/big"

	"filscan_lotus/models"
	"github.com/filecoin-project/lotus/chain/vm"
)

func (fs *Filscaner) handle_storage_miner_message(hctype string, tipset *types.TipSet, miner string) {
	address, err := address.NewFromString(miner)
	if err!=nil {
		fs.Printf("handle miner(%s) message failed, message:%s\n", miner)
	}

	miner_state, err := fs.api_get_miner_state_at_tipset(address, tipset)
	if err != nil {
		fs.Printf("api_get_miner_state(%s) at tipset(%d) message failed, message:%s\n",
			miner, tipset.Height(), err.Error())
		return
	}

	if miner_state != nil {
		fs.models_update_miner(miner_state)
	}
}

func (fs *Filscaner) handle_miner_message_deprecated(hctype string, tipset *types.TipSet, method *MethodCall) {
	if method.actor_name != "StorageMinerActor" {
		return
	}

	hehe := method.MethodInfo.NewParam()
	if err := vm.DecodeParams(method.Params, hehe); err != nil {
		fs.Printf("decode actor_name(%s), method(%s) param failed, message:%s\n",
			method.actor_name, method.Name, err.Error())
		return
	}

	var (
		miner                      *models.MinerStateInTipset
		miners                     = []*models.MinerStateInTipset{}
		err                        error
		previousIncomingSectorSize = models.NewBigintFromInt64(0)
	)

	if miners, err = fs.get_minerstate_lte2(method.To, tipset.Height()); err == nil {
		length := len(miners)
		if length > 0 {
			if miners[0].TipsetHeight == tipset.Height() {
				miner = miners[0]
				if length > 1 {
					previousIncomingSectorSize = miners[1].ProvingSectorSize
				}
			} else {
				previousIncomingSectorSize = miners[0].ProvingSectorSize
			}
		}
	}

	if miner == nil || hctype == store.HCRevert || miner.TipsetHeight > (fs.header_height-50) {
		miner, err = fs.api_get_miner_state_at_tipset(method.To, tipset)
		if err != nil {
			fs.Printf("get miner at tipset failed, message:%s\n",
				method.To.String(), tipset.Height())
			return
		}
	}

	switch method.Name {
	case "PreCommitSector":
		{
			if _, isok := hehe.(*actors.SectorPreCommitInfo); !isok {
				return
			}
			miner.ProvingSectorSize.Add(
				big.NewInt(int64(miner.SectorSize)), previousIncomingSectorSize.Int)
		}
	case "ProveCommitSector":
		{
			if _, isok := hehe.(*actors.SectorProveCommitInfo); !isok {
				return
			}
			if previousIncomingSectorSize.Uint64() > miner.SectorSize {
				miner.ProvingSectorSize.Sub(previousIncomingSectorSize.Int,
					big.NewInt(int64(miner.SectorSize)))
			}
		}
	}

	// fs.Printf("miner:%s, updated a <sectorstoragemessage> : %s, current incomming-sectorsize = %s\n",
	// 	method.To.String(), method.Name, XSizeString(miner.ProvingSectorSize))
	fs.models_update_miner(miner)
}

func (fs *Filscaner) api_update_miner_state_at_tipset(miner *models.MinerStateInTipset, tipset *types.TipSet) error {
	miner.TipsetHeight = tipset.Height()
	miner.MineTime = tipset.MinTimestamp()

	miner_addr, err := address.NewFromString(miner.MinerAddr)
	if err != nil {
		return err
	}

	var (
		miner_power         api.MinerPower
		sectors             []*api.ChainSectorInfo
		proving_sector_size *big.Int
		peerid              peer.ID
	)

	if miner_power, err = fs.api.StateMinerPower(fs.ctx, miner_addr, tipset); err != nil {
		fs.Printf("get miner power failed, message:%s\n", err.Error())
		return err
	}

	if sectors, err = fs.api.StateMinerSectors(fs.ctx, miner_addr, tipset); err != nil {
		fs.Printf("get miner sector failed, message:%s\n", err.Error())
		return err
	}

	if peerid, err = fs.api.StateMinerPeerID(fs.ctx, miner_addr, tipset); err != nil {
		fs.Printf("get peerid failed, message:%s\n", err.Error())
		return err
	}

	if proving_sector, err := fs.api.StateMinerProvingSet(fs.ctx, miner_addr, tipset); err != nil {
		proving_sector_size = big.NewInt(0).Mul(big.NewInt(int64(miner.SectorSize)), big.NewInt(int64(len(proving_sector))))
	} else {
		proving_sector_size = big.NewInt(0)
	}

	miner.Power.Set(miner_power.MinerPower.Int)
	miner.TotalPower.Set(miner_power.TotalPower.Int)
	miner.ProvingSectorSize.Set(proving_sector_size)
	miner.SectorCount = uint64(len(sectors))
	miner.PeerId = peerid.String()
	return nil
}

func (fs *Filscaner) api_get_miner_state_at_tipset(miner_addr address.Address, tipset *types.TipSet) (*models.MinerStateInTipset, error) {
	var (
		peerid              core.PeerID
		owner               address.Address
		power               api.MinerPower
		sectors             []*api.ChainSectorInfo
		sector_size         uint64
		proving_sector_size = models.NewBigintFromInt64(0)
		err                 error
	)

	if power, err = fs.api.StateMinerPower(fs.ctx, miner_addr, tipset); err != nil {
		fs.Printf("get miner power failed, message:%s\n", err.Error())
		return nil, err
	}
	if sectors, err = fs.api.StateMinerSectors(fs.ctx, miner_addr, tipset); err != nil {
		fs.Printf("get miner sector failed, message:%s\n", err.Error())
		return nil, err
	}

	if peerid, err = fs.api.StateMinerPeerID(fs.ctx, miner_addr, tipset); err != nil {
		fs.Printf("get peerid failed, address=%s message:%s\n", miner_addr.String(), err.Error())
	}

	if owner, err = fs.api.StateMinerWorker(fs.ctx, miner_addr, tipset); err != nil {
		fs.Printf("get miner worker failed, message:%s\n", err.Error())
		return nil, err
	}

	if sector_size, err = fs.api.StateMinerSectorSize(fs.ctx, miner_addr, tipset); err != nil {
		fs.Printf("get miner sectorsize failed, message:%s\n", err.Error())
		return nil, err
	}

	if proving_sector, err := fs.api.StateMinerProvingSet(fs.ctx, miner_addr, tipset); err != nil {
		fs.Printf("state_miner_proving_set failed, message:%s\n", err.Error())
	} else {
		proving_sector_size.Set(big.NewInt(0).Mul(big.NewInt(int64(sector_size)), big.NewInt(int64(len(proving_sector)))))
	}

	// 这里应该是把错误的数据使用最近的数据来代替19807040628566131532430835712
	if len(power.TotalPower.String()) >= 29 {
		if fs.latest_total_power != nil {
			power.TotalPower.Set(fs.latest_total_power)
		} else {
			power.TotalPower.SetUint64(0)
		}
	} else {
		if fs.latest_total_power == nil {
			fs.latest_total_power = big.NewInt(0)
		}
		fs.latest_total_power.Set(power.TotalPower.Int)
	}

	miner := &models.MinerStateInTipset{
		PeerId:            peerid.String(),
		MinerAddr:         miner_addr.String(),
		Power:             models.NewBigInt(power.MinerPower.Int),
		TotalPower:        models.NewBigInt(power.TotalPower.Int),
		SectorSize:        sector_size,
		WalletAddr:        owner.String(),
		SectorCount:       uint64(len(sectors)),
		TipsetHeight:      tipset.Height(),
		ProvingSectorSize: proving_sector_size,
		MineTime:          tipset.MinTimestamp(),
	}

	miner.SectorCount = uint64(len(sectors))

	return miner, nil
}
