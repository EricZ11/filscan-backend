package filscaner

import (
	inner_err "filscan_lotus/error"
	. "filscan_lotus/filscanproto"
	"filscan_lotus/models"
	"filscan_lotus/utils"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"math/big"
	"sort"
	"strconv"
	"time"
)

// https://docs.mongodb.com/manual/reference/operator/aggregation/first/#grp._S_first
// https://stackoverflow.com/questions/6498506/mongodb-select-the-top-n-rows-from-each-group
// https://stackoverflow.com/questions/34375163/how-to-use-mongodb-aggregate-to-get-the-first-of-each-group-including-nulls
// https://stackoverflow.com/questions/34325714/how-to-get-lastest-n-records-of-each-group-in-mongodb
// https://stackoverflow.com/questions/16409719/can-i-get-first-document-not-field-in-mongodb-aggregate-query

type Models_Block_reward struct {
	Height          uint64             `bson:"height"`
	ReleasedRewards *models.BsonBigint `bson:"reward"`
}

func (fs *Filscaner) do_upsert_miners() error {
	if fs.to_update_miner_index <= 0 {
		return nil
	}

	var offset uint64

	if fs.to_update_miner_index >= fs.to_update_miner_size {
		offset = fs.to_update_miner_index * 2
	}

	if err := models.BulkUpsertMiners(fs.to_upsert_miners[0:offset]); err != nil {
		return err
	}
	fs.to_update_miner_index = 0

	return nil
}

func (fs *Filscaner) models_update_miner(miner *models.MinerStateAtTipset) error {
	var err error

	if fs.to_update_miner_index >= fs.to_update_miner_size {
		if err = fs.do_upsert_miners(); err != nil {
			return err
		}
	}

	miner.GmtCreate = time.Now().Unix()
	miner.GmtModified = miner.GmtCreate

	offset := fs.to_update_miner_index * 2

	fs.to_upsert_miners[offset] = bson.M{"miner_addr": miner.MinerAddr, "tipset_height": miner.TipsetHeight}
	fs.to_upsert_miners[offset+1] = miner
	fs.to_update_miner_index++

	return nil
}

func (fs *Filscaner) models_get_minerstate_at_tipset(address address.Address,
	tipset_height uint64) (*models.MinerStateAtTipset, error) {
	return models.FindMinerStateAtTipset(address, tipset_height)
}

func (fs *Filscaner) models_search_miner(searchtxt string) ([]string, error) {
	ms, c := models.Connect(models.MinerCollection)
	defer ms.Close()

	miners := struct {
		Count  uint64
		Miners []string
	}{}

	q_find := []bson.M{
		{"$match": bson.M{"$or": []bson.M{{"peer_id": searchtxt}, bson.M{"miner_addr": searchtxt}}}},
		{"$sort": bson.M{"miner_time": -1}},
		{"$group": bson.M{"_id": "$miner_addr", "record": bson.M{"$first": "$$ROOT"}}},
		{"$group": bson.M{"_id": nil, "miners": bson.M{"$push": "$record.miner_addr"}, "count": bson.M{"$sum": 1}}},
		{"$project": bson.M{"_id": 0}}}
	if err := c.Pipe(q_find).One(&miners); err != nil {
		return nil, err
	}

	return miners.Miners, nil

}

type mashalMinerStateInTipset struct {
	PeerId            string      `bson:"peer_id" json:"peer_id"`
	MinerCreate       string      `bson:"miner_create" json:"miner_create"`
	MinerAddr         string      `bson:"miner_addr" json:"miner_addr"`
	NickName          string      `bson:"nick_name" json:"nick_name"`
	BlockCount        uint64      `bson:"block_count" json:"block_count"`
	Power             interface{} `bson:"power" json:"power"`
	TotalPower        interface{} `bson:"total_power" json:"total_power"`
	WalletAddr        string      `bson:"wallet_addr" json:"wallet_addr"`
	SectorSize        uint64      `bson:"sector_size" json:"sector_size"`
	SectorCount       uint64      `bson:"sector_count" json:"sector_count"`
	BlockCountPercent float64     `bson:"block_count_percent" json:"block_count_percent"`
	PowerPercent      float64     `bson:"power_percent" json:"power_percent"`

	ProvingSectorSize interface{} `bson:"incoming_sectorsize" json:"incoming_sectorsize"`
	TipsetHeight      uint64      `bson:"tipset_height" json:"tipset_height"`
	MineTime          uint64      `bson:"mine_time" json:"mine_time"`

	GmtCreate   int64 `bson:"gmt_create" json:"gmt_create"`
	GmtModified int64 `bson:"gmt_modified" json:"gmt_modified"`
}

func Models_miner_state_in_time(c *mgo.Collection, miners []string, at, start uint64) ([]*models.MinerStateAtTipset, error) {
	miner_size := len(miners)
	if miner_size == 0 {
		return nil, nil
	}

	if c == nil {
		var session *mgo.Session
		session, c = models.Connect(models.MinerCollection)
		defer session.Close()
	}

	q_res := struct {
		Miners []*models.MinerStateAtTipset `bson:"miners"`
	}{}

	q_pipe := []bson.M{
		{"$match": bson.M{"mine_time": bson.M{"$gt": start, "$lte": at}, "miner_addr": bson.M{"$in": miners}}},
		{"$sort": bson.M{"mine_time": -1}},
		{"$group": bson.M{"_id": bson.M{"miner_addr": "$miner_addr"}, "miner": bson.M{"$first": "$$ROOT"}}},
		{"$group": bson.M{"_id": nil, "miners": bson.M{"$push": "$miner"}}},
	}

	colation := &mgo.Collation{Locale: "zh", NumericOrdering: true}
	if err := c.Pipe(q_pipe).Collation(colation).AllowDiskUse().One(&q_res); err != nil {
		return nil, err
	}

	return q_res.Miners, nil
}

