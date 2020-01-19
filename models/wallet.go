package models

type Wallet struct {
	WalletAddr  string  `bson:"wallet_addr" json:"wallet_addr"`
	Balance     float64 `orm:"column(balance);null;digits(50);decimals(18)"`
	GmtCreate   string  `orm:"column(gmt_create);type(datetime);null"`
	GmtModified string  `orm:"column(gmt_modified);type(datetime);null"`
	State       int8    `orm:"column(state);null" description:"1-当前使用 2-过期地址"`
}
