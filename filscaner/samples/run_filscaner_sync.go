package main

import (
	"filscan_lotus/filscaner"
	"filscan_lotus/models"
	"filscan_lotus/utils"
	"time"
)

func filscaner_test() {
	models.Db_init(utils.GetConfiger())
	filscanor, err := filscaner.NewInstance(ctx, "./conf/app.conf", lotus_api)
	if err != nil {
		panic(err)
	}
	filscanor.Run()
	for {
		time.Sleep(time.Minute)
	}
	return

	// tipset, _ := lotus_api.ChainHead(ctx)
	// for i:=0; i<30; i++ {
	// 	tipset, _ = lotus_api.ChainGetTipSet(ctx, tipset.Parents())
	// }
	// latest_tipset, err := filscanor.sync_to_genesis(tipset)
	// if latest_tipset!=nil {
	// 	filscanor.Printf("synced to tipset = %d\n", latest_tipset.Height())
	// }
	// if err!=nil {
	// 	filscanor.Printf("error:%s\n", err.Error())
	// }
}

func main() {
	filscaner_test()
}
