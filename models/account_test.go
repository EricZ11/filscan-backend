package models

import (
	"fmt"
	"github.com/filecoin-project/lotus/chain/types"
	"strconv"
	"testing"
)

func TestGetActorByAddress(t *testing.T) {
	gotRes, err := GetActorByAddress("t01")
	if err != nil {
		panic(err)
	}
	fmt.Println(gotRes)
}

func TestUpdateIsOwnerByAdress(t *testing.T) {
	err := UpdateIsOwnerByAdress("t011004")
	if err != nil {
		panic(err)
	}
	fmt.Println("ok")
}

func TestUpdateIsMinerByAdress(t *testing.T) {
	err := UpdateIsMinerByAdress("t01004")
	if err != nil {
		panic(err)
	}
	fmt.Println("ok")
}

func TestGetAccountSumBalance(t *testing.T) {
	gotTotal, err := GetAccountSumBalance()
	if err != nil {
		panic(err)
	}
	s1 := strconv.FormatFloat(gotTotal, 'f', -1, 64)
	bigI, err := types.BigFromString(s1)
	if err != nil {
		panic(err)
	}
	fmt.Println(types.FIL(bigI).String())

}
