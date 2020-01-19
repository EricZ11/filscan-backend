package utils

import (
	"fmt"
	"testing"
)

func TestGetIpDetails(t *testing.T) {
	gotRes, err := GetIpDetails("", "", "221.225.82.17")
	if err != nil {
		panic(err)
	}
	fmt.Println(gotRes)
}
