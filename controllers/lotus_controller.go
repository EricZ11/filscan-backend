package controllers

import (
	"context"
	"encoding/json"
	"filscan_lotus/controllers/filscaner"
	"filscan_lotus/models"
	"filscan_lotus/utils"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"strconv"
	"strings"
	"time"
)

var SyncChain int64
var TipsetQueueSize int
var DBHaveTipset map[uint64]bool //数据库已有tipset map

func FirstSynLotus() {
	TipsetQueue = NewQueue()
	syncChain := conf("syncChain")
	SyncChain, _ = strconv.ParseInt(syncChain, 10, 64)
	tipsetQueueSize := conf("tipsetQueueSize")
	TipsetQueueSize, _ = strconv.Atoi(tipsetQueueSize)

	tipset, err := LotusApi.ChainHead(context.TODO())
	if err != nil {
		log("SynLotus ChainHead err = %v", err)
		return
	}
	if tipset.Height() < uint64(TipsetQueueSize) {
		log("tipset.Height< %v ,Retry after 300s ", TipsetQueueSize)
		time.Sleep(300 * time.Second)
		FirstSynLotus()
		return
	}
	t, err := LotusApi.ChainGetTipSetByHeight(context.TODO(), tipset.Height()-uint64(TipsetQueueSize), tipset) //
	//t, err := LotusApi.ChainGetTipSetByHeight(context.TODO(), 500, tipset) //
	if err != nil {
		log("ChainGetTipSetByHeight err = %v", err)
	}
	heights, _ := models.GetAllTipsetHeight()
	DBHaveTipset = make(map[uint64]bool)
	for _, value := range heights {
		DBHaveTipset[value.Height] = true
	}
	GetTipSetAdd(t, nil, nil)
	go func() {
		SynLotus()
	}()
}

func SynLotus() {
	//LastCycle = TimeNowStr
	tick := time.Tick(time.Duration(SyncChain) * time.Second)
	go func() {
		for {
			<-tick
			log("SynLotus ChainHead：%v", SyncChain)
			tipset, err := LotusApi.ChainHead(context.TODO())
			if err != nil {
				log("SynLotus ChainHead err = %v", err)
				return
			}
			GetTipSetPushQueue(tipset)
		}
	}()

	go func() { //start write mongo
		for {
			d := TipsetQueue.Size() - TipsetQueueSize
			if d > 0 {
				eList := TipsetQueue.GetHeaderList(d)
				for _, value := range eList {
					go func(e *Element) {
						SaveTipsetQueueE(e)
					}(value)
				}
			}
			time.Sleep(2 * time.Second)
		}
	}()
	//go func() {
	//	time.Sleep(3 * time.Second)
	//	tipset, err := LotusApi.ChainHead(context.TODO())
	//	if err != nil {
	//		log("SynLotus ChainHead err = %v", err)
	//		return
	//	}
	//	GetTipSetPushQueue(tipset)
	//
	//}()
}

func SaveTipsetQueueE(e *Element) error {
	bmList := e.blocks
	for _, value := range bmList {
		err := models.InsertFilscanBlock(value.block)
		if err != nil && len(err.Error()) > 6 && err.Error()[:6] != "E11000" {
			log("SynLotus InsertFilscanBlock err  = %v", err)
			log("block = %v ", value.block.Cid)
		}
		if len(value.msg) > 0 {
			err = models.UpsertFilscanMsgMulti(value.msg)
			if err != nil && len(err.Error()) > 6 && err.Error()[:6] != "E11000" {
				log("SynLotus InsertFilscanMsgMulti err  = %v", err)
				log("block = %v ", value.block.Cid)
			}
		}
	}
	err := models.AddTipSet(e.tipset)
	if err != nil {
		log("SynLotus AddTipSet err  = %v", err)
		return err
	}
	log("SaveTipsetQueue %v height=%v ,done", e.tipset.Cids(), e.tipset.Height())
	return nil
}

