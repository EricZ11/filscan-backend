module filscan_lotus

go 1.13

replace github.com/filecoin-project/filecoin-ffi => ./extern/lotus/extern/filecoin-ffi

replace github.com/filecoin-project/lotus => ./extern/lotus

replace gitlab.forceup.in/dev-go/gosf => github.com/ipfs-force-community/gosf v0.1.16

replace gitlab.forceup.in/dev-proto/common => github.com/ipfs-force-community/common v0.1.0

require (
	github.com/astaxie/beego v1.12.0
	github.com/filecoin-project/go-address v0.0.0-20200107215422-da8eea2842b5
	github.com/filecoin-project/lotus v0.0.0-00010101000000-000000000000
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8
	github.com/go-redis/redis v6.15.6+incompatible
	github.com/golang/protobuf v1.3.2
	github.com/ipfs/go-block-format v0.0.2
	github.com/ipfs/go-cid v0.0.4
	github.com/ipfs/go-ipfs-blockstore v0.1.1
	github.com/ipfs/go-log v1.0.1
	github.com/libp2p/go-libp2p-core v0.3.0
	github.com/savaki/geoip2 v0.0.0-20150727150920-9968b08fbf39
	github.com/shiena/ansicolor v0.0.0-20151119151921-a422bbe96644 // indirect
	github.com/whyrusleeping/cbor-gen v0.0.0-20200121162646-b63bacf5eaf8
	gitlab.forceup.in/dev-go/gosf v0.0.0-00010101000000-000000000000
	gitlab.forceup.in/dev-proto/common v0.1.0
	go.uber.org/zap v1.13.0
	golang.org/x/net v0.0.0-20190921015927-1a5e07d1ff72
	google.golang.org/grpc v1.23.0
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22
	gopkg.in/urfave/cli.v2 v2.0.0-20180128182452-d3ae77c26ac8
)

replace github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0
