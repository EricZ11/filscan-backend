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