func GetTipSetAdd(tipset *types.TipSet, msg []api.Message, msgReceipt []*types.MessageReceipt) {
	if tipset == nil {
		return
	}
	//blocks := tipset.Blocks()
	//cids := tipset.Cids()
	//num, err := models.QueryBlockNum(cids)
	//if err != nil {
	//	log("SynLotus FindCidsCount err  = %v", err)
	//	log("cids = %v ", cids)
	//	return
	//}
	parentMessages, err := LotusApi.ChainGetParentMessages(context.TODO(), tipset.Blocks()[0].Cid())
	if err != nil {
		log("ChainGetParentMessages err = %v", err)
		return
	}
	//parent msg ChainGetParentReceipts
	parentReceipts, err := LotusApi.ChainGetParentReceipts(context.TODO(), tipset.Blocks()[0].Cid())
	if err != nil {
		log("ChainGetParentReceipts err = %v", err)
		return
	}
	_, ok := DBHaveTipset[tipset.Height()]
	if ok {
		log("Syn success tipset Height =%v ", tipset.Height())
		//return
		parents := tipset.Parents().Cids()
		go func(cid2 cid.Cid, msg []api.Message, msgReceipt []*types.MessageReceipt) {
			//t, err := LotusApi.ChainGetTipSet(context.TODO(), parents[0])
			if len(parents) < 1 {
				return
			}
			parent, _ := LotusApi.ChainGetBlock(context.TODO(), cid2)
			t, err := LotusApi.ChainGetTipSetByHeight(context.TODO(), parent.Height, tipset)
			if err != nil {
				log("SynLotus ChainGetTipSet err  = %v", err)
				return
			}
			time.Sleep(50 * time.Microsecond)
			GetTipSetAdd(t, msg, msgReceipt)
			return
		}(parents[0], parentMessages, parentReceipts)
		delete(DBHaveTipset, tipset.Height())
		return
	}

	// filscaner.FilscanerInst.NotifyHeaderChanged(store.HCApply, tipset)

	starttime := time.Now().Unix()
	defer func() {
		endtime := time.Now().Unix()
		timediff := float64(endtime - starttime)

		if timediff > 30 {
			filscaner.FilscanerInst.Printf("<><><><warning:::: in gettipsetadd(height=%d) block num  =(%d)  notify very slow..............\n", tipset.Height(), len(tipset.Blocks()))
		}

		filscaner.FilscanerInst.Printf("~~~~~~gettipsetadd height = %d, use time = %.3f(m)\n", tipset.Height(), (float64(endtime-starttime))/60)
	}()

	bmList, err := Tipset2BlocksMsg(tipset)
	if err != nil {
		return
	}
	for _, value := range bmList {
		err = models.InsertFilscanBlock(value.block)
		if err != nil && len(err.Error()) > 6 && err.Error()[:6] != "E11000" {
			log("SynLotus InsertFilscanBlock err  = %v", err)
			log("block = %v ", value.block.Cid)
		}
		if len(value.msg) > 0 {
			if msg != nil && msgReceipt != nil {
				for _, m := range value.msg {
					for key, msgV := range msg {
						if m.Cid == msgV.Cid.String() {
							rebyte, _ := json.Marshal(msgReceipt[key])
							var returnS models.MsgReceipt
							json.Unmarshal(rebyte, &returnS)
							m.ExitCode = strconv.Itoa(returnS.ExitCode)
							m.GasUsed = returnS.GasUsed
							m.Return = returnS.Return
						}
					}
				}
			}
			err = models.UpsertFilscanMsgMulti(value.msg)
			if err != nil && len(err.Error()) > 6 && err.Error()[:6] != "E11000" {
				log("SynLotus InsertFilscanMsgMulti err  = %v", err)
				log("block = %v ", value.block.Cid)
			}
		}
	}
	err = models.AddTipSet(tipset)
	if err != nil {
		log("SynLotus AddTipSet err  = %v", err)
		return
	}
	log("%v,height=%v ,done", tipset.Cids(), tipset.Height())
	parents := tipset.Parents().Cids()

	go func(cid2 cid.Cid, msg []api.Message, msgReceipt []*types.MessageReceipt) {
		//t, err := LotusApi.ChainGetTipSet(context.TODO(), parents[0])
		if len(parents) < 1 {
			return
		}
		parent, _ := LotusApi.ChainGetBlock(context.TODO(), cid2)
		t, err := LotusApi.ChainGetTipSetByHeight(context.TODO(), parent.Height, tipset)
		if err != nil {
			log("SynLotus ChainGetTipSet err  = %v", err)
			return
		}
		time.Sleep(300 * time.Microsecond)
		GetTipSetAdd(t, msg, msgReceipt)
		return
	}(parents[0], parentMessages, parentReceipts)
	return
}