func models_miner_state_exist_newer(miner string, time int64) bool {
	c, err := models.FindCount(models.MinerCollection,
		bson.M{"miner_addr": miner, "mine_time": bson.M{"$gt": time}}, nil)
	if err != nil {
		return false
	}
	return c > 0
}

// todo: use a loop to search miner instead of use '$match in ...', which may improve proformance
func (fs *Filscaner) models_miner_power_increase_in_time(miners []string, start, end uint64) (map[string]*MinerIncreasedPowerRecord, error) {
	if start >= end {
		return nil, inner_err.ErrInvalidParam
	}

	ms, c := models.Connect(models.MinerCollection)
	defer ms.Close()

	var match = bson.M{}
	miner_size := len(miners)
	if miner_size != 0 {
		if true {
			ors := make([]bson.M, miner_size)
			for index, m := range miners {
				ors[index] = bson.M{"miner_addr": m}
			}
			// {$match:{$or:[{"miner_addr":"to1234"},{"miner_addr":"t01111"}]}},
			match = bson.M{"$or": ors}
		} else {
			match = bson.M{"miner_addr": bson.M{"$in": miners}}
		}
	}

	var mine_time_match bson.M
	if start > 0 {
		mine_time_match = bson.M{"$gte": start}
	}
	if end > 0 {
		if mine_time_match != nil {
			mine_time_match["$lt"] = end
		} else {
			mine_time_match = bson.M{"$lt": end}
		}
	}

	if mine_time_match != nil {
		match["mine_time"] = mine_time_match
	}

	q_pipe := []bson.M{
		{"$match": match},
		{"$sort": bson.M{"mine_time": -1}},
		{"$addFields": bson.M{"fpower": bson.M{"$toDouble": "$power"}}},
		{"$group": bson.M{"_id": bson.M{"miner_addr": "$miner_addr"}, "fmin_power": bson.M{"$last": "$fpower"}, "fmax_power": bson.M{"$first": "$fpower"}, "record": bson.M{"$first": "$$ROOT"}}},
		{"$project": bson.M{"increased_power": bson.M{"$subtract": []string{"$fmax_power", "$fmin_power"}}, "record": "$record"}}}

	q_res := []*MinerIncreasedPowerRecord{}

	if err := c.Pipe(q_pipe).Collation(fs.colation).AllowDiskUse().All(&q_res); err != nil {
		return nil, err
	}
	res := make(map[string]*MinerIncreasedPowerRecord)
	for _, r := range q_res {
		res[r.Record.MinerAddr] = r
	}
	return res, nil
}

func (fs *Filscaner) get_totalpower_at_time(timestop uint64) (*models.MinerStateAtTipset, error) {
	ms, c := models.Connect(models.MinerCollection)
	defer ms.Close()
	match := bson.M{}
	if timestop > 0 {
		match["mine_time"] = bson.M{"$lte": timestop}
	}

	ops := []bson.M{
		{"$match": match},
		{"$sort": bson.M{"tipset_height": -1, "power": -1}},
		{"$group": bson.M{
			"_id":        bson.M{"mine_addr": "$miner_addr"},
			"record":     bson.M{"$first": "$$ROOT"},
			"totalpower": bson.M{"$max": "$totalpower"}}},
		// {"$sort": bson.M{"record.power": -1}},
	}

	var res = []MinerStateRecordInterface{}
	if err := c.Pipe(ops).All(&res); err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	records := []models.MinerStateRecord{}
	if err := utils.UnmarshalJSON(res, &records); err != nil {
		return nil, err
	}
	return records[0].Record, nil
}

func models_miner_top_power(c *mgo.Collection, time_at, ofset, limit int64) ([]*models.MinerStateAtTipset, uint64, error) {
	if c == nil {
		var session *mgo.Session
		session, c = models.Connect(models.MinerCollection)
		defer session.Close()
	}

	q_match := bson.M{}

	if time_at != 0 {
		q_match = bson.M{"mine_time": bson.M{"$lt": time_at}}
	}

	q_count := []bson.M{
		{"$match": q_match},
		{"$group": bson.M{"_id": bson.M{"miner_addr": "$miner_addr"}}},
		{"$group": bson.M{"_id": "", "count": bson.M{"$sum": 1}}},
	}

	colation := &mgo.Collation{Locale: "zh", NumericOrdering: true}
	// db.miner.aggregate([
	// {"$match": {"mine_time":{"$lt":1577176373}}},
	// {"$sort": {"tipset_height": -1, "power": -1}},
	// {"$group": {
	// 	"_id":    {"miner_addr": "$miner_addr"},
	// 	"record": {"$first": "$$ROOT"},
	// }},
	// {"$sort": {"record.power": -1}},
	// {"$skip": 0},
	// {"$limit": 10}])
	q_count_res := struct{ Count uint64 }{}
	if err := c.Pipe(q_count).Collation(colation).AllowDiskUse().One(&q_count_res); err != nil {
		return nil, 0, err
	}

	q_result := []bson.M{
		{"$match": q_match},
		{"$sort": bson.M{"tipset_height": -1, "power": -1}},
		{"$group": bson.M{
			"_id":    bson.M{"miner_addr": "$miner_addr"},
			"record": bson.M{"$first": "$$ROOT"},
		}},
		{"$sort": bson.M{"record.power": -1}},
		{"$skip": ofset},
		{"$limit": limit},
	}

	var res = []struct {
		Record *models.MinerStateAtTipset `json:"record" bson:"record"`
	}{}

	if err := c.Pipe(q_result).Collation(colation).AllowDiskUse().All(&res); err != nil {
		return nil, 0, err
	}

	miners := make([]*models.MinerStateAtTipset, len(res))

	for index, miner := range res {
		miners[index] = miner.Record
	}

	return miners, q_count_res.Count, nil
}

