package filscaner

import (
	"sync"
)

var upsert_account_message_cash sync.Map

func (fs *Filscaner) handle_upsert_account_message(method *MethodCall) {
	/*
		starttime := time.Now().Unix()

		defer func() {
			endtime := time.Now().Unix()
			fs.Printf("-----------------------------------handle_upsert_account_message at(%v) : use time = %.3f(m)\n",
				method.Message.Cid().String(), (float64(endtime-starttime))/60)
		}()

		msg := method.Message
		to := msg.To
		from := msg.From
		toupdate, took := upsert_account_message_cash.Load(to.String())
		fromupdate, fromok := upsert_account_message_cash.Load(from.String())
		timeNow := time.Now().Unix()
		var doubleAddress []address.Address
		if !to.Empty() && (!took || timeNow-toupdate.(int64) > 10) && len(to.String()) > 2 { //不为空 &&（value 不存在 || update time > 10 ） && len >2
			doubleAddress = append(doubleAddress, to)
		}
		if !from.Empty() && (!fromok || timeNow-fromupdate.(int64) > 10) && len(from.String()) > 2 {
			doubleAddress = append(doubleAddress, from)
		}
		if len(doubleAddress) == 0 {
			return
		}
		var doubleAccount []*models.Account
		for _, address := range doubleAddress {
			ac := new(models.Account)
			ac.Address = address.String()
			actor, err := fs.GetActorByAddress(address)
			if err != nil {
				fs.Printf("get GetActorByAddress failed, message:%s\n",
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

			upsert_account_message_cash.Store(address.String(), time.Now().Unix())
		}
		err := models.UpsertAccountArr(doubleAccount)
		if err != nil {
			fs.Printf("handle_upsert_account_message UpsertAccountArr failed, err:%s\n",
				err.Error())
			return
		}
	*/
}

/*func (fs *Filscaner) GetActorByAddress(address address.Address) (actoe *types.Actor, err error) {

	defer recover()

	tipset, err := fs.api.ChainHead(context.TODO())
	if err != nil {
		fs.Printf("get ActorByAddress failed, message:%s\n", err.Error())
		return nil, err
	}
	actoe, err = fs.api.StateGetActor(context.TODO(), address, tipset)
	if err != nil {
		fmt.Println(err)
	}
	return
}*/