/*func AccountUpdateInsert(msg types.Message) {
	to := msg.To
	from := msg.From
	var doubleAddress []address.Address
	if !to.Empty() && len(to.String()) > 2 {
		doubleAddress = append(doubleAddress, to)
	}
	if !from.Empty() && len(from.String()) > 2 {
		doubleAddress = append(doubleAddress, from)
	}
	var doubleAccount []*models.Account
	for _, address := range doubleAddress {
		ac := new(models.Account)
		ac.Address = address.String()
		actor, err := GetActorByAddress(address)
		if err != nil {
			ps("get GetActorByAddress failed, message:%s\n",
				address.String())
			return
		}
		ac.Actor = actor
		switch address.String()[0:2] {
		//case "t0":
		case "t3":
			ac.IsWallet = true
		}
		doubleAccount = append(doubleAccount, ac)
	}
	err := models.UpsertAccountArr(doubleAccount)
	if err != nil {
		ps("handle_upsert_account_message UpsertAccountArr failed, err:%s\n",
			err.Error())
		return
	}
}*/

func Tipset2BlocksMsg(tipset *types.TipSet) (bmList []*BlockAndMsg, err error) {
	bs := tipset.Blocks()
	if len(bs) < 1 {
		return
	}
	//var bmList []*BlockAndMsg
	for _, b := range bs {
		bm := new(BlockAndMsg)
		msg, err := LotusApi.ChainGetBlockMessages(context.TODO(), b.Cid())
		if err != nil {
			log("SynLotus ChainGetBlockMessages err  = %v", err)
			log("block = %v ", b.Cid())
			return nil, err
		}
		block := new(models.FilscanBlock)
		block.Cid = b.Cid().String()
		block.BlockHeader = b
		block.MsgCids = msg.Cids
		block.BlockReward = 0
		bbyte, _ := b.Serialize()
		block.Size = int64(len(bbyte))
		//blockList = append(blockList,block)
		bm.block = block
		if err != nil {
			log("SynLotus InsertFilscanBlock err  = %v", err)
			log("block = %v ", b.Cid())
			return nil, err
		}
		var msgList []*models.FilscanMsg
		for _, value := range msg.BlsMessages {
			m := new(models.FilscanMsg)
			m.Message = *value
			m.Cid = value.Cid().String()
			m.BlockCid = b.Cid().String()
			m.RequiredFunds = value.RequiredFunds()
			mbyte, _ := value.Serialize()
			m.Size = int64(len(mbyte))
			m.MsgCreate = b.Timestamp
			m.Height = b.Height
			if value.Method == 0 {
				m.MethodName = "Transfer"
				msgList = append(msgList, m)
				continue
			}
			_, method, err := filscanerInst.ParseActorMessage(value)
			if err != nil {
				log("filscanerInst.ParseActorMessage err =%v", err.Error())
			} else {
				m.MethodName = method.Name
			}
			msgList = append(msgList, m)
		}
		bm.msg = msgList
		bmList = append(bmList, bm)
	}
	return
}

