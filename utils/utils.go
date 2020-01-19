
package utils

import (
	"fmt"
	"github.com/filecoin-project/lotus/build"
	"math"
	"math/big"
)

const PrecisionDefault = float64(0.00001)

var BlocksPerEpoch = big.NewInt(build.BlocksPerEpoch)

func ToFil(v *big.Int) float64 {
	fbig, _ := big.NewFloat(0).SetString(v.String())
	fv, _ := fbig.Float64()
	return TruncateNaive(fv/build.FilecoinPrecision, PrecisionDefault)
}

func ToFilStr(v *big.Int) string {
	value := ToFil(v)
	return fmt.Sprintf("%.4f", value)
}


func TruncateNaive(f float64, unit float64) float64 {
	x := f / unit
	return math.Trunc(x) * unit
}

func to_xsize(power *big.Int, x XSIZE) float64 {
	fw := big.NewFloat(0)
	fw.SetString(power.String())

	ftbsize := big.NewFloat(0)
	ftbsize.SetString(x.String())

	v1, _ := fw.Float64()
	v2, _ := ftbsize.Float64()

	return TruncateNaive(v1/v2, 0.01)
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
	// var x XSIZE
	// if size.Cmp(TB) > 0 {
	// 	x = TB
	// } else {
	// 	x = GB
	// }
	// return fmt.Sprintf("%.2f%s", to_xsize(size, x), XSizeUintName(x))
}
