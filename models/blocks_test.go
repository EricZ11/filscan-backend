package models

import (
	"fmt"
	"testing"
)

func TestGetBlockCountByTime(t *testing.T) {
	gotCount, err := GetBlockCountByTime(1576617525, 1576635525)
	if err != nil {
		panic(err)
	}
	fmt.Println(gotCount)
}

func TestGetBlockSumSizeByTime(t *testing.T) {
	gotCount, err := GetBlockSumSizeByTime(1576617525, 1576635525)
	if err != nil {
		panic(err)
	}
	fmt.Println(gotCount)
}

func TestGetBlockTotalRewardFilByMiner(t *testing.T) {
	var a []string
	a = append(a, "t01540")
	gotTotal, err := GetBlockTotalRewardFilByMiner(a)
	if err != nil {
		panic(err)
	}
	fmt.Println(gotTotal)

}

func TestGetDistinctMinerByTime(t *testing.T) {
	gotRes, err := GetDistinctMinerByTime(0, 1581758125)
	if err != nil {
		panic(err)
	}
	fmt.Println(gotRes)
}

func TestGetBlockByMsg(t *testing.T) {

	gotRes, err := GetBlockByMsg("bafy2bzaceb3uo4ripf6ch2u6gl5xspfymwkdidufr4wnh2b53au4pss4wnmpk")
	if err != nil {
		panic(err)
	}
	fmt.Println(gotRes)

}