func GetTipSetPushQueue(tipset *types.TipSet) {
	if tipset == nil {
		return
	}
	cids := tipset.Cids()
	num, err := models.QueryBlockNum(cids)
	if err != nil {
		log("SynLotus FindCidsCount err  = %v", err)
		log("cids = %v ", cids)
		return
	}
	if num == len(cids) {
		log("Syn tipset exist Height =%v ", tipset.Height())
		return
	}
	// filscaner.FilscanerInst.NotifyHeaderChanged(store.HCApply, tipset)
	var e Element
	e.tipset = tipset
	bmList, err := Tipset2BlocksMsg(tipset)
	if err != nil {
		return
	}
	e.blocks = bmList
	tipset.Parents()
	parents := tipset.Parents().Cids()
	if len(parents) < 1 {
		return
	}
	parent, _ := LotusApi.ChainGetBlock(context.TODO(), parents[0])
	ok := TipsetQueue.UpdatePush(&e, parent.Height) //
	parentMessages, err := LotusApi.ChainGetParentMessages(context.TODO(), tipset.Blocks()[0].Cid())
	if err != nil {
		log("ChainGetParentMessages err = %v", err)
		return
	}
	//parent msg ChainGetParentReceipts
	parentReceipts, err := LotusApi.ChainGetParentReceipts(context.TODO(), tipset.Blocks()[0].Cid())
	if err != nil {
		log("ChainGetParentReceipts err = %v", err)
		return
	}
	TipsetQueue.MsgUpdateReceipt(parentMessages, parentReceipts, parent.Height, 0) //update parent msg Receipt

	go func(ok bool, parentHeight uint64) {
		if !ok {
			return
		}

		t, err := LotusApi.ChainGetTipSetByHeight(context.TODO(), parentHeight, tipset)
		if err != nil {
			log("SynLotus ChainGetTipSet err  = %v", err)
		}
		time.Sleep(100 * time.Millisecond)
		GetTipSetPushQueue(t)
	}(ok, parent.Height)
}

func GetPledgeCollateral(tipset *types.TipSet) (string, error) {
	bigInt, err := LotusApi.StatePledgeCollateral(context.TODO(), tipset)
	return types.FIL(bigInt).String(), err
}

func GetLotusHead() (tipset *types.TipSet, err error) {
	tipset, err = LotusApi.ChainHead(context.TODO())
	if err != nil {
		log("SynLotus ChainHead err = %v", err)
		return
	}
	return
}

func GetActorByAddress(ad string) (actor *types.Actor, err error) {
	if len(ad) < 1 {
		return
	}
	address, err := address.NewFromString(ad)
	if err != nil {
		return
	}
	tipset, err := LotusApi.ChainHead(context.TODO())
	if err != nil {
		ps("get ActorByAddress failed, message:%s\n", err.Error())
		return nil, err
	}
	actor, err = LotusApi.StateGetActor(context.TODO(), address, tipset)
	return
}

func SavePeers() (err error) {
	res, err := GetNetPeers()
	if err != nil {
		log("SavePeers err =%v", err)
		return
	}
	geoIpUserId := conf("geoIpUserId")
	geoIpKey := conf("geoIpKey")
	var peers []*models.Peer
	for _, value := range res {
		res, err := models.GetPeerByPeerId(value.Ip) //peer is have
		if err != nil {
			log("GetPeerByPeerId err =%v", err)
			continue
		}
		if res != nil && models.TimeNow-res.GmtModified < 7*24*60*60 { //latest update time > 1 week
			models.UpdatePeerGmtModifiedByPeerId(res.PeerId)
			continue
		}
		info, err := utils.GetIpDetails(geoIpUserId, geoIpKey, value.Ip)
		if err != nil || info == nil {
			log("err =%v ,info=%v", err, info)
			continue
		}
		value.LocationCN = info.LocationCN
		value.LocationEN = info.LocationEN
		value.Longitude = info.Longitude
		value.Latitude = info.Latitude
		peers = append(peers, value)
	}
	err = models.InsertPeerMulti(peers)
	if err != nil {
		log("InsertPeer err = [%v]", err)
		return err
	}
	return nil
}

func GetNetPeers() ([]*models.Peer, error) {
	if LotusCommonApi == nil {
		return nil, nil
	}
	peers, err := LotusCommonApi.NetPeers(context.TODO())

	if err != nil {
		log("GetNetPeers err=%v", err)
	}
	var res []*models.Peer
	for _, value := range peers {
		p := new(models.Peer)
		if len(value.Addrs) > 0 {
			p.IpAddr = value.Addrs[0].String()
			// /ip4/192.168.0.102/tcp/53814   len=5
			ipArr := strings.Split(value.Addrs[0].String(), "/")
			if len(ipArr) > 3 {
				p.Ip = ipArr[2]
			}
		}
		p.PeerId = ps("%v", value.ID)
		res = append(res, p)
	}
	return res, nil
}

