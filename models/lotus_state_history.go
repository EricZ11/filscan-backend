package models

type LotusStateHistory struct {
	MinerCount       uint64  `bson:"miner_count" json:"miner_count"`
	Power            uint64  `bson:"power" json:"power"`
	BlockCount       uint64  `bson:"block_count" json:"block_count"`
	Floating         float64 `bson:"floating" json:"floating"`                   //流通的
	PledgeCollateral float64 `bson:"pledge_collateral" json:"pledge_collateral"` //质押中的
	Outstanding      float64 `bson:"outstanding" json:"outstanding"`             //全网可用的
	GmtCreate        int64   `bson:"gmt_create" json:"gmt_create"`
	GmtModified      int64   `bson:"gmt_modified" json:"gmt_modified"`
}
