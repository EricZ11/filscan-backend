package utils

import (
	bc "github.com/astaxie/beego/config"
)

var beego_conf bc.Configer

func GetConfiger() bc.Configer {
	if beego_conf == nil {
		var err error
		if beego_conf, err = bc.NewConfig("ini", "conf/app.conf"); err != nil {
			panic(err)
		}
	}
	return beego_conf
}
