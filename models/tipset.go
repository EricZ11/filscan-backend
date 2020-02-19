package models

import (
	"encoding/json"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/globalsign/mgo"
	"github.com/ipfs/go-cid"
	"gopkg.in/mgo.v2/bson"
)

type FilscanTipSet struct {
	Key          string       `bson:"key" json:"key"`
	ParentKey    string       `bson:"parent_key" json:"parent_key"`
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
	TipSetCollection = "tipset"
)

/**

db.Tipset.ensureIndex({"height":-1})
db.Tipset.ensureIndex({"gmt_create":-1})
db.Tipset.ensureIndex({"cids./":1})
*/

func Create_tipset_index() {
	ms, c := Connect(TipSetCollection)
	defer ms.Close()
	ms.SetMode(mgo.Monotonic, true)

	indexs := []mgo.Index{
		{Key: []string{"height"}, Unique: false, Background: true},
		{Key: []string{"key"}, Unique: false, Background: true},
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

	tips.Key = t.Key().String()
	tips.ParentKey = t.Parents().String()

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
	_, err = Upsert(TipSetCollection, s, p)
	return err
}

func GetTipSetByHeight(start, end uint64) (res []*FilscanTipSetResult, err error) {
	q := bson.M{"height": bson.M{"$gte": start, "$lte": end}}
	err = FindAll(TipSetCollection, q, nil, &res)
	return
}

func GetTipSetByOneHeight(height uint64) (res *FilscanTipSetResult, err error) {
	q := bson.M{"height": height}
	var result []*FilscanTipSetResult
	err = FindAll(TipSetCollection, q, nil, &result)
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
	err = FindSortLimit(TipSetCollection, "-height", q, nil, &result, 0, 1)
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
	return FindCount(TipSetCollection, q, nil)
}

func GetTipsetCountByMinCreat(MinTime int64) (total int, err error) {
	q := bson.M{"gmt_create": bson.M{"$gte": MinTime}}
	return FindCount(TipSetCollection, q, nil)
}
func GetTipsetCountByMinTime(MinTime int64) (total int, err error) {
	q := bson.M{"mine_time": bson.M{"$gte": MinTime}}
	return FindCount(TipSetCollection, q, nil)
}

/**
db.Tipset.find({"cids./":"bafy2bzaceajqqryadk2jbmywvmmwtvdwrh43xpbzzq4muve3bigz6dikr5nkw"})
*/
func GetTipSetByBlockCid(block_cid string) (res *FilscanTipSetResult, err error) {
	if len(block_cid) < 1 {
		return
	}
	q := bson.M{"cids./": block_cid}
	var tipsets []*FilscanTipSetResult
	err = FindAll(TipSetCollection, q, nil, &tipsets)
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
	return FindCount(TipSetCollection, q, nil)
}

func GetTipsetCountByStartEndTime(start, end int64) (num int, err error) {
	q := bson.M{"mine_time": bson.M{"$gte": start, "$lt": end}}
	num, err = FindCount(TipSetCollection, q, nil)
	if err != nil {
		return
	}
	return
}

type Heights struct {
	TipsetKey  string `josn:"key"`
	ParenteKey string `json:"parent_key"`
	Height     uint64 `json:"height"`
}

type Heights_list []*Heights

func (hl Heights_list) To_Height_map() map[uint64]*Heights {
	res := make(map[uint64]*Heights)
	for _, h := range hl {
		res[h.Height] = h
	}
	return res
}

func (h *Heights) IsTipset(tipset *types.TipSet) bool {
	return h.Height == tipset.Height() &&
		h.ParenteKey == tipset.Parents().String() &&
		h.TipsetKey == tipset.Key().String()
}

func GetTipset_with_height_range(head, foot uint64) (Heights_list, error) {
	var his []*Heights
	err := FindSortLimit(TipSetCollection, "-height",
		bson.M{"height": bson.M{"$lt": head, "$gte": foot}},
		bson.M{"height": 1, "key": 1, "parent_key": 1},
		&his, 0, int(head-foot+1))

	return his, err
}

func GetAllTipsetHeight() (heights []*Heights, err error) {
	q := bson.M{}
	err = FindAll(TipSetCollection, q, nil, &heights)
	return
}

func GetTipsetByTime(time int64) (tipset *FilscanTipSetResult, err error) {
	q := bson.M{"mine_time": bson.M{"$lte": time}}
	var res []*FilscanTipSetResult
	err = FindSortLimit(TipSetCollection, "-mine_time", q, nil, &res, 0, 1)
	if err != nil {
		return
	}
	if len(res) > 0 {
		tipset = res[0]
	}
	return
}
