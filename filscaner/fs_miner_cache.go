package filscaner

import (
	fspt "filscan_lotus/filscanproto"
	"filscan_lotus/models"
	. "filscan_lotus/utils"
	"fmt"
	"github.com/globalsign/mgo"
	"math/big"
	"sync"
	"time"
)

const MAX_MINER_CACHE_COUNT = int64(6)

func new_MinerStateAtTipset(addr string, time int64) *models.MinerStateAtTipset {
	return &models.MinerStateAtTipset{
		MinerAddr:         addr,
		Power:             models.NewBigintFromInt64(0),
		TotalPower:        models.NewBigintFromInt64(0),
		ProvingSectorSize: models.NewBigintFromInt64(0),
		MineTime:          uint64(time),
	}
}

func (fs *Filscaner) init_miners_caches() error {
	fs.Printf("››››››››››››››››››››init_miner_states››››››››››››››››››››\n")
	begin_time := time.Now()
	defer func() {
		second := uint64(time.Since(begin_time).Seconds())
		fs.Printf("››››››››››››››››init_miner_states finished, time_used=%d(s)!!!››››››››››››››››\n", second)
	}()

	fs.miner_state_chan = make(chan *models.MinerStateAtTipset, 256)

	session, db := models.Copy()
	defer session.Close()

	c := db.C(models.MinerCollection)

	var (
		time_now       = time.Now().Unix()
		max_cache_size = int64(24)
		time_diff      = int64(3600)
		time_start     = time_now - (time_now % time_diff) - (time_diff * (max_cache_size - 2))
	)

	miners, _, err := models_miner_top_power(c, time_now, 0, MAX_MINER_CACHE_COUNT)
	if err != nil && err != mgo.ErrNotFound {
		fs.Printf("error, load_top_power_miner failed, message:%s\n", err.Error())
		return err
	}

	fs.miner_cache24h = (&fs_miner_cache{}).init(time_diff, time_start, MAX_MINER_CACHE_COUNT, max_cache_size)

	if false { // this is a testing for time_to_index
		time_now = 23
		max_cache_size = int64(3)
		time_diff = int64(5)
		time_start = time_now - (time_now % time_diff) - (time_diff * (max_cache_size - 2))

		cache := (&fs_miner_cache{}).init(time_diff, time_start, MAX_MINER_CACHE_COUNT, max_cache_size)
		for i := int64(14); i < 27; i++ {
			index, ofst := cache.time_to_index(i)
			var want_index = int64(0)
			var want_ofseted = false
			if i <= 15 {
				want_index = 2
				want_ofseted = false
			} else if i > 15 && i <= 20 {
				want_index = 1
				want_ofseted = false
			} else if i > 20 && i <= 25 {
				want_index = 0
				want_ofseted = false
			} else if i > 25 {
				want_index = 0
				want_ofseted = true
			}

			result := "test ‹success›"
			if want_index != index || want_ofseted != ofst {
				result = "test ‹failed›"
			}
			fmt.Printf("test result:%s, time_now=%d, index=%d,is_ofst=%t, want_index=%d, want_ofsetd=%t\n",
				result, i, index, ofst, want_index, want_ofseted)
		}
	}

	if err := fs.miner_cache24h.models_set_index_and_load_histroy(c, miners, 0, true); err != nil {
		fs.Printf("error, miner_cache.load_history failed, message:%s\n", err.Error())
		return err
	}

	time_diff = 86400
	max_cache_size = 30
	time_start = time_now - (time_now % time_diff) - (time_diff * (max_cache_size - 1))

	fs.miner_cache1day = (&fs_miner_cache{}).init(time_diff, time_start, MAX_MINER_CACHE_COUNT, max_cache_size)
	if err := fs.miner_cache1day.models_set_index_and_load_histroy(c, miners, 0, true); err != nil {
		fs.Printf("error, miner_cache.load_history failed, message:%s\n", err.Error())
		return err
	}

	return nil
}

type fs_miner_cache struct {
	// miner_address -> miner_state
	// miners          map[string]([]*models.MinerStateAtTipset)
	miners map[string][]*models.MinerStateAtTipset

	max_cached_size int64
	max_miner_count int64

	recent_refresh_time int64

	min string
	max string

	mutx sync.Mutex

	time_duration int64
	start_time    int64
}

func (this *fs_miner_cache) init(time_duration, start_time, miner_count, cache_size int64) *fs_miner_cache {
	this.miners = make(map[string][]*models.MinerStateAtTipset)
	this.time_duration = time_duration
	this.start_time = start_time
	this.max_miner_count = miner_count
	this.max_cached_size = cache_size

	if false {
		index, ofst := this.time_to_index(time.Now().Unix())
		fmt.Println(index, ofst)
		index, ofst = this.time_to_index(this.start_time - 1)
		fmt.Println(index, ofst)
	}
	return this
}

