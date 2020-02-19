package main

import (
	"fmt"
	"github.com/globalsign/mgo/bson"
	"testing"
	"time"
)

func TestUnmarshalLargeNumber(t *testing.T) {
	fmt.Printf("time = %d\n", time.Now().Unix())
	largenumber := &LargeNubmerSturct{
		Name:   "zengliang",
		Bigint: NewMyBigint(200000000000000000)}
	data, err := bson.Marshal(largenumber)
	if err != nil {
		fmt.Printf("err : %s\n", err.Error())
	} else {
		fmt.Printf("larg number = %s\n", string(data))
	}

	newlargenumber := LargeNubmerSturct{}
	if err := bson.Unmarshal(data, &newlargenumber); err != nil {
		fmt.Printf("err : %s\n", err.Error())
	}

	fmt.Printf("number is : %s\n", newlargenumber.Bigint.String())
	fmt.Printf("ok\n")
}
