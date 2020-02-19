package models

import (
	"encoding/json"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/globalsign/mgo"
	"gopkg.in/mgo.v2/bson"
	"sync"
)

// Include  StateListActors interface data（Miner） 、produce Msg wallet
type Account struct {
	Address string       `bson:"address" json:"address"`
	Actor   *types.Actor `bson:"actor" json:"actor"`

	IsStorageMiner bool `bson:"is_storage_miner" json:"is_storage_miner"`
	IsOwner        bool `bson:"is_owner" json:"is_owner"`

	IsMiner  bool `bson:"is_miner" json:"is_miner"`
	IsWallet bool `bson:"is_wallet" json:"is_wallet"`
	//todo
	GmtCreate   int64 `bson:"gmt_create" json:"gmt_create"`
	GmtModified int64 `bson:"gmt_modified" json:"gmt_modified"`
}

type AccountResult struct {
	Address        string       `bson:"address" json:"address"`
	Actor          FilscanActor `bson:"actor" json:"actor"`
	IsStorageMiner bool         `bson:"is_storage_miner" json:"is_storage_miner"`
	IsOwner        bool         `bson:"is_owner" json:"is_owner"`

	IsMiner     bool  `bson:"is_miner" json:"is_miner"`
	IsWallet    bool  `bson:"is_wallet" json:"is_wallet"`
	GmtCreate   int64 `bson:"gmt_create" json:"gmt_create"`
	GmtModified int64 `bson:"gmt_modified" json:"gmt_modified"`
}

type FilscanActor struct {
	Code    FilscanCid `bson:"Code" json:"Code"`
	Head    FilscanCid `bson:"Head" json:"Head"`
	Nonce   uint64     `bson:"Nonce" json:"Nonce"`
	Balance string     `bson:"Balance" json:"Balance"`
}

var isOwnerCash sync.Map
var isMinerCash sync.Map

const (
	AccountCollection = "account"
)

func Create_account_index() {
	ms, c := Connect(AccountCollection)
	defer ms.Close()
	ms.SetMode(mgo.Monotonic, true)

	indexs := []mgo.Index{
		//{Key: []string{"address"}, Unique: false, Background: true},
		{Key: []string{"address"}, Unique: true, Background: true},
	}
	for _, index := range indexs {
		if err := c.EnsureIndex(index); err != nil {
			panic(err)
		}
	}
}

func UpsertAccount(account *Account) (err error) {
	account.GmtCreate = TimeNow
	account.GmtModified = TimeNow
	tbyte, _ := json.Marshal(account)
	var p interface{}
	err = json.Unmarshal(tbyte, &p)
	if err != nil {
		return err
	}
	selector := bson.M{"address": account.Address}
	_, err = Upsert(AccountCollection, selector, p)
	return
}

func InsertAccount(account *Account) (err error) {
	account.GmtCreate = TimeNow
	account.GmtModified = TimeNow
	tbyte, _ := json.Marshal(account)
	var p interface{}
	err = json.Unmarshal(tbyte, &p)
	if err != nil {
		return err
	}
	err = Insert(AccountCollection, p)
	return
}

func UpdateAccount(account *AccountResult) (err error) {
	account.GmtCreate = account.GmtCreate
	account.GmtModified = TimeNow
	tbyte, _ := json.Marshal(account)
	var p interface{}
	err = json.Unmarshal(tbyte, &p)
	if err != nil {
		return err
	}
	selector := bson.M{"address": account.Address}
	_, err = Upsert(AccountCollection, selector, p) //== update
	return
}

func UpsertAccountArr(accounts []*Account) (err error) {
	if len(accounts) == 0 {
		return
	}
	for _, account := range accounts {
		q := bson.M{"address": account.Address}
		var res []*AccountResult
		err = FindAll(AccountCollection, q, nil, &res)
		if err != nil {
			return
		}
		if len(res) < 1 {
			err = InsertAccount(account)
			if err != nil {
				return
			}
		} else {
			if account.Actor == nil {
				continue
			}
			res[0].Actor.Balance = account.Actor.Balance.String()
			res[0].Actor.Code.Str = account.Actor.Code.String()
			res[0].Actor.Nonce = account.Actor.Nonce
			res[0].Actor.Head.Str = account.Actor.Head.String()
			res[0].IsOwner = account.IsOwner
			res[0].IsWallet = account.IsWallet
			res[0].IsMiner = account.IsMiner
			res[0].IsStorageMiner = account.IsStorageMiner
			err = UpdateAccount(res[0])
			if err != nil {
				return
			}
		}
	}
	return
}

/**
db.account.find().collation({
    "locale": "zh",
    numericOrdering: true
}).sort({
    "actor.Balance":  1
})
*/
func GetAccountBySort(begindex, count int) (res []*AccountResult, total int, err error) {
	sort := "-actor.Balance"
	q := bson.M{}
	collation := new(mgo.Collation)
	collation.Locale = "zh"
	collation.NumericOrdering = true
	err = FindSortCollationLimit(AccountCollection, sort, q, nil, &res, begindex, count, collation)
	if err != nil {
		return
	}
	total, err = FindCount(AccountCollection, q, nil)
	if err != nil {
		return nil, 0, err
	}
	return
}

func GetAccountSumBalance() (total float64, err error) {
	o0 := bson.M{"$match": bson.M{"actor.Balance": bson.M{"$ne": ""}}}
	o1 := bson.M{"$group": bson.M{"_id": "", "totalBalance": bson.M{"$sum": bson.M{"$toDouble": "$actor.Balance"}}}}
	operations := []bson.M{o0, o1}
	type result struct {
		Id           bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
		TotalBalance float64       `json:"totalBalance,omitempty" bson:"totalBalance,omitempty"`
	}
	var res []result
	err = AggregateAll(AccountCollection, operations, &res)
	if err != nil {
		return 0, err
	}
	if len(res) > 0 {
		return res[0].TotalBalance, nil
	} else {
		return 0, nil
	}
	return
}

func GetActorByAddress(address string) (res *AccountResult, err error) {
	if len(address) < 1 {
		return
	}
	q := bson.M{"address": address}
	var list []*AccountResult
	err = FindAll(AccountCollection, q, nil, &list)
	if err != nil {
		return
	}
	if len(list) > 0 {
		return list[0], nil
	} else {
		return nil, nil
	}
}

func UpdateIsOwnerByAdress(adress string) error {
	_, ok := isOwnerCash.Load(adress)
	if ok {
		return nil
	}
	q := bson.M{"address": adress}
	u := bson.M{"$set": bson.M{"is_owner": true}}
	err := Update(AccountCollection, q, u)
	if err != nil && err.Error() != "not found" {
		return err
	} else {
		isOwnerCash.Store(adress, 1)
	}
	return nil
}

func UpdateIsMinerByAdress(adress string) error {
	_, ok := isMinerCash.Load(adress)
	if ok {
		return nil
	}
	q := bson.M{"address": adress}
	u := bson.M{"$set": bson.M{"is_miner": true}}
	err := Update(AccountCollection, q, u)
	if err != nil && err.Error() != "not found" {
		return err
	} else {
		isMinerCash.Store(adress, 1)
	}
	return nil
}

//func init(){
//	go func() {
//		time.Sleep(1 * time.Second)
//		res ,err := GetAccountBySort(0,100)
//		if err != nil {
//			fmt.Println("err = %v",err)
//		}
//		fmt.Sprintln(res)
//
//	}()
//}