func (fs *Filscaner) delete_minerstate_at(tipset_height uint64) error {
	return models.Remove(
		models.MinerCollection,
		bson.M{"tipset_height": tipset_height})
}

func (fs *Filscaner) get_minerstate_lte2(address address.Address, smollerthan uint64) ([]*models.MinerStateAtTipset, error) {
	miner := []*models.MinerStateAtTipset{}
	err := models.FindSortLimit(models.MinerCollection, "-tipset_height",
		bson.M{
			"tipset_height": bson.M{"$lte": smollerthan},
			"miner_addr":    address.String(),
		},
		nil, &miner, 0, 2)
	return miner, err
}

func (fs *Filscaner) to_resp_slice(in []*models.MinerStateAtTipset) []*MinerState {
	var minerStates = make([]*MinerState, len(in))
	for index, miner := range in {
		minerStates[index] = miner.State()
	}
	return minerStates
}

func (fs *Filscaner) to_resp_map(in map[string]*models.MinerStateAtTipset) map[string]*MinerState {
	var minerStates = make(map[string]*MinerState)

	for k, miner := range in {
		minerStates[k] = &MinerState{
			Address:      miner.MinerAddr,
			Power:        utils.XSizeString(miner.Power.Int),
			PowerPercent: utils.BigToPercent(miner.Power.Int, miner.TotalPower.Int),
			PeerId:       miner.PeerId,
		}
	}
	return minerStates
}

func (fs *Filscaner) get_minerstate_activate_at_time(attime uint64) ([]*models.MinerStateAtTipset, error) {
	ms, c := models.Connect(models.MinerCollection)
	defer ms.Close()

	begintime := attime - (60 * 60 * 24)
	time.Unix(int64(attime), 0)

	ops := []bson.M{
		{"$match": bson.M{"mine_time": bson.M{"$gte": begintime, "$lt": attime}}},
		{"$sort": bson.M{"tipset_height": -1, "power": -1}},
		{"$group": bson.M{"_id": bson.M{"mine_addr": "$miner_addr"},
			"record": bson.M{"$first": "$$ROOT"},
		}},
	}

	var res = []MinerStateRecordInterface{}
	if err := c.Pipe(ops).All(&res); err != nil {
		return nil, err
	}

	records := []models.MinerStateRecord{}
	if err := utils.UnmarshalJSON(res, &records); err != nil {
		return nil, err
	}

	var minerStates = make([]*models.MinerStateAtTipset, len(records))
	for index, record := range records {
		minerStates[index] = record.Record
	}

	return minerStates, nil
}

func (fs *Filscaner) model_active_miner_count_at_time(at_time, time_diff uint64) (uint64, error) {
	ms, c := models.Connect(models.MinerCollection)
	defer ms.Close()

	var begintime uint64
	if at_time > time_diff {
		begintime = at_time - time_diff
	} else {
		begintime = 0
	}

	// db.miner.aggregate([
	// {"$match": {"mine_time": {"$gte":1577404800, "$lt":1577491200}}},
	// {"$sort": {"tipset_height": -1, "power": -1}},
	// {"$group": {"_id": {"mine_addr": "$miner_addr"}}},
	// {"$group": {"_id":null, "count": {"$sum": 1}}} ])
	ops := []bson.M{
		{"$match": bson.M{"mine_time": bson.M{"$gte": begintime, "$lt": at_time}}},
		{"$sort": bson.M{"tipset_height": -1, "power": -1}},
		{"$group": bson.M{"_id": bson.M{"mine_addr": "$miner_addr"}}},
		{"$group": bson.M{"_id": nil, "count": bson.M{"$sum": 1}}}}

	res := &struct {
		Count uint64
	}{}

	if err := c.Pipe(ops).Collation(fs.colation).AllowDiskUse().One(res); err != nil {
		return 0, err
	}

	return res.Count, nil
}

// func (fs *Filscaner) models_miner_power_increase_time_range(miners []string, start, end uint64) (map[string]*big.Int, error) { if start >= end {
// 		return nil, inner_err.ErrInvalidParam
// 	}
//
// 	ms, c := models.Connect(models.MinerCollection)
// 	defer ms.Close()
//
// 	q_pipe := []bson.M{
// 		{"$match": bson.M{"miner":bson.M{"$in":miners}, "mine_time": bson.M{"$gte": start, "$lt": end}}},
// 		{"$sort": bson.M{"mine_time": -1}},
// 		{"$addFields": bson.M{"fpower": bson.M{"$toDouble": "$power"}}},
// 		{"$group": bson.M{"_id": bson.M{"miner_addr": "$miner_addr"}, "fmin_power": bson.M{"$last": "$fpower"}, "fmax_power": bson.M{"$first": "$fpower"} }},
// 		{"$project": bson.M{"miner":"$_id.miner_addr", "increased_power": bson.M{"$subtract": []string{"$fmax_power", "$fmin_power"}}}},
// 		{"$sort": bson.M{"increased_power": -1}}}
//
// 	q_res := []*struct {
// 		Miner string `bson:"miner"`
// 		Increased_power string `bson:"miner"`
// 	}{}
//
// 	if err := c.Pipe(q_pipe).Collation(fs.colation).AllowDiskUse().All(&q_res); err != nil {
// 		return nil, err
// 	}
//
// 	res := make(map[string]*big.Int)
// 	for _, v := range q_res {
// 		res[v.Miner], _ = big.NewInt(0).SetString(v.Increased_power, 10)
// 	}
// 	return res, nil
// }

