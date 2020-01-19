package models

type TopPowerMiner struct {
	MinerAddr     string `bson:"miner_addr" json:"miner_addr"`
	Power         uint64 `bson:"power" json:"power"`
	End           int64  `bson:"end" json:"end"`
	FilecoinPower uint64 `bson:"filecoin_power" json:"filecoin_power"`
	GmtCreate     int64  `bson:"gmt_create" json:"gmt_create"`
	GmtModified   int64  `bson:"gmt_modified" json:"gmt_modified"`
}
