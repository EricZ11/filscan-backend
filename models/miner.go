package models

import (
	"encoding/json"
	fspt "filscan_lotus/filscanproto"
	"filscan_lotus/utils"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"math/big"
	"time"
)

const (
	MinerCollection       = "miner"
	BlockRewardCollection = "block_reward"
)

type BsonBigint struct{ *big.Int }

func NewBigintFromInt64(i int64) *BsonBigint {
	return &BsonBigint{
		Int: big.NewInt(i),
	}
}

func NewBigInt(i *big.Int) *BsonBigint {
	return &BsonBigint{
		Int: big.NewInt(0).SetBytes(i.Bytes()),
	}
}

func (mbig *BsonBigint) Set(i *big.Int) {
	if mbig.Int == nil {
		mbig.Int = big.NewInt(0)
	}
	if i != nil {
		mbig.Int.Set(i)
	}
}

func (mbig *BsonBigint) GetBSON() (interface{}, error) {
	if mbig.Int == nil {
		mbig.Int = big.NewInt(0)
	}
	return mbig.String(), nil
}

func (mbig *BsonBigint) SetBSON(raw bson.Raw) error {
	var num string
	if err := raw.Unmarshal(&num); err != nil {
		return err
	}

	if mbig.Int == nil {
		mbig.Int = big.NewInt(0)
	}

	if _, isok := mbig.SetString(num, 10); !isok {
		return fmt.Errorf("convert '%s' to bigint failed", num)
	}
	return nil
}

type MinerStateAtTipset struct {
	PeerId            string      `bson:"peer_id" json:"peer_id"`
	MinerAddr         string      `bson:"miner_addr" json:"miner_addr"`
	BlockCount        uint64      `bson:"block_count" json:"block_count"`
	Power             *BsonBigint `bson:"power" json:"power"`
	TotalPower        *BsonBigint `bson:"total_power" json:"total_power"`
	WalletAddr        string      `bson:"wallet_addr" json:"wallet_addr"`
	SectorSize        uint64      `bson:"sector_size" json:"sector_size"`
	SectorCount       uint64      `bson:"sector_count" json:"sector_count"`
	BlockCountPercent float64     `bson:"block_count_percent" json:"block_count_percent"`
	PowerPercent      float64     `bson:"power_percent" json:"power_percent"`
	ProvingSectorSize *BsonBigint `bson:"incoming_sectorsize" json:"incoming_sectorsize"`
	TipsetHeight      uint64      `bson:"tipset_height" json:"tipset_height"`
	MineTime          uint64      `bson:"mine_time" json:"mine_time"`

	GmtCreate   int64 `bson:"gmt_create" json:"gmt_create"`
	GmtModified int64 `bson:"gmt_modified" json:"gmt_modified"`
}

func (this *MinerStateAtTipset) State() *fspt.MinerState {
	if this == nil {
		return nil
	}
	return &fspt.MinerState{
		Address:      this.MinerAddr,
		Power:        utils.XSizeString(this.Power.Int),
		PowerPercent: utils.BigToPercent(this.Power.Int, this.TotalPower.Int),
		PeerId:       this.PeerId}
}

func BulkUpsertMiners(miners []interface{}) error {
	_, err := BulkUpsert(nil, MinerCollection, miners)
	return err
}

func UpsertMinerStateInTipset(miner *MinerStateAtTipset) error {
	go func() {
		err := UpdateIsOwnerByAdress(miner.WalletAddr)
		if err != nil {
			fmt.Println(ps("UpdateIsOwnerByAdress address =[%v],err=[%v]", miner.WalletAddr, err.Error()))
		}
		err = UpdateIsMinerByAdress(miner.MinerAddr)
		if err != nil {
			fmt.Println(ps("UpdateIsMinerByAdress address =[%v],err=[%v]", miner.WalletAddr, err.Error()))
		}
	}()
	_, err := Upsert(MinerCollection, bson.M{"miner_addr": miner.MinerAddr, "tipset_height": miner.TipsetHeight}, miner)
	return err
}

func FindMinerStateAtTipset(address address.Address, tipset_height uint64) (*MinerStateAtTipset, error) {
	miner := &MinerStateAtTipset{}

	query := bson.M{"miner_addr": address.String()}
	if tipset_height > 0 {
		query["tipset_height"] = tipset_height
	}
	err := FindOne(MinerCollection, query, nil, miner)
	return miner, err
}

func MaxBlockHegith() (uint64, error) {
	var res []FilscanBlockResult
	err := FindSortLimit(BlocksCollection, "-block_header.Height", nil,
		bson.M{"block_header.Height": 1}, &res, 0, 1)
	if err != nil {
		return 0, nil
	}
	if len(res) > 0 {
		return res[0].BlockHeader.Height, nil
	} else {
		return 0, nil
	}
}

func MinerListByWalletAddr(walletAddress string) (res []string, err error) {
	if len(walletAddress) < 1 {
		return
	}
	q := bson.M{"wallet_addr": walletAddress}
	//var result []interface{}
	err = Distinct(MinerCollection, "miner_addr", q, &res)
	return
}

func MinerByAddress(address string) (miner *MinerStateAtTipset, err error) {
	q := bson.M{"miner_addr": address}
	var res []*MinerStateAtTipset
	err = FindSortLimit(MinerCollection, "-mine_time", q, nil, &res, 0, 1)
	if err != nil {
		return
	}
	if len(res) < 1 {
		return
	}
	return res[0], nil
}