// 返回值:bool, 是否所有数据为nil, 都为伪造出来的
func (this *fs_miner_cache) index(index int) ([]*fspt.MinerState, bool) {
	if index < 0 || int64(index) >= this.max_cached_size {
		return nil, false
	}
	this.lock()
	defer this.unlock()

	size := len(this.miners)
	stats := make([]*fspt.MinerState, size+1)

	var (
		max_total = big.NewInt(0)
		other     = big.NewInt(0)
	)

	i := 0

	var miner_state *fspt.MinerState

	var all_is_nil = true

	for _, v := range this.miners {
		state := v[index]
		if state == nil {
			// time := this.start_time + this.time_duration*(this.max_cached_size-int64(index)-1)
			// state = new_MinerStateAtTipset(v[0].MinerAddr, time)
			miner_state = &fspt.MinerState{
				Address:      v[0].MinerAddr,
				Power:        "0",
				PowerPercent: "0.00%"}
		} else {
			all_is_nil = false
			miner_state = state.State()
			other.Add(other, state.Power.Int)
			if max_total.Cmp(state.TotalPower.Int) < 0 {
				max_total.Set(state.TotalPower.Int)
			}
		}
		stats[i] = miner_state
		i++
	}

	other.Sub(max_total, other)

	power_str := "0"
	power_percent_str := "0.00%"

	if other.Cmp(big.NewInt(0)) > 0 {
		power_str = XSizeString(other)
		power_percent_str = BigToPercent(other, max_total)
	}

	stats[i] = &fspt.MinerState{
		Address:      "other",
		Power:        power_str,
		PowerPercent: power_percent_str,
	}

	return stats, all_is_nil
}

func (this *fs_miner_cache) display(address string) {
	fmt.Printf("------------%s power information----------\n", address)
	if arr, exist := this.miners[address]; exist {
		for index, miner := range arr {
			if miner == nil {
				continue
			}
			fmt.Printf("index:%d, power:%s\n", index, ToXSize(miner.Power.Int, TB))
		}
	}
	fmt.Printf("------------------------\n\n")
}

func (this *fs_miner_cache) models_set_index_and_load_histroy(c *mgo.Collection, miners []*models.MinerStateAtTipset, index int64, lock bool) error {
	start_time_models_set_index_load_history := time.Now()
	defer func() {
		fmt.Printf(" models_set_index_and_load_histroy, used_time = %d(s)\n",
			int(time.Since(start_time_models_set_index_load_history).Seconds()))
	}()

	if len(miners) == 0 {
		return nil
	}

	if lock {
		this.lock()
		defer this.unlock()
	}

	this.set_miners_at_index(miners, index)

	slc_miners := SlcObjToSlc(miners, "MinerAddr").([]string)

	time_at := this.start_time
	var start uint64

	if c == nil {
		session, db := models.Copy()
		c = db.C(models.MinerCollection)
		defer session.Close()
	}

	for idex := this.max_cached_size - 1; idex > index; idex-- {
		// start_time_models_miner_state_at_time := time.Now()

		if idex == this.max_cached_size-1 {
			start = 0
		} else {
			start = uint64(time_at - this.time_duration)
		}

		miner_at_tipsets, err := Models_miner_state_in_time(c, slc_miners, uint64(time_at), start)

		// fmt.Printf("Models_miner_state_in_time, index=%d, used time = %.3f(s)\n",
		// 	idex, time.Since(start_time_models_miner_state_at_time).Seconds())

		if err != nil {
			if err == mgo.ErrNotFound {
				time_at += this.time_duration
				continue
			}
			return err
		}

		this.set_miners_at_index(miner_at_tipsets, idex)

		time_at += this.time_duration
	}

	return nil
}

func (this *fs_miner_cache) lock() {
	this.mutx.Lock()
}

func (this *fs_miner_cache) unlock() {
	this.mutx.Unlock()
}

func (this *fs_miner_cache) next_refresh_time() int64 {
	return this.start_time + this.max_cached_size*this.time_duration
}

func (this *fs_miner_cache) time_to_index(time int64) (int64, bool) {
	offseted := false

	if time <= this.start_time {
		return this.max_cached_size - 1, false
	} else if time > (this.start_time + ((this.max_cached_size - 1) * this.time_duration)) {
		return 0, true
	}

	diff := time - this.start_time
	diff += (this.time_duration - 1)
	index := diff / this.time_duration
	index = this.max_cached_size - 1 - index

	return index, offseted
}

func (this *fs_miner_cache) do_offset() {
	for _, miner_states := range this.miners {
		for index := this.max_cached_size - 1; index > 0; index-- {
			miner_states[index] = miner_states[index-1]
		}
	}
	this.start_time += this.time_duration
}

