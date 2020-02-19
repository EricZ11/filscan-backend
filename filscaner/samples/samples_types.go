package main

import (
	"context"
	"filscan_lotus/controllers"
	"filscan_lotus/filscaner"
	"filscan_lotus/models"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/lib/jsonrpc"
	"github.com/globalsign/mgo"
	"log"
)

var rpc_uri = "ws://127.0.0.1:1234/rpc/v0"

type MongoLog struct{}

func (MongoLog) Output(calldepth int, s string) error {
	log.SetFlags(log.Lshortfile)
	return log.Output(calldepth, s)
}

var fscaner *filscaner.Filscaner
var ctx context.Context
var cancel context.CancelFunc

var lotus_api api.FullNode
var lotus_closer jsonrpc.ClientCloser

var sm_API api.StorageMiner

// func NewStorageMinerRPC(addr string, requestHeader http.Header) (api.StorageMiner, jsonrpc.ClientCloser, error) {

func InitMongoLogs() {
	mgo.SetLogger(new(MongoLog))
}

func init() {
	ctx, cancel = context.WithCancel(context.TODO())

	var err error
	if lotus_api, lotus_closer, err = client.NewFullNodeRPC(rpc_uri, nil); err != nil {
		panic(err)
	}

	if sm_API, _, err = client.NewStorageMinerRPC(rpc_uri, nil); err != nil {
		panic(err)
	}

	if false {
		tipset, _ := lotus_api.ChainGetTipSetByHeight(ctx, 1, nil)
		fmt.Printf("tipset.key=%s, tipset.pareant=%s,height=%d\n",
			tipset.Key().String(), tipset.Parents().String(), tipset.Height())
	}
	// address, _ := address.NewFromString("t02718")
	// InitMongoLogs()
}

func init() {
	ctx, _ := context.WithCancel(context.TODO())
	controllers.BeegoInit()
	controllers.LotusInit()
	var err error

	var file_path string = "./conf/app.conf"

	models.Db_init(beego.AppConfig)

	fscaner, err = filscaner.NewInstance(ctx, file_path, controllers.LotusApi)
	if err != nil {
		panic(err)
	}
	fscaner.ChainHeadTest()
}
