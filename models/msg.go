package models

import (
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/globalsign/mgo"
	"gopkg.in/mgo.v2/bson"
	"strconv"
	"time"
)

const (
	MsgCollection = "Msg"
)

type FilscanMsg struct {
	Message       types.Message `bson:"message" json:"message"`
	Cid           string        `bson:"cid" json:"cid"`
	BlockCid      string        `bson:"block_cid" json:"block_cid"`
	ActorName     string        `bson:"actor_name" json:"actor_name"`
	MethodName    string        `bson:"method_name" json:"method_name"`
	ExitCode      string        `bson:"exit_code" json:"exit_code"` //默认值 不应为 0
	Return        string        `bson:"return" json:"return"`
	GasUsed       string        `bson:"gas_used" json:"gas_used"`
	RequiredFunds types.BigInt  `bson:"required_funds" json:"required_funds"`
	Size          int64         `bson:"size" json:"size"`
	Height        uint64        `bson:"height" json:"height"`
	MsgCreate     uint64        `bson:"msg_create" json:"msg_create"`
	GmtCreate     int64         `bson:"gmt_create" json:"gmt_create"`
	GmtModified   int64         `bson:"gmt_modified" json:"gmt_modified"`
}

type FilscanMsgResult struct {
	Message       FilscanResMsgMessage `bson:"message" json:"message"`
	Cid           string               `bson:"cid" json:"cid"`
	BlockCid      string               `bson:"block_cid" json:"block_cid"`
	MethodName    string               `bson:"method_name" json:"method_name"`
	ExitCode      string               `bson:"exit_code" json:"exit_code"`
	Return        string               `bson:"return" json:"return"`
	GasUsed       string               `bson:"gas_used" json:"gas_used"`
	RequiredFunds string               `bson:"required_funds" json:"required_funds"`
	Size          int64                `bson:"size" json:"size"`
	Height        uint64               `bson:"height" json:"height"`
	MsgCreate     uint64               `bson:"msg_create" json:"msg_create"`
	GmtCreate     int64                `bson:"gmt_create" json:"gmt_create"`
	GmtModified   int64                `bson:"gmt_modified" json:"gmt_modified"`
}

type FilscanResMsgMessage struct {
	To       string `bson:"To" json:"To"`
	From     string `bson:"From" json:"From"`
	Nonce    uint64 `bson:"Nonce" json:"Nonce"`
	Value    string `bson:"Value" json:"Value"`
	GasPrice string `bson:"GasPrice" json:"GasPrice"`
	GasLimit string `bson:"GasLimit" json:"GasLimit"`
	Method   int    `bson:"Method" json:"Method"`
	Params   string `bson:"Params" json:"Params"`
}

/**
db.Msg.ensureIndex({"message.Method":1})
db.Msg.ensureIndex({"cid":1},{"unique":true})
db.Msg.ensureIndex({"message.From":1})
db.Msg.ensureIndex({"message.To":1})
db.Msg.ensureIndex({"block_cid":1})
db.Msg.ensureIndex({"msg_create":-1})
db.Msg.ensureIndex({"method_name":-1})
db.Msg.ensureIndex({"message.From":1,"message.To":1})
*/
func Create_msg_index() {
	ms, c := Connect(MsgCollection)
	defer ms.Close()
	ms.SetMode(mgo.Monotonic, true)

	indexs := []mgo.Index{
		//{Key: []string{"cid"}, Unique: true, Background: true},
		{Key: []string{"cid"}, Unique: false, Background: true},
		{Key: []string{"message.Method"}, Unique: false, Background: true},
		//{Key: []string{"message.From"}, Unique: false, Background: true},
		{Key: []string{"message.To"}, Unique: false, Background: true},
		{Key: []string{"block_cid"}, Unique: false, Background: true},
		{Key: []string{"msg_create"}, Unique: false, Background: true},
		{Key: []string{"method_name"}, Unique: false, Background: true},
		{Key: []string{"message.From", "message.To"}, Unique: false, Background: true},
	}
	for _, index := range indexs {
		if err := c.EnsureIndex(index); err != nil {
			panic(err)
		}
	}
}

func InsertFilscanMsg(m *FilscanMsg) error {
	m.GmtCreate = TimeNow
	m.GmtModified = TimeNow
	tbyte, _ := json.Marshal(m)
	var p interface{}
	err := json.Unmarshal(tbyte, &p)
	if err != nil {
		return err
	}
	return Insert(MsgCollection, p)
}

