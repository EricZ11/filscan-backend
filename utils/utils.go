package utils

import (
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"math/big"
	"strconv"
	"strings"
	"time"
)

const PrecisionDefault = 8

var BlocksPerEpoch = big.NewInt(build.BlocksPerEpoch)

func ToFil(v *big.Int) float64 {
	fbig, _ := big.NewFloat(0).SetString(v.String())
	fv, _ := fbig.Float64()
	return TruncateNative(fv/build.FilecoinPrecision, PrecisionDefault)
}

func ToFilStr(v *big.Int) string {
	value := ToFil(v)
	return fmt.Sprintf("%.4f", value)
}

func TruncateNative(f float64, precision int) float64 {
	fs := fmt.Sprintf(fmt.Sprintf("%%.%df", precision+1), f)
	f, _ = strconv.ParseFloat(fs[:len(fs)-1], 64)
	return f
}

func to_xsize(power *big.Int, x XSIZE) float64 {
	fw := big.NewFloat(0)
	fw.SetString(power.String())

	ftbsize := big.NewFloat(0)
	ftbsize.SetString(x.String())

	v1, _ := fw.Float64()
	v2, _ := ftbsize.Float64()

	return TruncateNative(v1/v2, 2)
}

type XSIZE = *big.Int

var GB = XSIZE(big.NewInt(1 << 30))
var TB = XSIZE(big.NewInt(1 << 40))

func XSizeUintName(x XSIZE) string {
	if name, isok := XSizeUnitName[x]; isok {
		return name
	}
	return "unit not registed"
}

var XSizeUnitName = map[XSIZE]string{GB: "Gib", TB: "Tib"}

func XSizeString(size *big.Int) string {
	if size == nil {
		return "0"
	}
	return size.String()
}

func ToXSize(size *big.Int, unit XSIZE) string {
	size = big.NewInt(0).Set(size)
	return fmt.Sprintf("%s(%s)", size.Div(size, unit).String(), XSizeUintName(unit))
}

func ToInterface(itfc interface{}) interface{} {
	data, _ := json.Marshal(itfc)
	var o interface{}
	json.Unmarshal(data, &o)
	return o
}

// from types.TipsetKey.String to *types.TipsetKey
func Tipsetkey_from_string(k string) *types.TipSetKey {
	if len(k) < 5 {
		return nil
	}
	k = k[1 : len(k)-1]
	ks := strings.Split(k, ",")
	cids := make([]cid.Cid, len(ks))
	for i, e := range ks {
		if id, err := cid.Decode(e); err != nil {
			return nil
		} else {
			cids[i] = id
		}
	}
	tipset_key := types.NewTipSetKey(cids[:]...)
	return &tipset_key
}

func UnmarshalJSON(in interface{}, out interface{}) error {
	in_data, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", string(in_data))

	return json.Unmarshal(in_data, out)
}

func TipsetTime(tipsetTime uint64) string {
	var cstZone = time.FixedZone("CST", 8*3600) // 时区为東8区
	return time.Unix(int64(tipsetTime), 0).In(cstZone).Format("2006-01-02 15:04:05")
}

func IntToPercent(v, total uint64) string {
	if total == 0 || v == 0 {
		return "0.00%"
	}
	return FloatToPercent(float64(v), float64(total))
}

func FloatToPercent(v, total float64) string {
	if total == 0 || v == 0 {
		return "0.00%"
	}
	return fmt.Sprintf("%.3f%%", v*100/total)
}

func StringToFloat(fstr string) float64 {
	f, _ := strconv.ParseFloat(fstr, 64)
	return f
}

func FloatToString(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func BigToPercent(up, down *big.Int) string {
	if up == nil || down == nil {
		return "%0"
	}
	fup, _ := big.NewFloat(0).SetString(up.String())
	fdown, _ := big.NewFloat(0).SetString(down.String())
	fu, _ := fup.Float64()
	fd, _ := fdown.Float64()
	return fmt.Sprintf("%.2f%%", (fu*100)/fd)
}

func Max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

func Min(x, y int64) int64 {
	if x > y {
		return y
	}
	return x
}