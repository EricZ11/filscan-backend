package main

import (
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"math/big"
)

var printf = fmt.Printf

type MyBigint struct{ *big.Int }

func NewMyBigint(i int64) *MyBigint {
	return &MyBigint{
		Int: big.NewInt(i),
	}
}

func (mbig *MyBigint) GetBSON() (interface{}, error) {
	fmt.Printf("get bosn return string:%s\n", mbig.String())
	return mbig.String(), nil
}

func (mbig *MyBigint) SetBSON(raw bson.Raw) error {
	var num string
	if err := raw.Unmarshal(&num); err != nil {
		return err
	}

	if mbig.Int == nil {
		mbig.Int = big.NewInt(0)
	}

	if _, isok := mbig.SetString(num, 10); !isok {
		return fmt.Errorf("convert '%s' to bigint failed", num)
	}
	return nil
}

type LargeNubmerSturct struct {
	Bigint *MyBigint
	Name   string
}

func main() {
	if addr, err := address.NewFromString("t01346"); err != nil {
		fmt.Printf("not a miner address:%s\n", err.Error())
	} else {
		fmt.Printf("address:%s\n", addr)
	}

	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	c := session.DB("test").C("test")

	var largenumber = &LargeNubmerSturct{
		Bigint: NewMyBigint(10),
		Name:   "zl",
	}

	if _, err := c.Upsert(nil, largenumber); err != nil {
		fmt.Printf("err message:%s\n", err.Error())
	}

	largnum := &LargeNubmerSturct{}
	c.Find(nil).One(largnum)

	fmt.Printf("%s\n", largnum.Bigint.String())
}

// {
// "_id": "5e00a797125181d5a443f492",
// "block_count": 0,
// "block_count_percent": 0,
// "gmt_create": 1577102769,
// "gmt_modified": 1577102769,
// "incoming_sectorsize": 0,
// "mine_time": 1577100690,
// "miner_addr": "t01011",
// "miner_create": "",
// "nick_name": "",
// "peer_id": "12D3KooWL9QUqDh8tGc2S9JZCnnUboSj7abKqthcnzjnxvFpfeAW",
// "power": 4795330985984,
// "power_percent": 0,
// "sector_count": 4545,
// "sector_size": 1073741824,
// "tipset_height": 22562,
// "total_power": 1.980704062856613e+28,
// "wallet_addr": "t3ro2qkf755ree4alk3lir7san6zys3vxxpfz2mauyw6gs4gusgo23em7bzdh6evsfewrfm6dmn6icnjswsmrq"
// }