func UpsertFilscanMsgMulti(m []*FilscanMsg) error {
	upsert_pairs := make([]interface{}, len(m)*2)
	for key, value := range m {
		value.GmtCreate = TimeNow
		value.GmtModified = TimeNow
		tbyte, _ := json.Marshal(value)
		var p interface{}
		err := json.Unmarshal(tbyte, &p)
		if err != nil {
			fmt.Sprintf("InsertFilscanMsgMulti Unmarshal err=%v", err)
			continue
		}
		//docs = append(docs, p)
		q := bson.M{"cid": value.Cid}
		upsert_pairs[key*2] = q
		upsert_pairs[key*2+1] = p
	}
	_, err := BulkUpsert(nil, MsgCollection, upsert_pairs)
	if err != nil {
		fmt.Sprintf("InsertFilscanMsgMulti BulkUpdate err = %v", err.Error())
	}
	return err
}

func GetMsgByMsgCid(msgCid string) (res []*FilscanMsgResult, err error) {
	q := bson.M{"cid": msgCid}
	err = FindAll(MsgCollection, q, nil, &res)
	return
}

func GetMsgByMsgCidSli(msgCid []string) (res []FilscanMsgResult, err error) {
	q := bson.M{"cid": bson.M{"$in": msgCid}}
	err = FindOne(MsgCollection, q, nil, &res)
	return
}

func GetMsgByMsgCidSliMethodLimit(msgCid []string, method string, begindex, count int) (res []FilscanMsgResult, total int, err error) {
	var q bson.M
	if len(method) != 0 { //search part
		q = bson.M{"cid": bson.M{"$in": msgCid}, "message.Method": method}
	} else { //search all
		q = bson.M{"cid": bson.M{"$in": msgCid}}
	}
	err = FindAllLimit(MsgCollection, q, nil, &res, begindex, count)
	if err != nil {
		total, err = FindCount(MsgCollection, q, nil)
	}
	return
}

func GetMsgByBlockMethodNameLimit(block string, methodName string, begindex, count int) (res []*FilscanMsgResult, total int, err error) {
	q := bson.M{}
	//q := bson.M{"msg_create":bson.M{"$gt":TimeNow - 60*60*24 *7}}
	if len(block) != 0 { //search part
		q["block_cid"] = block
	} else {
		q["msg_create"] = bson.M{"$gt": TimeNow - 60*60*24*7}
	}
	if len(methodName) != 0 {
		q["method_name"] = methodName
		//q = bson.M{"message.Method":method}
	}
	if count > 0 {
		err = FindAllLimit(MsgCollection, q, nil, &res, begindex, count)
		if err != nil {
			return nil, 0, err
		}
	}
	total, err = FindCount(MsgCollection, q, nil)
	if err != nil {
		return nil, 0, err
	}
	return
}

func GetMsgByAddressFromMethodLimit(address, fromTo, methodName string, begindex, count int) (res []*FilscanMsgResult, total int, err error) {
	if len(address) < 1 {
		return
	}
	q := bson.M{}
	switch fromTo {
	case "from":
		q["message.From"] = address
	case "to":
		q["message.To"] = address
	default:
		bm := []bson.M{}
		q["$or"] = append(bm, bson.M{"message.To": address}, bson.M{"message.From": address})
	}
	if len(methodName) != 0 {
		q["method_name"] = methodName
		//q = bson.M{"message.Method":method}
	}
	if count > 0 {
		//err = FindAllLimit(MsgCollection, q, nil, &res, begindex, count)
		err = FindSortLimit(MsgCollection, "-msg_create", q, nil, &res, begindex, count)
		if err != nil {
			return nil, 0, err
		}
	}
	total, err = FindCount(MsgCollection, q, nil)
	if err != nil {
		return nil, 0, err
	}
	return
}

func GetMsgByAddressFromToMethodNameCount(address, fromTo, methodName string) (total int, err error) {
	if len(address) < 1 {
		return
	}
	q := bson.M{}
	switch fromTo {
	case "from":
		q["message.From"] = address
	case "to":
		q["message.To"] = address
	default:
		bm := []bson.M{}
		q["$or"] = append(bm, bson.M{"message.To": address}, bson.M{"message.From": address})
	}
	if len(methodName) != 0 {
		q["method_name"] = methodName
		//q = bson.M{"message.Method":method}
	}
	total, err = FindCount(MsgCollection, q, nil)
	if err != nil {
		return 0, err
	}
	return
}