func UpdateAccountInfo(startTime *int64) error {
	accountMap := make(map[string]string)
	fromArr, err := models.GetDistinctFromAddressByTime(*startTime, models.TimeNow)
	if err != nil {
		return err
	}
	toArr, err := models.GetDistinctToAddressByTime(*startTime, models.TimeNow)
	if err != nil {
		return err
	}
	minerArr, err := models.GetDistinctMinerAddressByTime(*startTime, models.TimeNow)
	if err != nil {
		return err
	}
	OwnerArr, err := models.GetDistinctWalletAddressByTime(*startTime, models.TimeNow)
	if err != nil {
		return err
	}
	for _, value := range fromArr {
		if value[0:2] == "t3" {
			accountMap[value] = "wallet"
		} else {
			accountMap[value] = "account"
		}
	}
	for _, value := range toArr {
		if value[0:2] == "t3" {
			accountMap[value] = "wallet"
		} else {
			accountMap[value] = "account"
		}
	}
	for _, value := range minerArr {
		accountMap[value] = "miner"
	}
	for _, value := range OwnerArr {
		accountMap[value] = "owner"
	}
	var accountlist []*models.Account
	for k, value := range accountMap {
		ac := new(models.Account)
		switch value {
		case "miner":
			ac.IsMiner = true
		case "wallet":
			ac.IsWallet = true
		case "owner":
			ac.IsOwner = true
		}
		ac.Address = k
		accountlist = append(accountlist, ac)
	}
	for k, account := range accountlist {
		log("%v", k)
		Actor, err := GetActorByAddress(account.Address)
		if err != nil {
			log("GetActorByAddress err = %v", err)
			continue
		}
		accountlist[k].Actor = Actor
	}
	err = models.UpsertAccountArr(accountlist)
	if err != nil {
		log("UpsertAccountArr err = %v", err)
		return err
	}
	t := time.Now().Unix()
	startTime = &t
	return nil
}

func GetStateListMiners() {
	tipset, err := LotusApi.ChainHead(context.TODO())
	if err != nil {
		log("GetStateListMiners err = %v", err)
		return
	}
	mingerList, err := LotusApi.StateListMiners(context.TODO(), tipset)
	fmt.Println("StateListMiners")
	for _, value := range mingerList {
		actoeInfo, _ := LotusApi.StateGetActor(context.TODO(), value, tipset)
		fmt.Println(ps("miner Address=%v,actoeInfo.Code=%v ,actoeInfo.Head=%v ,actoeInfo.Nonce=%v ,actoeInfo.Balance=%v ", value.String(), actoeInfo.Code.String(), actoeInfo.Head.String(), actoeInfo.Nonce, types.FIL(actoeInfo.Balance).String()))
	}
}

func GetStateListActors() {
	tipset, err := LotusApi.ChainHead(context.TODO())
	if err != nil {
		log("GetStateListActors err = %v", err)
		return
	}
	acList, err := LotusApi.StateListActors(context.TODO(), tipset)
	fmt.Println("StateListActors")
	for _, value := range acList {

		fmt.Print(value.String())
	}
}

func GetWalletList() {
	wList, err := LotusApi.WalletList(context.TODO())
	if err != nil {
		log("GetWalletList err = %v", err)
		return
	}
	fmt.Println("StateListActors")
	for _, value := range wList {
		fmt.Print(value.String())
	}
}

func GetWalletBalance() {
	ad := "t3uw66e5ctcjhrqc2zo4tr7yguweygwmnpdybzwav3erqwvm2hm7scnlf3ypwl5x3ws4jfbxm4t6tjr6ljoydq"
	addr, _ := address.NewFromString(ad)
	wbanlance, err := LotusApi.WalletBalance(context.TODO(), addr)
	if err != nil {
		log("GetWalletList err = %v", err)
		return
	}
	fmt.Println("", types.FIL(wbanlance).String())
}

