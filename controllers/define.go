package controllers

import (
	"context"
	"crypto/md5"
	"filscan_lotus/filscaner"
	"filscan_lotus/filscanproto"
	"filscan_lotus/models"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/config"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/go-redis/redis"
	"go.uber.org/zap"
	log2 "log"
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"time"
)

var UserTokenMap map[string]*vcode_t
var ps = fmt.Sprintf
var RedisClient *redis.Client
var SidLife int64
var TipsetQueue0 *SliceEntry
var flscaner *filscaner.Filscaner

var conf = beego.AppConfig.String
var Logger *zap.SugaredLogger
var LotusApi api.FullNode
var LotusCommonApi api.Common
var lotusBaseInformation *LotusBaseInformation
var peerPointCash *PeerPointCash
var avgBlockTimeCash *AvgBlockTimeCash
var avgBlockSizeCash *AvgBlockSizeCash
var totalPowerCash *TotalPowerCash

type LotusBaseInformation struct {
	TipsetHeight      uint64
	BlockReward       float64
	AvgMessageSize    float64
	AvgGasPrice       float64
	AvgMessagesTipset float64
	PledgeCollateral  string
	Time              int64
	CashTime          int64
}

type TotalPowerCash struct {
	StorageCapacity float64
	Data            []*filscanproto.TotalPowerGraphical
	Time            int64
	CashTime        int64
}

type PeerPointCash struct {
	PeerPoint []*filscanproto.PeerPoint
	Time      int64
	CashTime  int64
}

type AvgBlockTimeCash struct {
	BlockTime       []*filscanproto.Blocktime
	TotalBlockCount int
	Max             string
	Min             string
	AvgBlockTime    string
	Time            int64
	CashTime        int64
}

type AvgBlockSizeCash struct {
	BlockSize    []*filscanproto.Blocksize
	AvgBlocksize float64
	Max          float64
	Min          float64
	Time         int64
	CashTime     int64
}

type Recv struct {
	Code int
	Msg  string
	Data interface{}
}

type vcode_t struct {
	code     string
	lasttime int64
}

func Firstinit() {
	//RedisInit()
	//OriginInit()
	BeegoInit()
	LotusInit()
	models.Db_init(beego.AppConfig)
	models.TimenowInit()
	LoggerInit()
	// MysqlInit()
	ArgInit()
	UpdatePeers()
	UpdateAllAccount()
	return
}

func SetFilscaner(filscaner *filscaner.Filscaner) {
	flscaner = filscaner
}

func Run() {
	go FirstSynLotus()
}

func UpdateAllAccount() {
	//updateAccount
	updateAccount := conf("updateAccount")
	updateAccountInt, _ := strconv.ParseInt(updateAccount, 10, 64)
	if updateAccountInt == 0 {
		updateAccountInt = 60
	}
	go func() {
		var startTime int64 = 0
		tick := time.Tick(time.Duration(updateAccountInt) * time.Second)
		for {
			<-tick
			log("Run UpdateAllAccount cycle：%v ", updateAccountInt)
			err := UpdateAccountInfo(&startTime)
			if err != nil {
				log("UpdatePeers，err = %v ", err)
			}
		}
	}()
}

func UpdatePeers() {
	updatePeer := conf("updatePeer")
	updatePeerInt, _ := strconv.ParseInt(updatePeer, 10, 64)
	if updatePeerInt == 0 {
		updatePeerInt = 1800
	}
	go func() {
		tick := time.Tick(time.Duration(updatePeerInt) * time.Second)
		for {
			<-tick
			log("Run UpdatePeers cycle：%v ", updatePeerInt)
			err := SavePeers()
			if err != nil {
				log("UpdatePeers，err = %v ", err)
			}
		}
	}()
}

func ArgInit() {
	lotusBaseInformationCash := conf("lotusBaseInformationCash")
	CashTime, _ := strconv.ParseInt(lotusBaseInformationCash, 10, 64)
	lotusBaseInformation = &LotusBaseInformation{CashTime: CashTime}

	peerMapCash := conf("peerMapCash")
	peerMapCashTime, _ := strconv.ParseInt(peerMapCash, 10, 64)
	peerPointCash = &PeerPointCash{CashTime: peerMapCashTime}

	homeGraphicalCash := conf("homeGraphicalCash")
	HomeCashTime, _ := strconv.ParseInt(homeGraphicalCash, 10, 64)
	avgBlockTimeCash = &AvgBlockTimeCash{CashTime: HomeCashTime}
	avgBlockSizeCash = &AvgBlockSizeCash{CashTime: HomeCashTime}
	totalPowerCash = &TotalPowerCash{CashTime: HomeCashTime}

	if false {
		inst, err := filscaner.NewInstance(context.TODO(), "./conf/app.conf", LotusApi)
		if err != nil {
			panic(ps("filscaner.NewInstance fail:%v", err.Error()))
		}
		flscaner = inst
	}
}

