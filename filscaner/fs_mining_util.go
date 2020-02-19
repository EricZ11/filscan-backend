package filscaner

import (
	errs "filscan_lotus/error"
	"filscan_lotus/filscaner/force/factors"
	"filscan_lotus/models"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/types"
)

type MinerStateRecord struct {
	Id     string                     `bson:"_id" json:"id"`
	Record *models.MinerStateAtTipset `bson:"record" json:"record"`
}

type MinerStateRecordInterface struct {
	Id     string      `bson:"_id" json:"id"`
	Record interface{} `bson:"record" json:"record"`
}

type MinerIncreasedPowerRecord struct {
	IncreasedPower uint64                     `bson:"increased_power" json:"increased_power"`
	Record         *models.MinerStateAtTipset `bson:"record" json:"record"`
}

type MinerBlockRecord struct {
	Blockcount uint64                     `bson:"block_count" json:"block_count"`
	Record     *models.MinerStateAtTipset `bson:"record" json:"record"`
}

type MinedBlock struct {
	Miner      string `bson:"miner" json:"miner"`
	BlockCount uint64 `bson:"mined_block_count" json:"mined_block_count"`
}

func ParseActorMessage(message *types.Message) (*factors.ActorInfo, *factors.MethodInfo, error) {

	if message.Method == 0 {
		return nil, nil, errs.ErrActorNotFound
	}
	actor, exist := factors.LookupByAddress(message.To)
	if !exist {
		if actor, exist = factors.Lookup(actors.StorageMinerCodeCid); !exist {
			return nil, nil, errs.ErrActorNotFound
		}
	}

	method, exist := actor.LookupMethod(message.Method)
	if !exist {
		return nil, nil, errs.ErrMethodNotFound
	}

	return &actor, &method, nil
}