func GetWalletActor() {
	ad := "t3uw66e5ctcjhrqc2zo4tr7yguweygwmnpdybzwav3erqwvm2hm7scnlf3ypwl5x3ws4jfbxm4t6tjr6ljoydq"
	addr, _ := address.NewFromString(ad)
	tipset, err := LotusApi.ChainHead(context.TODO())
	if err != nil {
		log("GetStateListActors err = %v", err)
		return
	}
	actoeInfo, _ := LotusApi.StateGetActor(context.TODO(), addr, tipset)
	if err != nil {
		log("GetWalletList err = %v", err)
		return
	}
	fmt.Println(ps("Wallet Address=%v,actoeInfo.Code=%v ,actoeInfo.Head=%v ,actoeInfo.Nonce=%v ,actoeInfo.Balance=%v ", addr.String(), actoeInfo.Code.String(), actoeInfo.Head.String(), actoeInfo.Nonce, types.FIL(actoeInfo.Balance).String()))
}

func init() {
	go func() {
		time.Sleep(2 * time.Second)
		//GetStateListMiners()
		//GetStateListActors()
		//GetWalletActor()
		//GetWalletList()
		//GetWalletBalance()
		//TestUpsertAccount()
		//GetNetPeers()
		//SavePeers()
		//GetParentMsgReceipts()
		//var a  int64
		//UpdateAccountInfo(&a)
	}()
}

func GetParentMsgReceipts() {
	tipset, err := LotusApi.ChainHead(context.TODO())
	if err != nil {
		log("ChainHead err = %v", err)
		return
	}
	for k, value := range tipset.Blocks() {
		bm, err := LotusApi.ChainGetBlockMessages(context.TODO(), value.Cid())
		if err != nil {
			log("ChainGetBlockMessages err = %v", err)
			return
		}
		log("block %v msg len=%v", k, len(bm.Cids))
		bmb, _ := json.Marshal(bm.Cids)
		log("ChainGetParentMessages = %v", string(bmb))

		pr, err := LotusApi.ChainGetParentReceipts(context.TODO(), value.Cid())
		if err != nil {
			log("ChainGetParentReceipts err = %v", err)
			return
		}
		log("block %v msg len=%v", k, len(pr))
		prb, _ := json.Marshal(pr)
		log("ChainGetParentMessages = %v", string(prb))

	}
	pm, err := LotusApi.ChainGetParentMessages(context.TODO(), tipset.Blocks()[0].Cid())
	if err != nil {
		log("ChainGetParentMessages err = %v", err)
		return
	}
	log("ChainGetParentMessages len=%v", len(pm))
	b, _ := json.Marshal(pm)
	log("ChainGetParentMessages = %v", string(b))

	//re ,err := LotusApi.StateGetReceipt(context.TODO(),pm[0].Cid,tipset)
	//if err != nil {
	//	log("StateGetReceipt err = %v", err)
	//	return
	//}
	//reb,_ := json.Marshal(re)
	//log("StateGetReceipt = %v",string(reb))

}

func TestUpsertAccount() {
	//ad  := "t3uw66e5ctcjhrqc2zo4tr7yguweygwmnpdybzwav3erqwvm2hm7scnlf3ypwl5x3ws4jfbxm4t6tjr6ljoydq"
	//addr ,_ := address.NewFromString(ad)
	tipset, err := LotusApi.ChainHead(context.TODO())
	if err != nil {
		log("GetStateListActors err = %v", err)
		return
	}
	acList, err := LotusApi.StateListActors(context.TODO(), tipset)
	fmt.Println("StateListActors")
	for _, value := range acList {
		actoeInfo, _ := LotusApi.StateGetActor(context.TODO(), value, tipset)
		if err != nil {
			log("GetWalletList err = %v", err)
			return
		}
		ac := new(models.Account)
		ac.Address = value.String()
		ac.Actor = actoeInfo
		ac.IsWallet = true
		err = models.UpsertAccount(ac)
		if err != nil {
			panic(err)
		}
	}

}