func MinerByPeerId(address string) (miner *MinerStateAtTipset, err error) {
	q := bson.M{"peer_id": address}
	var res []*MinerStateAtTipset
	err = FindAll(MinerCollection, q, nil, &res)
	if err != nil {
		return
	}
	if len(res) > 0 {
		return res[0], nil
	} else {
		return nil, nil
	}
}

func HaveMinerStateAt(address string, tipset_height uint64) bool {
	count, err := FindCount(MinerCollection, bson.M{"miner_addr": address, "tipset_height": tipset_height}, nil)
	if err != nil {
		return false
	}
	return count > 0
}

func GetMinerstateActivateAtTime(attime uint64) ([]*MinerStateAtTipset, error) {
	ms, c := Connect(MinerCollection)
	defer ms.Close()

	begintime := attime - (60 * 60 * 24)
	time.Unix(int64(attime), 0)

	ops := []bson.M{
		{"$match": bson.M{"mine_time": bson.M{"$gte": begintime, "$lt": attime}}},
		{"$sort": bson.M{"tipset_height": -1, "power": -1}},
		{"$group": bson.M{"_id": bson.M{"mine_addr": "$miner_addr"},
			"records": bson.M{"$first": "$$ROOT"},
		}},
	}
	type minerStateRecordInterface struct {
		Id      string      `bson:"_id" json:"id"`
		Records interface{} `bson:"record" json:"record"`
	}
	var res = []minerStateRecordInterface{}

	if err := c.Pipe(ops).All(&res); err != nil {
		return nil, err
	}

	records, err := ToMineState(res)
	if err != nil {
		return nil, err
	}
	var minerStates = make([]*MinerStateAtTipset, len(records))
	for index, record := range records {
		minerStates[index] = record.Record
	}
	return minerStates, nil
}

type MinerStateRecord struct {
	Id     string              `bson:"_id" json:"id"`
	Record *MinerStateAtTipset `bson:"record" json:"record"`
}

func ToMineState(in interface{}) ([]MinerStateRecord, error) {
	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var res = []MinerStateRecord{}
	err = json.Unmarshal(data, &res)

	return res, err
}

/**
db.miner.find({
	"mine_time": {
		"$lte": 1576807695
	}
}).sort({"mine_time":-1}).skip(0).limit(1)
*/

func GetTotalpowerAtTime(timestop uint64) (*MinerStateAtTipset, error) {
	q := bson.M{"mine_time": bson.M{"$lte": timestop}}
	var res []*MinerStateAtTipset
	err := FindSortLimit(MinerCollection, "-mine_time", q, nil, &res, 0, 1)
	if err != nil {
		return nil, err
	}
	if len(res) > 0 {
		return res[0], nil
	} else {
		return nil, nil
	}
}

func Create_block_reward_index() {
	ms, c := Connect(BlockRewardCollection)
	defer ms.Close()

	ms.SetMode(mgo.Monotonic, true)

	indexs := []mgo.Index{
		{Key: []string{"height"}, Unique: true, Background: true},
	}

	for _, index := range indexs {
		if err := c.EnsureIndex(index); err != nil {
			panic(err)
		}
	}
}
func GetDistinctMinerAddressByTime(startTime, endTime int64) (res []string, err error) {
	q := bson.M{"gmt_create": bson.M{"$gte": startTime, "$lte": endTime}}
	//err = Distinct(MsgCollection, "message.Method", q, &res)
	err = Distinct(MinerCollection, "miner_addr", q, &res)
	return
}
func GetDistinctWalletAddressByTime(startTime, endTime int64) (res []string, err error) {
	q := bson.M{"gmt_create": bson.M{"$gte": startTime, "$lte": endTime}}
	//err = Distinct(MsgCollection, "message.Method", q, &res)
	err = Distinct(MinerCollection, "wallet_addr", q, &res)
	return
}

func GetMinersByT3(t3s []string) (map[string]string, error) {
	ms, c := Connect(MinerCollection)
	defer ms.Close()

	// db.miner.aggregate([
	// {"$match":{"wallet_addr": {"$in": ["t3ucd6jd3xfzmnezoim4wohl4k3aju5l57qwy2ca7snbhmkv6uskci2fn3r6v6vpa4456ewhgnybwc64j3hdgq"]},}},
	// {"$group":{"_id":"$wallet_addr", "miner":{"$first":"$miner_addr"}}},
	// {"$project":{"_id":0, "wallet_addr":"$_id", "miner":1}} ])
	q_pipe := []bson.M{
		{"$match": bson.M{"wallet_addr": bson.M{"$in": t3s}}},
		{"$group": bson.M{"_id": "$wallet_addr", "miner": bson.M{"$first": "$miner_addr"}}},
		{"$project": bson.M{"_id": 0, "wallet_addr": "$_id", "miner": 1}}}

	res := []*struct {
		WalletAddr string `bson:"wallet_addr"`
		Miner      string `bson:"miner"`
	}{}

	if err := c.Pipe(q_pipe).All(&res); err != nil {
		return nil, err
	}

	x := make(map[string]string)
	for _, v := range res {
		x[v.Miner] = v.WalletAddr
	}

	return x, nil
}
