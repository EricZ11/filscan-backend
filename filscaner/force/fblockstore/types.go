package fblockstore

import (
	"github.com/globalsign/mgo/bson"
	"reflect"
	"strconv"

	"filscan_lotus/filscaner/force/fblockstore/encoding"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	cid "github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
)

var chainTypes = []reflect.Type{
	reflect.TypeOf(types.BlockHeader{}),
	reflect.TypeOf(types.Message{}),
	reflect.TypeOf(types.SignedMessage{}),
	reflect.TypeOf(types.MsgMeta{}),
}

func init() {
	encoding.RegisterInto(reflect.TypeOf(cid.Cid{}), encoding.Into(cidInto))
	encoding.RegisterInto(reflect.TypeOf(types.BigInt{}), encoding.Into(bigIngInto))
	encoding.RegisterInto(reflect.TypeOf(address.Address{}), encoding.Into(addressInto))
	encoding.RegisterInto(reflect.TypeOf(peer.ID("")), encoding.Into(peerIDInto))
	encoding.RegisterInto(reflect.TypeOf(uint64(0)), encoding.Into(uint64Into))

	for _, t := range chainTypes {
		if err := encoding.RegisterStruct(t); err != nil {
			panic(err)
		}
	}
}

func cidInto(v interface{}) (interface{}, error) {
	return v.(cid.Cid).String(), nil
}

func bigIngInto(v interface{}) (interface{}, error) {
	return bson.ParseDecimal128(v.(types.BigInt).String())
}

func addressInto(v interface{}) (interface{}, error) {
	return v.(address.Address).String(), nil
}

func peerIDInto(v interface{}) (interface{}, error) {
	return v.(peer.ID).String(), nil
}

func uint64Into(v interface{}) (interface{}, error) {
	return bson.ParseDecimal128(strconv.FormatUint(v.(uint64), 10))
}
