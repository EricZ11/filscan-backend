package models

import (
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/config"
	"testing"
)

func init() {
	var conf = beego.AppConfig.String
	conf1, err := config.NewConfig("ini", "../conf/app.conf")
	if err != nil {
		panic(err)
	}
	conf = conf1.String
	host := conf("mongoHost")
	user := conf("mongoUser")
	pass := conf("mongoPass")
	mongoDB := conf("mongoDB")
	mgosession := GetGlobalSession(host, user, pass, mongoDB)
	_, err = mgosession.DatabaseNames()
	if err != nil {
		panic(ps("mongoInit fail:%v", err))
	}
	DB = mongoDB
	fmt.Sprintf("mongoInit success dbName=%v", DB)
}

func TestGetSumGasPrice(t *testing.T) {
	//tests := []struct {
	//	name    string
	//	wantSum float64
	//	wantErr bool
	//}{
	//	// TODO: Add test cases.
	//}
	//for _, tt := range tests {
	//	t.Run(tt.name, func(t *testing.T) {
	//		gotSum, err := GetSumGasPrice()
	//		if (err != nil) != tt.wantErr {
	//			t.Errorf("GetSumGasPrice() error = %v, wantErr %v", err, tt.wantErr)
	//			return
	//		}
	//		if gotSum != tt.wantSum {
	//			t.Errorf("GetSumGasPrice() gotSum = %v, want %v", gotSum, tt.wantSum)
	//		}
	//	})
	//}
	gotSum, err := GetSumGasPrice()
	if err != nil {
		t.Errorf("GetSumGasPrice() error = %v ", err)
		return
	}
	fmt.Println(gotSum)
}

func TestGetSumSize(t *testing.T) {
	//tests := []struct {
	//	name    string
	//	wantSum uint64
	//	wantErr bool
	//}{
	//	// TODO: Add test cases.
	//
	//}
	//for _, tt := range tests {
	//	t.Run(tt.name, func(t *testing.T) {
	//		gotSum, err := GetSumSize()
	//		if (err != nil) != tt.wantErr {
	//			t.Errorf("GetSumSize() error = %v, wantErr %v", err, tt.wantErr)
	//			return
	//		}
	//		if gotSum != tt.wantSum {
	//			t.Errorf("GetSumSize() gotSum = %v, want %v", gotSum, tt.wantSum)
	//		}
	//
	//	})
	//}
	gotSum, err := GetSumSize()
	if err != nil {
		fmt.Sprintf("GetSumSize err %v", err)
	}
	fmt.Println(gotSum)
}

func TestGetMsgCount(t *testing.T) {
	tests := []struct {
		name      string
		wantTotal int
		wantErr   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTotal, err := GetMsgCount()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMsgCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotTotal != tt.wantTotal {
				t.Errorf("GetMsgCount() gotTotal = %v, want %v", gotTotal, tt.wantTotal)
			}
		})
	}
	gotTotal, err := GetMsgCount()
	if err != nil {
		fmt.Sprintf("GetSumSize err %v", err)
	}
	fmt.Println(gotTotal)

}

func TestGetSumGasPriceByMsgMinCreat(t *testing.T) {
	sum, err := GetSumGasPriceByMsgMinCreat(1576496942)
	if err != nil {
		t.Errorf("GetSumGasPrice() error = %v ", err)
		return
	}
	fmt.Println(sum)
}

func TestGetMsgCountByMsgMinCreat(t *testing.T) {
	/*type args struct {
		MinTime int64
	}
	tests := []struct {
		name      string
		args      args
		wantTotal int
		wantErr   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTotal, err := GetMsgCountByMsgMinCreat(tt.args.MinTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMsgCountByMsgMinCreat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotTotal != tt.wantTotal {
				t.Errorf("GetMsgCountByMsgMinCreat() gotTotal = %v, want %v", gotTotal, tt.wantTotal)
			}
		})
	}
	*/
	gotTotal, err := GetMsgCountByMsgMinCreat(1576578555)
	if err != nil {
		t.Errorf("GetSumGasPrice() error = %v ", err)
		return
	}
	fmt.Println(gotTotal)
}

func TestGetSumSizeByMsgMinCreat(t *testing.T) {
	gotTotal, err := GetSumSizeByMsgMinCreat(1577697456)
	if err != nil {
		t.Errorf("GetSumGasPrice() error = %v ", err)
		return
	}
	fmt.Println(gotTotal)

}

func TestGetMsgByAddressFromToMethodCount(t *testing.T) {
	gotTotal, err := GetMsgByAddressFromToMethodCount("t05337", "", "")
	if err != nil {
		t.Errorf("GetMsgByAddressFromToMethodCount() error = %v ", err)
		return
	}
	fmt.Println(gotTotal)
}

func TestGetMsgMethod(t *testing.T) {
	gotTotal, err := GetMsgMethod([]string{"bafy2bzacearkzeohwforku45s4yn5t5dqztmps3tqznk3sesupic6x6jgk5fi"})
	if err != nil {
		t.Errorf("GetMsgByAddressFromToMethodCount() error = %v ", err)
		return
	}
	fmt.Println(gotTotal)
}
