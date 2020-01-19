package models

import (
	"encoding/json"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/globalsign/mgo"
	"github.com/ipfs/go-cid"
	"gopkg.in/mgo.v2/bson"
)

type FilscanBlock struct {
	Cid         string             `bson:"cid" json:"cid"`
	BlockHeader *types.BlockHeader `bson:"block_header" json:"block_header"`
	MsgCids     []cid.Cid          `bson:"msg_cids" json:"msg_cids"`
	BlockReward float64            `bson:"block_reword" json:"block_reword"`
	Size        int64              `bson:"size" json:"size"`
	IsMaster    int                `bson:"is_master" json:"is_master"`
	GmtCreate   int64              `bson:"gmt_create" json:"gmt_create"`
	GmtModified int64              `bson:"gmt_modified" json:"gmt_modified"`
}
type FilscanBlockResult struct {
	Cid         string       `bson:"cid" json:"cid"`
	BlockHeader BlockHeader  `bson:"block_header" json:"block_header"`
	MsgCids     []FilscanCid `bson:"msg_cids" json:"msg_cids"`
	BlockReword float64      `bson:"block_reword" json:"block_reword"`
	Size        int64        `bson:"size" json:"size"`
	IsMaster    int          `bson:"is_master" json:"is_master"`
	GmtCreate   int64        `bson:"gmt_create" json:"gmt_create"`
	GmtModified int64        `bson:"gmt_modified" json:"gmt_modified"`
}

type BlockHeader struct {
	Miner string `bson:"Miner" json:"Miner"`

	Ticket BlockHeaderTicket `bson:"Ticket" json:"Ticket"`

	ElectionProof string `bson:"ElectionProof" json:"ElectionProof"`

	Parents []FilscanCid `bson:"Parents" json:"Parents"`

	ParentWeight string `bson:"ParentWeight" json:"ParentWeight"`

	Height uint64 `bson:"Height" json:"Height"`

	ParentStateRoot FilscanCid `bson:"ParentStateRoot" json:"ParentStateRoot"`

	ParentMessageReceipts FilscanCid `bson:"ParentMessageReceipts" json:"ParentMessageReceipts"`

	Messages FilscanCid `bson:"Messages" json:"Messages"`

	BLSAggregate BlockHeaderSignature `bson:"BLSAggregate" json:"BLSAggregate"`

	Timestamp uint64 `bson:"Timestamp" json:"Timestamp"`

	BlockSig BlockHeaderSignature `bson:"BlockSig" json:"BlockSig"`
}
type FilscanCid struct {
	Str string `bson:"/" json:"/"`
}

type BlockHeaderTicket struct {
	VRFProof string `bson:"VRFProof" json:"VRFProof"`
}
type BlockHeaderSignature struct {
	Type string `bson:"Type" json:"Type"`
	Data string `bson:"Data" json:"Data"`
}

const (
	BlocksCollection = "block"
)

func Create_block_index() {
	ms, c := Connect(BlocksCollection)
	defer ms.Close()
	ms.SetMode(mgo.Monotonic, true)

	indexs := []mgo.Index{
		{Key: []string{"cid"}, Unique: true, Background: true},
		//{Key: []string{"cid"}, Unique: false, Background: true},
		{Key: []string{"block_header.Height"}, Unique: false, Background: true},
		{Key: []string{"block_header.Timestamp"}, Unique: false, Background: true},
	}
	for _, index := range indexs {
		if err := c.EnsureIndex(index); err != nil {
			panic(err)
		}
	}
}

func UpsertFilscanBlock(fb *FilscanBlock) error {
	fb.GmtCreate = TimeNow
	fb.GmtModified = TimeNow
	tbyte, _ := json.Marshal(fb)
	var p interface{}
	err := json.Unmarshal(tbyte, &p)
	if err != nil {
		return err
	}
	return Insert(BlocksCollection, p)
}

func InsertFilscanBlock(fb *FilscanBlock) (err error) {
	fb.GmtCreate = TimeNow
	fb.GmtModified = TimeNow
	tbyte, _ := json.Marshal(fb)
	var p interface{}
	err = json.Unmarshal(tbyte, &p)
	if err != nil {
		return err
	}
	s := bson.M{"cid": fb.Cid}
	_, err = Upsert(BlocksCollection, s, p)
	return
}