func (fs *Filscaner) models_miner_power_increase_top_n(start, end, offset, limit uint64) ([]*MinerIncreasedPowerRecord, uint64, error) {
	if start >= end {
		return nil, 0, inner_err.ErrInvalidParam
	}

	ms, c := models.Connect(models.MinerCollection)
	defer ms.Close()

	q_count := []bson.M{
		{"$match": bson.M{"mine_time": bson.M{"$gte": start, "$lt": end}}},
		{"$group": bson.M{"_id": bson.M{"miner_addr": "$miner_addr"}}},
		{"$group": bson.M{"_id": nil, "count": bson.M{"$sum": 1}}},
	}

	q_count_res := struct {
		Count uint64
	}{}
	if err := c.Pipe(q_count).One(&q_count_res); err != nil {
		panic(err)
		return nil, 0, nil
	}

	// db.miner.aggregate([
	// {"$match": {"mine_time": {"$gte":1576741815}}},
	// {"$sort": {"mine_time": -1}},
	// {"$addFields": { "fpower": { "$toDouble": "$power"}}},
	// {"$group": {"_id": {"miner_addr": "$miner_addr"}, "fmin_power": {"$last": "$fpower"}, "fmax_power":{"$first":"$fpower"}, "record": {"$first": "$$ROOT"}}},
	// {"$project": {"increased_power": {"$subtract": ["$fmax_power", "$fmin_power"]}, "record": "$record"}},
	// {"$sort": {"increased_power":-1}},
	// {"$skip":0},
	// {"$limit":5} ])

	q_pipe := []bson.M{
		{"$match": bson.M{"mine_time": bson.M{"$gte": start, "$lt": end}}},
		{"$sort": bson.M{"mine_time": -1}},
		{"$addFields": bson.M{"fpower": bson.M{"$toDouble": "$power"}}},
		{"$group": bson.M{"_id": bson.M{"miner_addr": "$miner_addr"}, "fmin_power": bson.M{"$last": "$fpower"}, "fmax_power": bson.M{"$first": "$fpower"}, "record": bson.M{"$first": "$$ROOT"}, "old_power": bson.M{"$last": "$power"}}},
		{"$project": bson.M{"increased_power": bson.M{"$subtract": []string{"$fmax_power", "$fmin_power"}}, "record": "$record"}},
		{"$sort": bson.M{"increased_power": -1}},
		{"$skip": offset}, {"$limit": limit}}

	q_res := []*MinerIncreasedPowerRecord{}

	if err := c.Pipe(q_pipe).Collation(fs.colation).AllowDiskUse().All(&q_res); err != nil {
		return nil, 0, err
	}

	return q_res, q_count_res.Count, nil
}

func (fs *Filscaner) models_miner_block_top_n(start, end, offset, limit uint64) ([]*MinedBlock, uint64, error) {
	if start >= end {
		return nil, 0, inner_err.ErrInvalidParam
	}

	ms, c := models.Connect(models.BlocksCollection)
	defer ms.Close()

	// db.block.aggregate([
	// {"$match": {"block_header.Timestamp": {"$gte":0x5dfb2bb7, "$lt":0x5dff000c}}},
	// {"$group": {"_id": {"miner":"$block_header.Miner"}}},
	// {"$group": {"_id": null, "miner_count":{"$sum":1}}} ])
	q_miner_count := []bson.M{
		{"$match": bson.M{"block_header.Timestamp": bson.M{"$gte": start, "$lt": end}}},
		{"$group": bson.M{"_id": bson.M{"miner": "$block_header.Miner"}}},
		{"$group": bson.M{"_id": nil, "miner_count": bson.M{"$sum": 1}}},
	}
	q_count_res := struct {
		MinerCount uint64 `bson:"miner_count" json:"miner_count"`
	}{}
	if err := c.Pipe(q_miner_count).One(&q_count_res); err != nil {
		panic(err)
		return nil, 0, nil
	}
	// db.block.aggregate([
	// {"$match": {"block_header.Timestamp": {"$gte":0x5dfb2bb7, "$lt":0x5dff000c}}},
	// {"$group": {
	// 	"_id": {"miner":"$block_header.Miner"},
	// 	"mined_block_count":{"$sum":1},
	// 	"miner":{"$first":"$block_header.Miner"}, } },
	// {"$sort":{"mined_block_count":-1}}, ])
	// TODO:这里可能需要把满足条件的区块数量计算出来再返回..
	q_miners := []bson.M{
		{"$match": bson.M{"block_header.Timestamp": bson.M{"$gte": start, "$lt": end}}},
		{"$group": bson.M{
			"_id":               bson.M{"miner": "$block_header.Miner"},
			"mined_block_count": bson.M{"$sum": 1},
			"miner":             bson.M{"$first": "$block_header.Miner"}}},
		{"$sort": bson.M{"mined_block_count": -1}},
		{"$skip": offset},
		{"$limit": limit}}

	q_miner_res := []*MinedBlock{}

	if err := c.Pipe(q_miners).All(&q_miner_res); err != nil {
		return nil, 0, nil
	}

	return q_miner_res, q_count_res.MinerCount, nil
}

