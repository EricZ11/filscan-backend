package filscaner

import (
	"encoding/json"
	"filscan_lotus/models"
	"fmt"
	"github.com/filecoin-project/go-address"
	"go4.org/sort"
	"math"
	"math/big"
	"strconv"
	"sync"
	"time"
)

type sort_function func(a, b *models.MinerStateInTipset) bool

type Miners struct {
	miners        []*models.MinerStateInTipset
	address_index map[string]*models.MinerStateInTipset
	less          func(a, b *models.MinerStateInTipset) bool

	maxsize uint64

	mutx sync.Mutex
}

func new_sorted_miners(maxsize uint64) *Miners {
	return &Miners{
		miners:        make([]*models.MinerStateInTipset, maxsize),
		address_index: make(map[string]*models.MinerStateInTipset),
		less:          sort_by_power,
		maxsize:       maxsize,
	}
}

func (m *Miners) Len() int {
	m.mutx.Lock()
	defer m.mutx.Unlock()
	return len(m.miners)
}

func (m *Miners) Swap(i, j int) {
	m.mutx.Lock()
	defer m.mutx.Unlock()
	m.miners[i], m.miners[j] = m.miners[j], m.miners[i]
}

func (m *Miners) Less(i, j int) bool {
	m.mutx.Lock()
	defer m.mutx.Unlock()
	return m.less(m.miners[i], m.miners[j])
}

func (m *Miners) sort(sf sort_function, lockme bool) {
	if lockme {
		m.mutx.Lock()
		defer m.mutx.Unlock()
	}

	m.less = sf
	if m.less == nil {
		m.less = sort_by_power
	}
	sort.Sort(sort.Reverse(m))
}

func (m *Miners) exist(address address.Address) bool {
	m.mutx.Lock()
	defer m.mutx.Unlock()
	_, isok := m.address_index[address.String()]
	return isok
}

func (m *Miners) search(address address.Address) *models.MinerStateInTipset {
	m.mutx.Lock()
	defer m.mutx.Unlock()
	miner, exist := m.address_index[address.String()]
	if !exist {
		return nil
	}
	tmpminer := *miner
	return &tmpminer
}

func (m *Miners) update(in *models.MinerStateInTipset) {
	m.mutx.Lock()
	defer m.mutx.Unlock()

	miner, exist := m.address_index[in.MinerAddr]
	length := uint64(len(m.miners))

	if !exist {
		tmp_miner := *in
		// 如果小于最大size直接插入到末尾, 然后排序
		if length < m.maxsize {
			length = length - 1
		} else {
			// 比最小的算力还小, 就不用放近前maxsize的队列中了
			if m.miners[m.maxsize-1].Power.Cmp(in.Power.Int) > 0 {
				return
			}
		}

		m.miners[length] = &tmp_miner
	} else {
		*miner = *in
	}

	m.sort(nil, false)
}

func (m *Miners) get_miner(offset, count uint64) ([]*models.MinerStateInTipset, uint64) {
	m.mutx.Lock()
	defer m.mutx.Unlock()

	length := uint64(len(m.miners))
	if offset >= length {
		return nil, length
	}

	if offset+count > length {
		count = length - offset
	}

	copyed := make([]*models.MinerStateInTipset, count)
	for index, _ := range copyed {
		tmp := *(m.miners[index+int(offset)])
		copyed[index] = &tmp
	}

	return copyed, length
}

func sort_by_power(a, b *models.MinerStateInTipset) bool {
	return a.Power.Cmp(b.Power.Int) > 0
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

type MinerStateRecord struct {
	Id     string                     `bson:"_id" json:"id"`
	Record *models.MinerStateInTipset `bson:"record" json:"record"`
}

type MinerStateRecordInterface struct {
	Id     string      `bson:"_id" json:"id"`
	Record interface{} `bson:"record" json:"record"`
}

type MinerIncreasedPowerRecord struct {
	IncreasedPower uint64                     `bson:"increased_power" json:"increased_power"`
	Record         *models.MinerStateInTipset `bson:"record" json:"record"`
}

type MinerBlockRecord struct {
	Blockcount uint64                     `bson:"block_count" json:"block_count"`
	Record     *models.MinerStateInTipset `bson:"record" json:"record"`
}

type MinedBlock struct {
	Miner      string `bson:"miner" json:"miner"`
	BlockCount uint64 `bson:"mined_block_count" json:"mined_block_count"`
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