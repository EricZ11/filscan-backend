package main

import (
	"context"
	"filscan_lotus/controllers"
	"filscan_lotus/filscaner"
	lotus_filscan "filscan_lotus/filscanproto"
	"fmt"
	"github.com/astaxie/beego/config"
	"github.com/ipfs-force-community/gosf/jsonrpc"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	// 初始化 logger 实例
	rawLogger, err := zap.NewDevelopment(zap.Fields(zap.String("serive", "lotus_filscan")))
	if err != nil {
		panic(err)
	}

	//jsonrpc.AddCustomizeCORSHeader("sid", "token", "X-Force-Token", "X-Force-Key")
	Logger := rawLogger.Sugar()

	// 初始化 root mux
	rootMux := jsonrpc.NewRootMux("", Logger)

	// base server
	baseMux := lotus_filscan.NewJSONRpcMuxForFilscan(Logger, &controllers.FilscanServer{})

	filscaner := filscaner.FilscanerInst

	tipsetMux := lotus_filscan.NewJSONRpcMuxForFilscanTipset(Logger, &controllers.FilscanTipset{})
	msgMux := lotus_filscan.NewJSONRpcMuxForFilscanMessages(Logger, &controllers.FilscanMessages{})
	accountMux := lotus_filscan.NewJSONRpcMuxForFilscanAccount(Logger, &controllers.FilscanAccount{})
	peerMux := lotus_filscan.NewJSONRpcMuxForFilscanPeer(Logger, &controllers.FilscanPeer{})

	miningMux := lotus_filscan.NewJSONRpcMuxForFilscanMining(Logger, filscaner)
	tokenMux := lotus_filscan.NewJSONRpcMuxForFilscanToken(Logger, filscaner)

	/*baseMux := forceup.NewJSONRpcMuxForForceUp(Logger, &service.ForceUpUserBaseServer{})
	// 生成 FilScan API Group   用户端
	userMux := forceup.NewJSONRpcMuxForForceUpUser(Logger, &service.ForceUpUserOnlineServer{})
	userMux.Use(session.InjectLoginUser())
	fileMux := forceup.NewJSONRpcMuxForForceUpFiles(Logger, &service.ForceUpUserOnlineFileServer{})
	fileMux.Use(session.InjectLoginUser())

	// 将 force API Group 注册到 root 中
	adminBaseMux := forceup.NewJSONRpcMuxForForceAdmin(Logger, &service.ForceAdminBaseServer{})
	adminListMux := forceup.NewJSONRpcMuxForForceAdminOnline(Logger, &service.ForceAdminListServer{})
	adminListMux.Use(session.InjectLoginUser())
	adminOperateMux := forceup.NewJSONRpcMuxForForceAdminOperate(Logger, &service.ForceAdminOperateServer{})
	adminOperateMux.Use(session.InjectLoginUser())

	// user 依赖 base
	baseMux.AddSubs(userMux)
	baseMux.AddSubs(fileMux)
	baseMux.AddSubs(adminBaseMux)
	baseMux.AddSubs(adminListMux)
	baseMux.AddSubs(adminOperateMux)
	*/
	// 将 force API Group 注册到 root 中
	//rootMux.AddSubs(baseMux)
	//rootMux.AddSubs(adminBaseMux)

	// 使用标准库中的 http.ServeMux 和 http.Server 构建最终的 http 服务
	// 初始化一个空的 http.ServeMux
	baseMux.AddSubs(tipsetMux)
	baseMux.AddSubs(msgMux)
	baseMux.AddSubs(accountMux)
	baseMux.AddSubs(miningMux)
	baseMux.AddSubs(tokenMux)
	baseMux.AddSubs(peerMux)
	rootMux.AddSubs(baseMux)
	httpMux := http.NewServeMux()

	// 将 rootMux 中携带的接口, 中间件, 子组等封装成 http.HandlerFunc 并注册到 http.ServeMux 上
	jsonrpc.RegisterMux(httpMux, rootMux)

	config_file := "conf/app.conf"

	iniconf, err := config.NewConfig("ini", config_file)
	// iniconf, err := config.NewConfig("ini", "conf/local.app.conf")
	if err != nil {
		panic(err)
	}
	httpport := iniconf.String("httpport")

	//go metric.Run(context.Background(), metric.DefaultConfig)

	listenAdd := iniconf.String("listenAdd")

	httpServer := http.Server{
		Addr:    fmt.Sprintf("%s:%s", listenAdd, httpport),
		Handler: httpMux,
	}
	controllers.Firstinit() //初始化
	if err := filscaner.Init(context.TODO(), config_file, controllers.LotusApi); err != nil {
		panic(err)
	}
	controllers.SetFilscaner(filscaner)
	filscaner.Run()
	controllers.Run() //SynLotus

	Logger.Info(fmt.Sprintf("server will listen %s:%s", listenAdd, httpport))
	// 开始监听并想向外提供服务
	if err := httpServer.ListenAndServe(); err != nil {
		Logger.Errorf("http server listen and serve error, cause=%v", err)
	}

	// todo: need a stop method...
	// filscaner.Stop()

}