func (fs *Filscaner) models_get_tipset_at_time(time_at uint64, befor bool) (uint64, error) {
	res := []*struct {
		Height uint64 `json:"height" bson:"height"`
	}{}

	cond := "$gte"
	sort := "mine_time"

	if befor {
		cond = "$lt"
		sort = "-mine_time"
	}

	err := models.FindSortLimit("tipset", sort, bson.M{"mine_time": bson.M{cond: time_at}},
		bson.M{"height": 1}, &res, 0, 1)
	if err != nil {
		return 0, err
	}

	if len(res) > 0 {
		return res[0].Height, nil
	}
	return 0, nil
}

func (fs *Filscaner) models_blockcount_time_range(start, end uint64) (uint64, uint64, uint64, error) {
	ms, c := models.Connect(models.BlocksCollection)
	defer ms.Close()

	// db.block.aggregate([
	// {"$match":{"block_header.Timestamp":{"$gte":1577171478, "$lt":1577178678}}},
	// {"$sort":{"blockheader.Height":-1}},
	// {"$group": {"_id": {"miner": "$block_header.Miner"}, "mx_height":{"$first":"$block_header.Height"}, "mi_height":{"$last":"$block_header.Height"}}},
	// {"$group":{"_id":null, "miner_count":{"$sum":1}, "min_height":{"$min":"$mi_height"}, "max_height":{"$max":"$mx_height"}}} ])
	q_pipe := []bson.M{
		{"$match": bson.M{"block_header.Timestamp": bson.M{"$gte": start, "$lt": end}}},
		{"$sort": bson.M{"block_header.Height": -1}},
		{"$group": bson.M{"_id": bson.M{"miner": "$block_header.Miner"},
			"mx_height": bson.M{"$first": "$block_header.Height"},
			"mi_height": bson.M{"$last": "$block_header.Height"}}},
		{"$group": bson.M{"_id": nil, "miner_count": bson.M{"$sum": 1},
			"min_height": bson.M{"$min": "$mi_height"},
			"max_height": bson.M{"$max": "$mx_height"}}}}

	res := &struct {
		MaxHeight  uint64 `json:"max_height" bson:"max_height"`
		MinHeight  uint64 `json:"min_height" bson:"min_height"`
		MinerCount uint64 `json:"miner_count" bson:"miner_count"`
	}{}

	err := c.Pipe(q_pipe).One(res)
	if err != nil {
		return 0, 0, 0, err
	}

	return res.MinHeight, res.MaxHeight, res.MinerCount, nil
}

func (fs *Filscaner) models_total_block_count() (uint64, error) {
	ms, c := models.Connect(models.BlocksCollection)
	defer ms.Close()

	total, err := c.Find(bson.M{}).Count()
	return uint64(total), err
}

func (fs *Filscaner) models_blockcount_by_miner(miner string) (uint64, error) {
	ms, c := models.Connect(models.BlocksCollection)
	defer ms.Close()

	block_count, err := c.Find(bson.M{"block_header.Miner": miner}).Count()
	if err != nil {
		return 0, err
	}
	return uint64(block_count), nil
}

// 查找某段时间内的爆块总数, 和指定miner的爆块数量
func (fs *Filscaner) models_blockcount_time_range_with_miners(miners []string, start, end uint64) (map[string]uint64, uint64, error) {
	ms, c := models.Connect(models.BlocksCollection)
	defer ms.Close()

	totalcount, err := c.Find(bson.M{"block_header.Timestamp": bson.M{"$gt": start, "$lte": end}}).Count()
	if err != nil {
		return nil, 0, err
	}

	// db.block.aggregate([
	// {"$match": {"block_header.Miner": {"$in":["t06266","t01493"]}, "block_header.Timestamp": {"$gt":1577671322, "$lte":1577757722}}},
	// {"$group": {"_id": {"miner": "$block_header.Miner"}, "block_count": {"$sum": 1}}},
	// {"$project": {"_id":0, "miner": "$_id.miner", "block_count":1}}, ])
	var q_pipe = []bson.M{
		{"$match": bson.M{"block_header.Miner": bson.M{"$in": miners}, "block_header.Timestamp": bson.M{"$gt": start, "$lte": end}}},
		{"$group": bson.M{"_id": bson.M{"miner": "$block_header.Miner"}, "block_count": bson.M{"$sum": 1}}},
		{"$project": bson.M{"_id": 0, "miner": "$_id.miner", "block_count": 1}}}
	q_res := []struct {
		Miner      string `bson:"miner"`
		BlockCount uint64 `bson:"block_count"`
	}{}
	err = c.Pipe(q_pipe).All(&q_res)
	if err != nil {
		return nil, 0, err
	}

	res := make(map[string]uint64)
	for _, m := range q_res {
		res[m.Miner] = m.BlockCount
	}

	return res, uint64(totalcount), nil
}

type Models_minerlist struct {
	TotalIncreasedPower  float64                   `bson:"total_increased_power"`
	TotalMinedBlockCount uint64                    `bson:"total_mined_block_count"`
	MinerCount           uint64                    `bson:"miner_count"`
	Miners               []*Modles_minerlist_miner `bson:"miners"`
	less                 func(i, j int) bool
}