func LotusInit() {
	lotusGetWay := conf("lotusGetWay")
	cli, stopper, err := client.NewFullNodeRPC("ws://"+lotusGetWay+"/rpc/v0", nil)
	if err != nil {
		defer stopper()
		log2.Fatalln(ps("get lotus connect err, ,err=[%v]", err))
	} else {
		LotusApi = cli
	}
	commonclient, commonstopper, err := client.NewCommonRPC("ws://"+lotusGetWay+"/rpc/v0", nil)
	if err != nil {
		defer commonstopper()
		log2.Fatalln(ps("get lotus commonclient connect err, ,err=[%v]", err))
	} else {
		peerId, err := cli.ID(context.TODO())
		if err != nil {
			defer commonstopper()
			log2.Fatalln(ps("get lotus commonclient connect err, ,err=[%v]", err))
		} else {
			log("connect lotus success,peerId=%v", peerId)
			LotusCommonApi = commonclient
		}
	}
}

func BeegoInit() {
	filepath := "conf/app.conf"
	// filepath = "conf/local.app.conf"
	conf1, err := config.NewConfig("ini", filepath)
	if err != nil {
		panic(err)
	}
	conf = conf1.String
}

func LoggerInit() {
	// 初始化 logger 实例
	rawLogger, err := zap.NewDevelopment(zap.Fields(zap.String("serive", "filscan")))
	if err != nil {
		panic(err)
	}
	Logger = rawLogger.Sugar()
}

func RedisInit() {
	/**
	文档地址：
	https://godoc.org/github.com/go-redis/redis
	*/
	client := redis.NewClient(&redis.Options{
		Addr:     conf("redis"),
		Password: conf("redisPwd"), // no password set
		DB:       0,                // use default DB
	})
	_, err := client.Ping().Result()
	if err != nil {
		log("err=%v", err)
		panic(ps("连接redis失败，终止启动，err=%v", err))
		return
	} else {
		log("redis init success")
	}
	RedisClient = client
}

func log(format string, v ...interface{}) {
	fmt.Println(fmt.Sprintf("[debug][%s]", models.TimeNowStr), fmt.Sprintf(format, v...))
	return
}

func StrToMD5(str string) (result string) {
	md5Ctx1 := md5.New()
	md5Ctx1.Write([]byte(str))
	result = ps("%x", md5Ctx1.Sum(nil))
	return
}

func MysqlInit() {
	//orm.Debug = false
	/*
		var maxIdle int = 30
		var maxConn int = 30
		err := orm.RegisterDataBase("default", "mysql", conf("mysql"), maxIdle, maxConn)
		if err != nil {
			panic("mysql init fail，终止启动")
			return
		} else {
			log("mysql init success")
		}*/
}

func CheckArg(a ...interface{}) bool {
	for _, arg := range a {
		switch reflect.TypeOf(arg).Kind() {
		case reflect.String:
			if arg.(string) == "" {
				return false
			}
		case reflect.Int64:
			if arg.(int64) == 0 {
				return false
			}
		case reflect.Int32:
			if arg.(int32) == 0 {
				return false
			}
		case reflect.Int:
			if arg.(int) == 0 {
				return false
			}
		case reflect.Float64:
			if arg.(float64) < 0.000001 && arg.(float64) > -0.000001 {
				return false
			}
		case reflect.Float32:
			if arg.(float32) < 0.000001 && arg.(float32) > -0.000001 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

//生成随机字符串 数字+小写
func GetRandomlowS(size int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < size; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

//生成随机字符串 数字+大写+小写
func GetRandomS(size int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < size; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

//小数点后n位 is 是否四舍五入
func Round(f float64, n int, is bool) float64 {
	pow10_n := math.Pow10(n)
	if is {
		return math.Trunc((f+0.5/pow10_n)*pow10_n) / pow10_n
	} else {
		return math.Trunc((f)*pow10_n) / pow10_n
	}
}

//小数点后n位 is 是否四舍五入
func RoundString(f float64, n int, is bool) string {
	pow10_n := math.Pow10(n)
	if is {
		return strconv.FormatFloat(math.Trunc((f+0.5/pow10_n)*pow10_n)/pow10_n, 'f', -1, 64)
	} else {
		return strconv.FormatFloat(math.Trunc((f)*pow10_n)/pow10_n, 'f', -1, 64)
	}
}
func getRandomString(size int) string {
	str := "0123456789"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < size; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}
