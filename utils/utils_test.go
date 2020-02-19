package utils

import (
	"fmt"
	"reflect"
	"testing"
)

func TestTipsetkey_from_string(t *testing.T) {
	k := "{bafy2bzacedyoq3papg7r3cyfpie6qnerevzgfgkr764lyp6as27ivxbittqvk,bafy2bzacebod755c3i5stf46omggd2zbpekbrmktmy7lgkxprzuwtfsf3zm56,bafy2bzacecysl7yr2xwpznhm2jrnmxi6xlsdxcd5bfrem7hjqzpbivpbx25ww,bafy2bzacedg56cslwrwylcqckocsckgkoz5ghkuivre7pxqsnvt3j54iupcqs}"
	fmt.Println(Tipsetkey_from_string(k).String() == k)
}

var display = fmt.Printf
var displayln = fmt.Println



func Test_arr_map(t *testing.T) {
	type MINER struct {
		Miner string
	}
	// miners := []*MINER{ &MINER{"t01234"}}
	miners := []string{"t01234"}

	in := interface{}(miners)

	out := SlcToMap(in, "", false)

	displayln(out)
	displayln(reflect.TypeOf(out))

	if o, isok := out.(map[string]struct{}); isok {
		displayln(o)
	} else if o, isok := out.(map[string]*MINER); isok {
		displayln(o)
	} else {
		displayln(o)
	}

}
