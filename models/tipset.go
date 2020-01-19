package models

import (
	"encoding/json"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/globalsign/mgo"
	"github.com/ipfs/go-cid"
	"gopkg.in/mgo.v2/bson"
)

type FilscanTipSet struct {
	Height       uint64       `bson:"height" json:"height"`
	Mintime      uint64       `bson:"mine_time" json:"mine_time"`
	Cids         []cid.Cid    `bson:"cids" json:"cids"`
	MinTicketCId cid.Cid      `bson:"min_ticket_block" json:"min_ticket_block"`
	Weight       types.BigInt `bson:"weight" json:"weight"`
	Parents      []cid.Cid    `bson:"parents" json:"parents"`
	TipSetCreate int64        `bson:"tipset_create" json:"tipset_create"`
	GmtCreate    int64        `bson:"gmt_create" json:"gmt_create"`
	GmtModified  int64        `bson:"gmt_modified" json:"gmt_modified"`
}

type FilscanTipSetResult struct {
	Height       uint64       `bson:"height" json:"height"`
	Cids         []FilscanCid `bson:"cids" json:"cids"`
	MinTicketCId FilscanCid   `bson:"min_ticket_block" json:"min_ticket_block"`
	Weight       string       `bson:"weight" json:"weight"`
	Parents      []FilscanCid `bson:"parents" json:"parents"`
	GmtCreate    int64        `bson:"gmt_create" json:"gmt_create"`
	GmtModified  int64        `bson:"gmt_modified" json:"gmt_modified"`
}

const (
	tipSetCollection = "tipset"
)

/**

db.tipset.ensureIndex({"height":-1})
db.tipset.ensureIndex({"gmt_create":-1})
db.tipset.ensureIndex({"cids./":1})
*/

func Create_tipset_index() {
	ms, c := Connect(tipSetCollection)
	defer ms.Close()
	ms.SetMode(mgo.Monotonic, true)

	indexs := []mgo.Index{
		{Key: []string{"height"}, Unique: false, Background: true},
		{Key: []string{"gmt_create"}, Unique: false, Background: true},
		{Key: []string{"cids./"}, Unique: false, Background: true},
	}
	for _, index := range indexs {
		if err := c.EnsureIndex(index); err != nil {
			panic(err)
		}
	}
}

func AddTipSet(t *types.TipSet) error {
	var tips FilscanTipSet
	tips.Cids = t.Cids()
	tips.Height = t.Height()
	tips.Mintime = t.MinTimestamp()
	tips.Parents = t.Parents().Cids()
	tips.GmtCreate = TimeNow
	tips.GmtModified = TimeNow
	tips.MinTicketCId = t.MinTicketBlock().Cid()
	tbyte, _ := json.Marshal(tips)
	var p interface{}
	err := json.Unmarshal(tbyte, &p)
	if err != nil {
		return err
	}
	s := bson.M{"height": t.Height()}
	_, err = Upsert(tipSetCollection, s, p)
	return err
}

func GetTipSetByHeight(start, end uint64) (res []*FilscanTipSetResult, err error) {
	q := bson.M{"height": bson.M{"$gte": start, "$lte": end}}
	err = FindAll(tipSetCollection, q, nil, &res)
	return
}

func GetTipSetByOneHeight(height uint64) (res *FilscanTipSetResult, err error) {
	q := bson.M{"height": height}
	var result []*FilscanTipSetResult
	err = FindAll(tipSetCollection, q, nil, &result)
	if err != nil {
		return
	}
	if len(result) > 0 {
		return result[0], nil
	}
	return
}
func GetMaxTipSet() (res *FilscanTipSetResult, err error) {
	q := bson.M{}
	var result []*FilscanTipSetResult
	err = FindSortLimit(tipSetCollection, "-height", q, nil, &result, 0, 1)
	if err != nil {
		return
	}
	if len(result) > 0 {
		return result[0], nil
	}
	return
}

func GetTipsetCount() (total int, err error) {
	q := bson.M{}
	return FindCount(tipSetCollection, q, nil)
}

func GetTipsetCountByMinCreat(MinTime int64) (total int, err error) {
	q := bson.M{"gmt_create": bson.M{"$gte": MinTime}}
	return FindCount(tipSetCollection, q, nil)
}

/**
db.tipset.find({"cids./":"bafy2bzaceajqqryadk2jbmywvmmwtvdwrh43xpbzzq4muve3bigz6dikr5nkw"})
*/
func GetTipSetByBlockCid(block_cid string) (res *FilscanTipSetResult, err error) {
	if len(block_cid) < 1 {
		return
	}
	q := bson.M{"cids./": block_cid}
	var tipsets []*FilscanTipSetResult
	err = FindAll(tipSetCollection, q, nil, &tipsets)
	if err != nil {
		return
	} else {
		if len(tipsets) > 0 {
			return tipsets[0], nil
		}
	}
	return
}

func ThanHeightCount(height uint64) (than int, err error) {
	q := bson.M{"height": bson.M{"$gte": height}}
	return FindCount(tipSetCollection, q, nil)
}

func GetTipsetCountByStartEndTime(start, end int64) (num int, err error) {
	q := bson.M{"mine_time": bson.M{"$gte": start, "$lt": end}}
	num, err = FindCount(tipSetCollection, q, nil)
	if err != nil {
		return
	}
	return
}

type Heights struct {
	Height uint64 `json:"height"`
}

func GetAllTipsetHeight() (heights []*Heights, err error) {
	q := bson.M{}
	err = FindAll(tipSetCollection, q, nil, &heights)
	return
}