//根据 addArr 分别获取每个人的 message数量  todo
//func GetMsgByAddressArrFromToMethodCount(address []string, fromTo, methodName string) (total int, err error) {
//	if len(address) < 1 {
//		return
//	}
//	q := bson.M{}
//	switch fromTo {
//	case "from":
//		q["message.From"] = bson.M{"$in": address}
//	case "to":
//		q["message.To"] = bson.M{"$in": address}
//	default:
//		bm := []bson.M{}
//		q["$or"] = append(bm, bson.M{"message.To": bson.M{"$in": address}}, bson.M{"message.From": bson.M{"$in": address}})
//	}
//	if len(method) != 0 {
//		q["method_name"] = methodName
//		//q = bson.M{"message.Method":method}
//	}
//
//	o1 := bson.M{"$group": bson.M{"_id": "", "totalGasPrice": bson.M{"$sum": bson.M{"$toDouble": "$message.GasPrice"}}}}
//	operations := []bson.M{o1}
//	type result struct {
//		Id            bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
//		TotalGasPrice float64       `json:"totalGasPrice,omitempty" bson:"totalGasPrice,omitempty"`
//	}
//	var res []result
//	err = AggregateAll(MsgCollection, operations, &res)
//	if err != nil {
//		return 0, err
//	}
//	if len(res) > 0 {
//		//return res[0].TotalGasPrice, nil
//	} else {
//		return 0, nil
//	}
//
//	total, err = FindCount(MsgCollection, q, nil)
//	if err != nil {
//		return 0, err
//	}
//	return
//}

/*func GetMsgByBlockCid(blockCid string)(res []FilscanMsgResult,err error) {
	q := bson.M{"block_cid":blockCid}
	err = FindOne(MsgCollection,q,nil,&res)
	return
}*/
func GetMsgMethodName(blockCids []string) (res []string, err error) {
	q := bson.M{}
	if len(blockCids) > 0 && len(blockCids[0]) > 0 {
		q["block_cid"] = bson.M{"$in": blockCids}
	}
	//err = Distinct(MsgCollection, "message.Method", q, &res)
	err = Distinct(MsgCollection, "method_name", q, &res)
	return
}

func GetMsgLatestList(num int) (res []*FilscanMsgResult, err error) {
	err = FindSortLimit(MsgCollection, "-msg_create", nil, nil, &res, 0, num)
	return res, err
}

func GetSumGasPrice() (sum float64, err error) {
	o1 := bson.M{"$group": bson.M{"_id": "", "totalGasPrice": bson.M{"$sum": bson.M{"$toDouble": "$message.GasPrice"}}}}
	operations := []bson.M{o1}
	type result struct {
		Id            bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
		TotalGasPrice float64       `json:"totalGasPrice,omitempty" bson:"totalGasPrice,omitempty"`
	}
	var res []result
	err = AggregateAll(MsgCollection, operations, &res)
	if err != nil {
		return 0, err
	}
	if len(res) > 0 {
		return res[0].TotalGasPrice, nil
	} else {
		return 0, nil
	}
}

func GetSumSize() (sum uint64, err error) {
	o1 := bson.M{"$group": bson.M{"_id": "", "totalSize": bson.M{"$sum": "$size"}}}
	operations := []bson.M{o1}
	type result struct {
		Id        bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
		TotalSize uint64        `json:"totalSize,omitempty" bson:"totalSize,omitempty"`
	}
	var res result
	err = AggregateOne(MsgCollection, operations, &res)
	if err != nil {
		return 0, err
	}
	return res.TotalSize, nil
}

func GetMsgCount() (total int, err error) {
	q := bson.M{}
	return FindCount(MsgCollection, q, nil)
}

func GetSumGasPriceByMsgMinCreat(MinTime int64) (sum float64, err error) {
	o1 := bson.M{"$group": bson.M{"_id": "", "totalGasPrice": bson.M{"$sum": bson.M{"$toDouble": "$message.GasPrice"}}}}
	o0 := bson.M{"$match": bson.M{"msg_create": bson.M{"$gte": MinTime}}}
	operations := []bson.M{o0, o1}
	type result struct {
		Id            bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
		TotalGasPrice float64       `json:"totalGasPrice,omitempty" bson:"totalGasPrice,omitempty"`
	}
	var res []result
	err = AggregateAll(MsgCollection, operations, &res)
	if err != nil {
		return 0, err
	}
	if len(res) > 0 {
		return res[0].TotalGasPrice, nil
	} else {
		return 0, nil
	}
}

