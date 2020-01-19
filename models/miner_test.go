package models

import (
	"fmt"
	"testing"
)

func TestMinerListByWalletAddr(t *testing.T) {
	gotRes, err := MinerListByWalletAddr("t3tc577a5vogaykgmduqyy45hvzghhl3gxixsecvair6pepgwvwudesiegtea5dlpjii5kfdndqbp6qkzwbjdq")
	if err != nil {
		panic(err)
	}
	fmt.Println(gotRes)
}

func TestMinerByPeerId(t *testing.T) {
	gotMiner, err := MinerByPeerId("12D3KooWNM6Fz53ynbPZunzTBLVVg9FJEr6UMiWWFfe255FSinsp")
	if err != nil {
		panic(err)
	}
	fmt.Println(gotMiner)
}

func TestGetMinerstateActivateAtTime(t *testing.T) {
	got, err := GetMinerstateActivateAtTime(1576763685)
	if err != nil {
		panic(err)
	}
	fmt.Println(got[0].TotalPower)
}

func TestGetTotalpowerAtTime(t *testing.T) {
	got, err := GetTotalpowerAtTime(1576281600)
	if err != nil {
		panic(err)
	}
	fmt.Println(got.TotalPower.Int64())
}
