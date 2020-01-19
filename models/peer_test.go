package models

import (
	"fmt"
	"testing"
	"time"
)

func TestGetPeerGroup(t *testing.T) {
	res, err := GetPeerGroup()
	if err != nil {
		panic(err)
	}
	fmt.Println(res)
}

func TestUpdatePeerGmtModifiedByPeerId(t *testing.T) {
	err := UpdatePeerGmtModifiedByPeerId("12D3KooWBHsuR5EfbAt5xr7Bi9E7ff54mUa3XheC2q2MsLuD7PuW")
	if err != nil {
		panic(err)
	}
	fmt.Println(err)
}

func TestGetActivePeerCountByTime(t *testing.T) {
	got, err := GetActivePeerCountByTime(time.Now().Unix())
	if err != nil {
		panic(err)
	}
	fmt.Println(got)
}
