package models

import (
	"fmt"
	"testing"
)

func TestGetTipSetByBlockCid(t *testing.T) {
	gotRes, err := GetTipSetByBlockCid("bafy2bzaceajqqryadk2jbmywvmmwtvdwrh43xpbzzq4muve3bigz6dikr5nkw")
	if err != nil {
		panic(ps("GetTipSetByBlockCid() error = %v ", err))
		return
	}
	fmt.Println(gotRes)

}

func TestThanHeightCount(t *testing.T) {
	gotRes, err := ThanHeightCount(2)
	if err != nil {
		panic(ps("GetTipSetByBlockCid() error = %v ", err))
		return
	}
	fmt.Println(gotRes)
}
