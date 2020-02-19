package main

import (
	"filscan_lotus/filscaner"
	"time"
)

func main() {
	testing_miner_power_in_time()
}

func testing_miner_power_in_time() {
	miners := []string{"t01005", "t09999", "t017504", "t01210", "t01475", "t01346"}
	time_now := time.Now()
	_, err := filscaner.Models_miner_state_in_time(nil, miners, 0)
	err = err
	// Models_miner_state_in_time(nil, miners, 0)
	// fscaner.init_miners_cache()

	fscaner.Printf("use time := %.2f\n", time.Since(time_now).Seconds())

}
