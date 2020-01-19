package fblockstore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/repo"
	blocks "github.com/ipfs/go-block-format"
	cid "github.com/ipfs/go-cid"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	logging "github.com/ipfs/go-log"
	cbg "github.com/whyrusleeping/cbor-gen"
	"filscan_lotus/controllers/filscaner/force/factors"
	"filscan_lotus/controllers/filscaner/force/fblockstore/encoding"
	"filscan_lotus/controllers/filscaner/force/ftypes"
	"gopkg.in/mgo.v2"
	"gopkg.in/urfave/cli.v2"
)

var (
	log = logging.Logger("blockstore_mongo")

	mgoDialTimeout = 3 * time.Second
)

var _ blockstore.Blockstore = (*MgoStore)(nil)

// NewMgoStore 生成 MgoStore
func NewMgoStore(cctx *cli.Context) func(r repo.LockedRepo) (dtypes.ChainBlockstore, error) {
	return func(r repo.LockedRepo) (dtypes.ChainBlockstore, error) {
		inner, err := modules.ChainBlockstore(r)
		if err != nil {
			return nil, err
		}

		log.Infow("init blockstore on mongo database")
		dsn := cctx.String("bstore-mongo-dsn")
		if dsn == "" {
			log.Info("mongo db disabled")
			return inner, nil
		}

		dbname := cctx.String("bstore-mongo-db")
		if dbname == "" {
			dbname = "chain_syncer"
		}

		sess, err := mgo.DialWithTimeout(dsn, mgoDialTimeout)
		if err != nil {
			return nil, fmt.Errorf("unable to connect to mongo db, dsn=%s, err=%w", dsn, err)
		}

		log.Infow("mongostore initialized", "dsn", dsn, "db", dbname)

		return &MgoStore{
			dbname:  dbname,
			session: sess,
			inner:   inner,
		}, nil
	}
}

// MgoStore 数据双写
type MgoStore struct {
	dbname  string
	session *mgo.Session
	inner   blockstore.Blockstore
}

// DeleteBlock 实现 blockstore.Blockstore
func (m *MgoStore) DeleteBlock(id cid.Cid) error {
	return m.inner.DeleteBlock(id)
}

// Has 实现 blockstore.Blockstore
func (m *MgoStore) Has(id cid.Cid) (bool, error) {
	return m.inner.Has(id)
}

// Get 实现 blockstore.Blockstore
func (m *MgoStore) Get(id cid.Cid) (blocks.Block, error) {
	return m.inner.Get(id)
}

// GetSize 实现 blockstore.Blockstore
func (m *MgoStore) GetSize(id cid.Cid) (int, error) {
	return m.inner.GetSize(id)
}

// Put 实现 blockstore.Blockstore
func (m *MgoStore) Put(blk blocks.Block) error {
	err := m.inner.Put(blk)
	go func() {
		m.put(context.Background(), blk)
	}()

	return err
}

// PutMany 实现 blockstore.Blockstore
func (m *MgoStore) PutMany(blks []blocks.Block) error {
	err := m.inner.PutMany(blks)
	go func() {
		m.put(context.Background(), blks...)
	}()

	return err
}

// AllKeysChan 实现 blockstore.Blockstore
func (m *MgoStore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	return m.inner.AllKeysChan(ctx)
}

// HashOnRead 实现 blockstore.Blockstore
func (m *MgoStore) HashOnRead(enabled bool) {
	m.inner.HashOnRead(enabled)
}