func (ml *Models_minerlist) GetMiners() []string {
	length := len(ml.Miners)
	if length == 0 {
		return nil
	}

	miners := make([]string, length)
	for index, m := range ml.Miners {
		miners[index] = m.MinerAddress
	}
	return miners
}

func (ml *Models_minerlist) GetMinersMap() map[string]*Modles_minerlist_miner {
	if len(ml.Miners) == 0 {
		return nil
	}

	miner_map := make(map[string]*Modles_minerlist_miner)
	for _, m := range ml.Miners {
		miner_map[m.MinerAddress] = m
	}
	return miner_map
}

func (ml *Models_minerlist) APIRespData() *MinerListResp_Data {
	data := &MinerListResp_Data{}

	data.TotalIncreasedPower = strconv.FormatFloat(ml.TotalIncreasedPower, 'f', -1, 64)
	data.TotalIncreasedBlock = ml.TotalMinedBlockCount
	data.MienrCount = ml.MinerCount

	data.Miners = make([]*MinerInfo, len(ml.Miners))

	for index, m := range ml.Miners {
		info := &MinerInfo{
			IncreasedPower:   strconv.FormatFloat(m.IncreasedPower, 'f', -1, 64),
			IncreasedBlock:   m.MinedBlockCount,
			Miner:            m.MinerAddress,
			PeerId:           m.PeerId,
			PowerPercent:     utils.FloatToPercent(m.IncreasedPower, ml.TotalIncreasedPower),
			BlockPercent:     utils.IntToPercent(m.MinedBlockCount, ml.TotalMinedBlockCount),
			MiningEfficiency: m.MiningEfficiency,
			StorageRate:      m.PowerRate,
		}
		// StorageRate:      m.PowerRate}
		data.Miners[index] = info
	}
	return data
}

// returns no-setted miner addresses
func (ml *Models_minerlist) SetPowerValues(ml_src *Models_minerlist) {
	miner_map := ml_src.GetMinersMap()
	if miner_map == nil {
		return
	}

	// var unset_miners []string
	ml.TotalIncreasedPower = ml_src.TotalIncreasedPower

	for _, miner := range ml.Miners {
		src_miner, isok := miner_map[miner.MinerAddress]
		if !isok || src_miner == nil {
			miner.PowerRate = "0.00"
			continue
		}
		miner.IncreasedPower = src_miner.IncreasedPower
		miner.PeerId = src_miner.PeerId
		miner.WalletAddress = src_miner.WalletAddress
		miner.PowerRate = src_miner.PowerRate
		// miner.MiningEfficiency = src_miner.MiningEfficiency
	}
}

func (ml *Models_minerlist) SetBlockValues(ml_src *Models_minerlist) {
	miner_map := ml_src.GetMinersMap()
	if miner_map == nil {
		return
	}
	ml.TotalMinedBlockCount = ml_src.TotalMinedBlockCount
	for _, miner := range ml.Miners {
		src_miner, isok := miner_map[miner.MinerAddress]
		if !isok || src_miner == nil {
			continue
		}
		miner.MinedBlockCount = src_miner.MinedBlockCount
	}
}

func (ml *Models_minerlist) SortBYMiningEfficency(sort_type int) {
	ml.less = ml.less_mining_efficency
	if sort_type < 0 {
		sort.Sort(sort.Reverse(ml))
	} else {
		sort.Sort(ml)
	}
}

func (ml *Models_minerlist) Len() int {
	return len(ml.Miners)
}

func (ml *Models_minerlist) less_mining_efficency(i, j int) bool {
	if ml.Miners[i].MiningEfficiency == "+Inf" {
		return true
	}
	if ml.Miners[j].MiningEfficiency == "+Inf" {
		return false
	}

	fi := utils.StringToFloat(ml.Miners[i].MiningEfficiency)
	fj := utils.StringToFloat(ml.Miners[j].MiningEfficiency)
	return fi < fj
}

func (ml *Models_minerlist) Less(i, j int) bool {
	return ml.less(i, j)
}

func (ml *Models_minerlist) Swap(i, j int) {
	ml.Miners[i], ml.Miners[j] = ml.Miners[j], ml.Miners[i]
}

func (ml *Models_minerlist) SetBlockEfficiency(ml_src *Models_minerlist) {
	miner_map := ml_src.GetMinersMap()
	unit_gb := float64(1 << 30)
	if miner_map == nil {
		return
	}

	for _, miner := range ml.Miners {
		src_miner, isok := miner_map[miner.MinerAddress]
		if !isok || src_miner == nil {
			continue
		}
		miner.PeerId = src_miner.PeerId
		miner.MiningEfficiency = fmt.Sprintf("%.4f", float64(miner.MinedBlockCount)*unit_gb/src_miner.IncreasedPower)
	}
}

type Modles_minerlist_miner struct {
	IncreasedPower   float64 `bson:"increased_power" json:"increased_power"`
	MinedBlockCount  uint64  `bson:"mined_block_count" json:"mined_block_count"`
	MinerAddress     string  `bson:"miner_addr" json:"miner_addr"`
	WalletAddress    string  `bson:"wallet_addr" json:"wallet_addr"`
	PowerRate        string  `bson:"power_rate"`
	MiningEfficiency string  `bson:"mining_efficiency"`
	PeerId           string  `bson:"peer_id" json:"peer_id"`
}