func QueryBlockNum(cids []cid.Cid) (num int, err error) {
	var cidList []string
	for _, value := range cids {
		cidList = append(cidList, value.String())
	}
	q := bson.M{"cid": bson.M{"$in": cidList}}
	return FindCount(BlocksCollection, q, nil)
}

func GetBlockByCid(cids []string) (res []FilscanBlockResult, err error) {
	q := bson.M{"cid": bson.M{"$in": cids}}
	//var r []interface{}
	err = FindAll(BlocksCollection, q, nil, &res)
	//err = FindAll(BlocksCollection, q, nil, &r)
	return
}

//func GetOneBlock(cid string) (res FilscanBlockResult, err error) {
//	q := bson.M{"cid": cid}
//	err = FindOne(BlocksCollection, q, nil, &res)
//	return
//}

func GetBlockByHeight(height uint64) (res []FilscanBlockResult, err error) {
	q := bson.M{"block_header.Height": height}
	err = FindAll(BlocksCollection, q, nil, &res)
	return
}
func GetBlockByTime(startTime, endTime int64) (res []*FilscanBlockResult, err error) {
	q := bson.M{"block_header.Timestamp": bson.M{"$gte": startTime, "$lt": endTime}}
	err = FindAll(BlocksCollection, q, nil, &res)
	return
}

func GetBlockCountByTime(startTime, endTime int64) (count int, err error) {
	q := bson.M{"block_header.Timestamp": bson.M{"$gte": startTime, "$lt": endTime}}
	return FindCount(BlocksCollection, q, nil)
}

func GetBlockSumSizeByTime(startTime, endTime int64) (sum int, err error) {
	o0 := bson.M{"$match": bson.M{"block_header.Timestamp": bson.M{"$gte": startTime, "$lt": endTime}}}
	o1 := bson.M{"$group": bson.M{"_id": "", "totalSize": bson.M{"$sum": "$size"}}}

	operations := []bson.M{o0, o1}
	type result struct {
		Id        bson.ObjectId `json:"_id,omitempty" bson:"_id,omitempty"`
		TotalSize int           `json:"totalSize,omitempty" bson:"totalSize,omitempty"`
	}
	var res []result
	err = AggregateAll(BlocksCollection, operations, &res)
	if err != nil {
		return 0, err
	}
	if len(res) > 0 {
		return res[0].TotalSize, nil
	} else {
		return 0, nil
	}
}

func AllBlockNum() (num int, err error) {
	return FindCount(BlocksCollection, nil, nil)
}

func GetLatestBlockList(num int) (res []*FilscanBlockResult, err error) {
	err = FindSortLimit(BlocksCollection, "-block_header.Height", nil, nil, &res, 0, num)
	return res, err
}

func GetBlockListByMiner(minerArr []string, begindex, count int) (res []*FilscanBlockResult, total int, err error) {
	q := bson.M{"block_header.Miner": bson.M{"$in": minerArr}}
	if count > 0 {
		err = FindSortLimit(BlocksCollection, "-block_header.Height", q, nil, &res, begindex, count)
		if err != nil {
			return nil, 0, err
		}
	}
	total, err = FindCount(BlocksCollection, q, nil)
	if err != nil {
		return nil, 0, err
	}

	return res, total, err
}

/*func GetBlockMsgs(blockCid  string)(res []FilscanBlockResult , err error){
	q := bson.M{"$lookup":bson.M{
		"from":"msg",
		"localField":"msg_cids./",
		"foreignField":
	} }
	//"cid":blockCid,bson.M{
}*/
/*
type FilscanBlock_01 struct {
	BlockHeader struct {
		Height  int64       `bson:"Height" json:"Height"`
		Parents interface{} `bson:"Parents" json:"Parents"`
	} `bson:"block_header" json:"block_header"`
	Size int64 `bson:"size" json:"size"`
}

func GetBlockByHeight() (err error) {
	q := bson.M{"cid": "bafy2bzaced4sstkwxreoqli6mr3tu56wpv4cnzrcnt3iytpsugt6isiyd7kou"}
	var res FilscanBlock_01
	//v := res["weight"].(string)
	err = FindOne(BlocksCollection, q, nil, &res)
	return
}


}
func init() {
	go func() {
		time.Sleep(time.Second * 1)
		GetOneBlockByCid("bafy2bzacecwkrykav6zimfcuypjhfcmp2tsnzt5tulpsww5mngb7x47p6m3wy")
	}()
}*/