func GetSumSizeByMsgMinCreat(MinTime int64) (sum uint64, err error) {
	o1 := bson.M{"$group": bson.M{"_id": "", "totalSize": bson.M{"$sum": "$size"}}}
	o0 := bson.M{"$match": bson.M{"msg_create": bson.M{"$gte": MinTime}}}
	operations := []bson.M{o0, o1}
	type result struct {
		Id        bson.ObjectId `json:"_id,omitempty" bson:"_id,omitempty"`
		TotalSize uint64        `json:"totalSize,omitempty" bson:"totalSize,omitempty"`
	}
	var res []result
	err = AggregateAll(MsgCollection, operations, &res)
	if err != nil {
		return 0, err
	}
	if len(res) > 0 {
		return res[0].TotalSize, nil
	} else {
		return 0, nil
	}
}

func GetMsgCountByMsgMinCreat(MinTime int64) (total int, err error) {
	q := bson.M{"msg_create": bson.M{"$gte": MinTime}}
	return FindCount(MsgCollection, q, nil)
}

type MsgReceipt struct {
	ExitCode int    `bson:"ExitCode" json:"ExitCode"` //默认值 不应为 0
	Return   string `bson:"Return" json:"Return"`
	GasUsed  string `bson:"GasUsed" json:"GasUsed"`
}

func UpdateMsgReceipts(msg []api.Message, msgReceipt []*types.MessageReceipt, loop int) {
	if loop > 1<<6 {
		return
	}
	if len(msg) != len(msgReceipt) {
		return
	}
	if len(msg) == 0 {
		return
	}
	//go func(m []api.Message, mr []*types.MessageReceipt,l int) {
	time.Sleep(1 * time.Second)
	upsert_pairs := make([]interface{}, len(msg)*2)
	oneQ := bson.M{}
	oneU := bson.M{}
	for key, value := range msg {

		q := bson.M{"cid": value.Cid.String()}
		rebyte, _ := json.Marshal(msgReceipt[key])
		var returnS MsgReceipt
		err := json.Unmarshal(rebyte, &returnS)
		if err != nil {
			fmt.Sprintf("err =%v", err)
		}
		//u := bson.M{"$set": bson.M{"exit_code": strconv.Itoa(returnS.ExitCode), "return": returnS.Return, "gas_used": returnS.GasUsed}}
		//err = Update(MsgCollection, q, u) //一个个update  todo
		//if err != nil && err == mgo.ErrNotFound {
		//	time.Sleep(1 * time.Second)
		//	UpdateMsgReceipts(m, mr,l) //如果第一个不存在  其他的也将不存在  直接交由自身  知道第一个存在 即其他也存在，
		//}
		if key == 0 {
			oneQ = bson.M{"cid": value.Cid.String()}
			oneU = bson.M{"$set": bson.M{"exit_code": string(returnS.ExitCode), "return": returnS.Return, "gas_used": returnS.GasUsed}}
		}
		if err != nil {
			fmt.Sprintf("UpdateMsgReceipts err = %v", err.Error())
			continue
		}
		upsert_pairs[key*2] = q
		code := strconv.Itoa(returnS.ExitCode)
		upsert_pairs[key*2+1] = bson.M{"$set": bson.M{"exit_code": code, "return": returnS.Return, "gas_used": returnS.GasUsed}}
	}
	err := Update(MsgCollection, oneQ, oneU) //先 拿第一个去试试 数据库有没有   由于是批量操作 没有的话可认为 同tipset下 其他msg也没有
	if err != nil && err == mgo.ErrNotFound {
		time.Sleep(5 * time.Second)
		UpdateMsgReceipts(msg, msgReceipt, loop) //如果第一个不存在  其他的也将不存在  直接交由自身  知道第一个存在 即其他也存在，
		return
	}
	_, err = BulkUpdate(MsgCollection, upsert_pairs)
	if err != nil {
		fmt.Sprintf("BulkUpdate err = %v", err.Error())
	}
	//}(Msg, msgReceipt,loop)
}

func GetDistinctFromAddressByTime(startTime, endTime int64) (res []string, err error) {
	q := bson.M{"gmt_create": bson.M{"$gte": startTime, "$lte": endTime}}
	//err = Distinct(MsgCollection, "message.Method", q, &res)
	err = Distinct(MsgCollection, "message.From", q, &res)
	return
}

func GetDistinctToAddressByTime(startTime, endTime int64) (res []string, err error) {
	q := bson.M{"gmt_create": bson.M{"$gte": startTime, "$lte": endTime}}
	//err = Distinct(MsgCollection, "message.Method", q, &res)
	err = Distinct(MsgCollection, "message.To", q, &res)
	return
}
