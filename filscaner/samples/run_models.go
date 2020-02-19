package main

import (
	"filscan_lotus/models"
	"filscan_lotus/utils"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	. "filscan_lotus/filscaner"
)

func init() {
	models.Db_init(utils.GetConfiger())
	mgo.SetLogger(new(MongoLog))
}

func main() {
	models_miner_power_increase_top_n(1576741815, 1576992780, 0, 1)
}

func models_miner_power_increase_top_n(start, end, offset, limit uint64) (uint64, error) {
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
		return 0, nil
	}

	q_pipe := []bson.M{
		{"$match": bson.M{"mine_time": bson.M{"$gte": start, "$lt": end}}},
		{"$sort": bson.M{"mine_time": -1}},
		{"$group": bson.M{"_id": bson.M{"miner_addr": "$miner_addr"}, "record": bson.M{"$first": "$$ROOT"}, "old_power": bson.M{"$last": "$power"}}},
		{"$project": bson.M{"increased_power": bson.M{"$subtract": []string{"$record.power", "$old_power"}}, "record": "$record"}},
		{"$sort": bson.M{"increased_power": -1}},
		{"$skip": offset}, {"$limit": limit}}

	q_res := []struct {
		Increased_power uint64      `bson:"increased_power" json:"increased_power"`
		Record          interface{} `bson:"record" json:"record"`
	}{}

	if err := c.Pipe(q_pipe).All(&q_res); err != nil {
		panic(err)
		return 0, nil
	}

	records := []*MinerIncreasedPowerRecord{}
	if err := utils.UnmarshalJSON(&q_res, &records); err != nil {
		panic(err)
	}

	return 0, nil
}