func (fs *Filscaner) models_minerlist_sort_power(miners []string, start, end, offset, limit uint64, sort_field string, sort int) (*Models_minerlist, error) {
	ms, c := models.Connect(models.MinerCollection)
	defer ms.Close()

	var power_sort_field_map = map[string]string{
		"power":      "miner.increased_power",
		"power_rate": "miner.power_rate"}

	sort_field, use_sort := power_sort_field_map[sort_field]

	// db.miner.aggregate([
	// {"$match": {"mine_time": {"$gte":1577404800, "$lt":1577491200}}},
	// {"$sort": {"mine_time": -1}},
	// {"$addFields": {"fpower": {"$toDouble": "$power"}}},
	// {"$group": {"_id": {"miner": "$miner_addr"}, "fmin_power": {"$last": "$fpower"}, "fmax_power": {"$first": "$fpower"}, "miner": {"$first": "$$ROOT"}}},
	// {"$set":{"miner.increased_power":{"$subtract": ["$fmax_power", "$fmin_power"]}}},
	// {"$sort":{"miner.increased_power": -1}},
	// {"$group": {"_id": null, "miners": {"$push": "$miner"}, "total_increased_power":{"$sum":"$miner.increased_power"}, "miner_count": {"$sum": 1}}},
	// {"$set": {"miner_filters":["t06241", "t01475", "t01493", "t06594", "t12345"]}},
	// {"$project":{ "miners":{ "$filter": { "input": "$miners", "as": "miners", "cond": { "$in":["$$miners.miner_addr", "$miner_filters"]} } }, "miner_count":1, "total_increased_power":1}},
	// {"$project":{"miner_count":1, "total_increased_power":1, "miners":{"$slice":["$miners",0,5]}}}])

	q_pipe := []bson.M{
		{"$match": bson.M{"mine_time": bson.M{"$gte": start, "$lt": end}}},
		{"$sort": bson.M{"mine_time": -1}},
		{"$addFields": bson.M{"fpower": bson.M{"$toDouble": "$power"}}},
		{"$group": bson.M{"_id": bson.M{"miner": "$miner_addr"}, "fmin_power": bson.M{"$last": "$fpower"}, "fmax_power": bson.M{"$first": "$fpower"}, "miner": bson.M{"$first": "$$ROOT"}}},
		{"$set": bson.M{"miner.increased_power": bson.M{"$subtract": []string{"$fmax_power", "$fmin_power"}}}},
		{"$set": bson.M{"miner.power_rate": bson.M{"$divide": []interface{}{"$miner.increased_power", (float64(end-start) / 3600 * 1024 * 1024 * 1024)}}}}}

	if use_sort && sort_field != "" {
		q_pipe = append(q_pipe, bson.M{"$sort": bson.M{sort_field: sort}})
	}

	q_pipe = append(q_pipe, bson.M{"$set": bson.M{"miner.power_rate": bson.M{"$toString": "$miner.power_rate"}}})
	q_pipe = append(q_pipe, bson.M{"$group": bson.M{"_id": nil, "miners": bson.M{"$push": "$miner"}, "total_increased_power": bson.M{"$sum": "$miner.increased_power"}, "miner_count": bson.M{"$sum": 1}}})

	if len(miners) != 0 {
		q_pipe = append(q_pipe, bson.M{"$set": bson.M{"miner_filters": miners}})
		q_pipe = append(q_pipe, bson.M{"$project": bson.M{"miner_count": 1, "total_increased_power": 1,
			"miners": bson.M{"$filter": bson.M{"input": "$miners", "as": "miners", "cond": bson.M{"$in": []string{"$$miners.miner_addr", "$miner_filters"}}}}}})
	}

	q_pipe = append(q_pipe, bson.M{"$project": bson.M{"miner_count": 1, "total_increased_power": 1, "miners": bson.M{"$slice": []interface{}{"$miners", offset, limit}}}})
	q_res := &Models_minerlist{}

	if err := c.Pipe(q_pipe).Collation(fs.colation).AllowDiskUse().One(&q_res); err != nil {
		return nil, err
	}

	return q_res, nil
}

