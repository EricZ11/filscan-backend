package models

type TopBlockMiner struct {
	MinerAddr     string `bson:"miner_addr" json:"miner_addr"`
	BlockCount    uint64 `bson:"block_count" json:"block_count"`
	End           int64  `bson:"end" json:"end"`
	FilecoinBlock uint64 `bson:"filecoin_block" json:"filecoin_block"`
	GmtCreate     int64  `bson:"gmt_create" json:"gmt_create"`
	GmtModified   int64  `bson:"gmt_modified" json:"gmt_modified"`
}