// Has 实现 blockstore.Blockstore
func (m *MgoStore) put(ctx context.Context, blks ...blocks.Block) {
	if m.session == nil {
		return
	}

	items := make([]*upsertItem, 0, len(blks))
	seen := time.Now().Unix()

	for _, blk := range blks {
		bcid := blk.Cid()
		buf := bytes.NewReader(blk.RawData())

		{
			buf.Seek(0, io.SeekStart)

			var bh types.BlockHeader
			if err := bh.UnmarshalCBOR(buf); err == nil {
				items = append(items, &upsertItem{
					collection: "block_headers",
					content: fBlockHeader{
						Cid:           bcid,
						BlockHeader:   bh,
						SortedParents: ftypes.CopySortCids(bh.Parents),
						FirstSeen:     seen,
					},
				})

				continue
			}
		}

		{
			buf.Seek(0, io.SeekStart)

			var msg types.Message
			if err := msg.UnmarshalCBOR(buf); err == nil {
				items = append(items, &upsertItem{
					collection: "messages",
					content: fMessage{
						Cid:       bcid,
						Message:   msg,
						CallInfo:  m.parseMessageCall(&msg),
						FirstSeen: seen,
					},
				})

				continue
			}
		}

		{
			buf.Seek(0, io.SeekStart)

			var smsg types.SignedMessage

			if err := smsg.UnmarshalCBOR(buf); err == nil {
				innerCid := smsg.Message.Cid()

				items = append(items,
					&upsertItem{
						collection: "signed_messages",
						content: fSignedMessage{
							Cid:           bcid,
							SignedMessage: smsg,
							FirstSeen:     seen,
						},
					},
					&upsertItem{
						collection: "messages",
						content: fMessage{
							Cid:       innerCid,
							Message:   smsg.Message,
							CallInfo:  m.parseMessageCall(&smsg.Message),
							FirstSeen: seen,
						},
					},
				)

				continue
			}
		}

		{

			buf.Seek(0, io.SeekStart)

			var mMeta types.MsgMeta

			if err := mMeta.UnmarshalCBOR(buf); err == nil {
				items = append(items, &upsertItem{
					collection: "msg_metas",
					content: fMsgMeta{
						Cid:       bcid,
						MsgMeta:   mMeta,
						FirstSeen: seen,
					},
				})

				continue
			}
		}

		log.Debugw("type of incoming raw data is unknown", "cid", bcid)
	}

	if len(items) == 0 {
		return
	}

	db := m.session.DB(m.dbname)
	for _, item := range items {
		err := db.C(item.collection).Insert(encoding.MarshalWrap(item.content))
		if err != nil && !mgo.IsDup(err) {
			log.Warnw("unable to upsert", "err", err, "collection", item.collection)
		}
	}
}

func (m *MgoStore) parseMessageCall(msg *types.Message) CallInfo {
	var call CallInfo

	if msg.To.Protocol() != address.ID {
		return call
	}

	act, ok := factors.LookupByAddress(msg.To)
	if !ok {
		act, ok = factors.Lookup(actors.StorageMinerCodeCid)
	}

	if !ok {
		return call
	}

	call.Actor = act.Name

	meth, ok := act.LookupMethod(msg.Method)
	if !ok {
		return call
	}

	call.Method = meth.Name

	param := meth.NewParam()
	if param == nil {
		return call
	}

	um, ok := param.(cbg.CBORUnmarshaler)
	if !ok {
		return call
	}

	if err := um.UnmarshalCBOR(bytes.NewBuffer(msg.Params)); err != nil {
		log.Debugw("unable to marshal param", "addr", msg.To, "method", msg.Method)
		return call
	}

	call.Param = param

	return call
}

type upsertItem struct {
	collection string
	content    fcontent
}

type fcontent interface {
	f()
}

type fBlockHeader struct {
	Cid cid.Cid `bson:"_id"`
	types.BlockHeader
	SortedParents []cid.Cid
	FirstSeen     int64
}

func (fBlockHeader) f() {}

type fMessage struct {
	Cid cid.Cid `bson:"_id"`
	types.Message
	CallInfo  CallInfo
	FirstSeen int64
}

func (fMessage) f() {}

type fSignedMessage struct {
	Cid cid.Cid `bson:"_id"`
	types.SignedMessage
	FirstSeen int64
}

func (fSignedMessage) f() {}

type fMsgMeta struct {
	Cid cid.Cid `bson:"_id"`
	types.MsgMeta
	FirstSeen int64
}

func (fMsgMeta) f() {}

// CallInfo detail info of the actor call
type CallInfo struct {
	Actor  string
	Method string
	Param  interface{}
}