func (this *fs_miner_cache) update(in *models.MinerStateAtTipset) error {
	this.lock()
	defer this.unlock()
	// todo : checkout why in day duration, first update, ofseted is 'true'
	index, ofsted := this.time_to_index(int64(in.MineTime))

	fmt.Printf("››››››miner state at tipset is going to update:%s, time_to_index=%d, in_hours=%.3f‹‹‹‹‹‹\n",
		in.MinerAddr, index, (float64(time.Now().Unix()-int64(in.MineTime)) / 3600))

	if ofsted && index == 0 {
		this.do_offset()
	}

	if miners, exist := this.miners[in.MinerAddr]; exist { // 如果已经存在
		if true {
			this.set_miners_at_index([]*models.MinerStateAtTipset{in}, index)
		} else {
			if miners[index] == nil || miners[index].MineTime < in.MineTime {
				miners[index] = in
				for i := index - 1; i > 0; i-- {
					if miners[i] == nil {
						miners[i] = miners[i+1]
					}
				}
			}
		}
	} else {
		if int64(len(this.miners)) < this.max_miner_count {
			if err := this.models_set_index_and_load_histroy(nil,
				[]*models.MinerStateAtTipset{in}, index, false); err != nil {
				return err
			}
		} else { // 检查最低算力是否小于in_miner的算力
			min_power_miner := this.miners[this.min][0]
			if min_power_miner.Power.Cmp(in.Power.Int) < 0 {
				// check to_insert_miner, if exist a newer state in database, do nothing
				if index != 0 && models_miner_state_exist_newer(in.MinerAddr, int64(in.MineTime)) {
					return nil
				}
				if err := this.models_set_index_and_load_histroy(nil,
					[]*models.MinerStateAtTipset{in}, index, false); err != nil {
					return err
				}
			} // else // nothing is needed to do
		}
	}

	// index>0, that means it's a history miner_state, nothing is need to do.
	return nil
}

func (this *fs_miner_cache) refresh_min_max(lock bool) {
	if lock {
		this.lock()
		defer this.unlock()
	}

	var min, max *models.MinerStateAtTipset
	for _, v := range this.miners {
		if min == nil || v[0].Power.Cmp(min.Power.Int) < 0 {
			min = v[0]
		}
		if max == nil || v[0].Power.Cmp(max.Power.Int) > 0 {
			max = v[0]
		}
	}
	if min != nil {
		this.min = min.MinerAddr
	}
	if max != nil {
		this.max = max.MinerAddr
	}
}

func (this *fs_miner_cache) set_miners_at_index(miners []*models.MinerStateAtTipset, index int64) {
	in_miners := SlcToMap(miners, "MinerAddr", true).(map[string]*models.MinerStateAtTipset)

	var arr []*models.MinerStateAtTipset

	for ink, inv := range in_miners {
		var exist = false
		if arr, exist = this.miners[ink]; exist {
			if arr[index] == nil || arr[index].MineTime < inv.MineTime {
				arr[index] = inv
			}
			for i := index - 1; i >= 0; i-- {
				if arr[i] != nil && arr[i].MineTime >= inv.MineTime {
					break
				}
				arr[i] = inv
			}
		} else { // not exist
			if len(this.miners) < int(this.max_miner_count) {
				arr = make([]*models.MinerStateAtTipset, this.max_cached_size)
				arr[index] = inv
				this.miners[ink] = arr
				for i := index - 1; i >= 0; i-- {
					arr[i] = arr[i+1]
				}
			} else {
				arr = this.miners[this.min]
				// check and replace min power miner state
				if arr[0].Power.Cmp(inv.Power.Int) < 0 {
					arr[index] = inv
					for i := index - 1; i >= 0; i-- {
						arr[i] = arr[i+1]
					}
					for i := index + 1; i < this.max_cached_size; i++ {
						arr[i] = nil
					}
					this.miners[inv.MinerAddr] = arr
					delete(this.miners, this.min)
				}
			}
			this.refresh_min_max(false)
		}
	}

	if false {
		if index == this.max_cached_size-1 {
			return
		}
		for k, v := range this.miners {
			if _, exist := in_miners[k]; exist {
				continue
			}
			if v[index+1] != nil && (v[index] == nil || v[index].MineTime < v[index].MineTime) {
				v[index] = v[index+1]
			}
		}
	}
}

// func (this *fs_miner_cache) insert_first_miners(miners []*models.MinerStateAtTipset) {
// 	mminers := SlcToMap(miners, "MinerAddr", true).(map[string]*models.MinerStateAtTipset)
// 	if int64(len(this.miners)) >= this.max_miner_count {
// 		return
// 	}
//
// 	for addr, m := range mminers {
// 		stats := make([]*models.MinerStateAtTipset, this.max_cached_size)
// 		this.miners[addr] = stats
// 		stats[0] = m
// 	}
// }