func (fs *Filscaner) models_miner_list_sort_block(miners []string, start, end, offset, limit uint64, sort_field string, sort int) (*Models_minerlist, error) {
	ms, c := models.Connect(models.BlocksCollection)
	defer ms.Close()

	var block_sort_field_map = map[string]string{
		"block":             "miner.mined_block_count",
		"mining_efficiency": "miner.mining_efficiency"}

	sort_field, use_sort := block_sort_field_map[sort_field]
	// db.block.aggregate([
	// { "$match": { "block_header.Timestamp": { "$gte": 1577404800, "$lt": 1577491200 } } },
	// { "$group": { "_id": { "miner": "$block_header.Miner" }, "block_count": { "$sum": 1 }, "miner": { "$first": "$block_header.Miner" } } },
	// { "$set": { "miner.mined_block_count": "$block_count", "miner.miner_addr": "$miner" } },
	// { "$set": { "miner.mining_efficiency": { "$cond": { "if": { "$lte": [0, "$miner.miner_block_count" ] }, "then": "n/a", "else": {"$divide":[{"$toDouble":"$miner.mined_block_count" }, 3600] } } } }},
	// {"$sort": {"miner.mined_block_count": -1}},
	// {"$group":{"_id":null, "miner_count":{"$sum":1}, "total_mined_block_count":{"$sum":"$miner.mined_block_count"},"miners":{"$push":"$miner"}}},
	// {"$set": {"miner_filters":["t06241", "t01475", "t01493", "t06594", "t12345"]}},
	// {"$project":{"_id":0, miners:{ $filter: { input: "$miners", as: "miners", cond: { $in:["$$miners.miner_addr", "$miner_filters"]} } }, miner_count:1, total_mined_block_count:1 }}
	// ])

	q_pipe := []bson.M{
		{"$match": bson.M{"block_header.Timestamp": bson.M{"$gte": start, "$lt": end}}},
		{"$group": bson.M{"_id": bson.M{"miner": "$block_header.Miner"}, "block_count": bson.M{"$sum": 1}, "miner": bson.M{"$first": "$block_header.Miner"}}},
		{"$set": bson.M{"miner.mined_block_count": "$block_count", "miner.miner_addr": "$miner"}},
		// {"$set": bson.M{"miner.mining_efficiency": bson.M{"$cond": bson.M{"if": bson.M{"$lte": []interface{}{0, "$miner.miner_block_count"}}, "then": "n/a", "else": bson.M{"$divide": []interface{}{bson.M{"$toDouble": "$miner.mined_block_count"}, }}}}}} ,
	}
	if use_sort {
		q_pipe = append(q_pipe, bson.M{"$sort": bson.M{sort_field: sort}})
	}

	q_pipe = append(q_pipe, bson.M{"$set": bson.M{"miner.mining_efficiency": bson.M{"$toString": "$miner.mining_efficiency"}}})
	q_pipe = append(q_pipe, bson.M{"$group": bson.M{"_id": nil, "miner_count": bson.M{"$sum": 1}, "total_mined_block_count": bson.M{"$sum": "$miner.mined_block_count"}, "miners": bson.M{"$push": "$miner"}}})

	if len(miners) != 0 {
		q_pipe = append(q_pipe, bson.M{"$set": bson.M{"miner_filters": miners}})
		q_pipe = append(q_pipe,
			bson.M{"$project": bson.M{"_id": 0, "miner_count": 1, "total_mined_block_count": 1,
				"miners": bson.M{"$filter": bson.M{"input": "$miners", "as": "ms", "cond": bson.M{"$in": []string{"$$ms.miner_addr", "$miner_filters"}}}}}})
	}
	q_pipe = append(q_pipe, bson.M{"$project": bson.M{"miner_count": 1, "total_mined_block_count": 1, "miners": bson.M{"$slice": []interface{}{"$miners", offset, limit}}}})

	q_res := &Models_minerlist{}

	if err := c.Pipe(q_pipe).Collation(fs.colation).AllowDiskUse().One(&q_res); err != nil {
		return nil, err
	}
	return q_res, nil
}

func models_block_released_rewards_at_height(height uint64) (*Models_Block_reward, error) {
	ms, c := models.Connect(models.BlockRewardCollection)
	defer ms.Close()

	block_reward := &Models_Block_reward{}

	err := c.Find(bson.M{"height": bson.M{"$lte": height}}).Sort("-height").Limit(1).One(block_reward)
	if err != nil {
		return nil, err
	}
	return block_reward, nil
}

func models_bulk_upsert_block_reward(brs []*Models_Block_reward, size int) error {
	upsert_pairs := make([]interface{}, size*2)

	for i := 0; i < size; i++ {
		br := brs[i]
		upsert_pairs[i*2] = bson.M{"height": br.Height}
		upsert_pairs[i*2+1] = br
	}

	_, err := models.BulkUpsert(nil, models.BlockRewardCollection, upsert_pairs)
	return err
}

func models_block_reward_head() (*Models_Block_reward, error) {
	ms, c := models.Connect(models.BlockRewardCollection)
	defer ms.Close()

	block_reward := &Models_Block_reward{}
	err := c.Find(nil).Sort("-height").One(&block_reward)
	if err != nil {
		if err == mgo.ErrNotFound {
			block_reward.ReleasedRewards = &models.BsonBigint{Int: big.NewInt(0)}
			return block_reward, nil
		}
		return nil, err
	}
	return block_reward, nil
}

func models_miner_heights(min, max uint64) (map[uint64]struct{}, error) {
	ms, c := models.Connect(models.BlockRewardCollection)
	defer ms.Close()

	heights := struct {
		Height []uint64 `bson:"heights"`
	}{}
	//db.miner.aggregate([
	//{"$match":{"tipset_height":{"$gte":40000, "$lt":50000}}},
	//{"$group":{"_id":"$tipset_height"}},
	//{"$project":{"_id":0, "height":"$_id"}},
	//{"$sort":{"height":1}},
	//{"$group":{_id:null, heights:{$push:"$height"}}},
	//{"$project":{heights:1, _id:0}} ] )

	heights_map := make(map[uint64]struct{})

	err := c.Pipe([]bson.M{
		{"$match": bson.M{"tipset_height": bson.M{"$gte": min, "$lt": max}}},
		{"$group": bson.M{"_id": "$tipset_height"}},
		{"$project": bson.M{"_id": 0, "height": "$_id"}},
		{"$sort": bson.M{"height": 1}},
		{"$group": bson.M{"_id": nil, "heights": bson.M{"$push": "$height"}}},
		{"$project": bson.M{"heights": 1, "_id": 0}}}).One(&heights)
	if err != nil {
		if err == mgo.ErrNotFound {
			return heights_map, nil
		}
		return nil, err
	}

	for _, h := range heights.Height {
		heights_map[h] = struct{}{}
	}
	return heights_map, nil
}

type models_miner_attime struct {
}

func models_minerinfo_at_time() {

}
