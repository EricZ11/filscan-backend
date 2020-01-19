package utils

import (
	"fmt"
	"os"
	"strings"
	"sync"
)


var file *os.File
var mutx sync.Mutex

func init() {
	var err error
	file, err = os.OpenFile("./log/ss.log", os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		fmt.Println("open file error:", err)
		return
	}
}

// TODO: use log4go to replace this sample log..
func Printf(prefix string, fmts string, args ...interface{}) {
	mutx.Lock()
	defer mutx.Unlock()

	if prefix = strings.Trim(prefix, " "); prefix != "" {
		fmts = "%s:" + fmts
		args = append([]interface{}{prefix}, args[:]...)
	}

	if l := len(fmts); fmts[l-1] != '\n' {
		fmts += "\n"
	}

	message := fmt.Sprintf(fmts, args[:]...)
	fmt.Printf(message)
	if file != nil {
		fmt.Fprintf(file, message)
	}
}
