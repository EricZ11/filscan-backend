package controllers

import (
	"encoding/json"
	"filscan_lotus/models"
	"filscan_lotus/utils"
	"fmt"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"sort"
	"strconv"
	"sync"
	"time"
)

type Element struct {
	tipset *types.TipSet
	blocks []*BlockAndMsg
}

type BlockAndMsg struct {
	block *models.FilscanBlock
	msg   []*models.FilscanMsg
}

type TipSetQueue interface {
	UpdatePush(e Element) //向队列中添加元素
	Poll() Element        //移除队列中最前面的元素
	Clear() bool          //清空队列
	Size() int            //获取队列的元素个数
	IsEmpty() bool        //判断队列是否是空
	Have() int
	AllElement() []Element
}

type SliceEntry struct {
	element []*Element
	lock    sync.Locker
	isLock  bool
}

func NewQueue() *SliceEntry {
	return &SliceEntry{
		lock: utils.NewHappiLock(false),
	}
}

func (entry *SliceEntry) Lock() {
	entry.lock.Lock()
}

func (entry *SliceEntry) Unlock() {
	entry.lock.Unlock()
}

//向队列中更新/添加元素
//bool 表示是否继续向下 获取新的tipset 置入
func (entry *SliceEntry) UpdatePush(e *Element, parentHeight uint64) bool {
	entry.Lock()
	defer entry.Unlock()

	if index := entry.Have(e); index > 0 {
		if len(entry.element[index].blocks) == len(e.blocks) {
			for _, blocks := range e.blocks {
				for _, elementvalue := range entry.element[index].blocks {
					if blocks.block.Cid == elementvalue.block.Cid {
						continue
					} else {
						entry.element[index] = e //同一高度 原值 与 新值不同，所以依旧向下获取
						return true
					}
				}
			}
			return false //新值 与 原值完全相同，不需要向下获取
		} else {
			entry.element[index] = e //同一高度 原值 与 新值 不同，所以依旧向下获取
			return true
		}
	} else {
		entry.element = append(entry.element, e)
		sort.Slice(entry.element, func(i, j int) bool {
			return entry.element[i].tipset.Height() < entry.element[j].tipset.Height()
		})
	}
	for i := 1; e.tipset.Height()-uint64(i) > parentHeight; i++ { //分叉链:父节点 与当前节点 之前跳高
		entry.DelTipsetByHeight(e.tipset.Height() - uint64(i))
	}
	//fmt.Println(">>>>>>>>>>>>>>>>>>", entry.Size(), "<<<<<<<<<<<<<<<<<<<<<")
	return true
}

//移除队列中最前面的额元素
func (entry *SliceEntry) Poll() *Element {
	entry.Lock()
	defer entry.Unlock()

	if entry.IsEmpty() {
		fmt.Println("queue is empty!")
		return &Element{}
	}
	firstElement := entry.element[0]
	entry.element = entry.element[1:]
	return firstElement
}

//获取前N个元素   并移除
func (entry *SliceEntry) GetHeaderList(num int) []*Element {
	entry.Lock()
	defer entry.Unlock()
	if entry.Size() < num {
		return entry.element
	}
	headerElement := entry.element[:num+1]
	entry.element = entry.element[num+1:]
	return headerElement
}

//获取 e 应该存放的 index   其实可以与Have（待续
func (entry *SliceEntry) RightIndex(e *Element) int {
	entry.Lock()
	defer entry.Unlock()
	if entry.Size() < 1 {
		return 0
	}
	var difference, index int
	for key, value := range entry.element {
		d := int(value.tipset.Height() - e.tipset.Height()) //获取比给定 height 最邻近height 的 index
		if d > 0 && d < difference {
			difference = d
			index = key
		}
	}
	return index
}

func (entry *SliceEntry) Clear() bool {
	entry.Lock()
	defer entry.Unlock()
	if entry.IsEmpty() {
		fmt.Println("queue is empty!")
		return false
	}
	for i := 0; i < entry.Size(); i++ {
		entry.element[i] = &Element{}
	}
	entry.element = nil
	return true
}

//Tipset Size
func (entry *SliceEntry) Size() int {
	entry.Lock()
	defer entry.Unlock()
	return len(entry.element)
}

//block Size
func (entry *SliceEntry) BlockSize() int {
	if !entry.isLock {
		entry.isLock = true
		defer entry.Unlock()
		defer func() {
			entry.isLock = false
		}()
		entry.Lock()
	}
	var blockSize int
	for _, value := range entry.element {
		blockSize += len(value.blocks)
	}
	return blockSize
}

//All block order by height desc
func (entry *SliceEntry) AllBlock() []*BlockAndMsg {
	entry.Lock()
	defer entry.Unlock()
	len := entry.Size()
	var res []*BlockAndMsg
	for key, _ := range entry.element {
		oneTipset := entry.element[len-1-key]
		for _, b := range oneTipset.blocks {
			res = append(res, b)
		}
	}
	return res
}

//All Msg order by height desc
func (entry *SliceEntry) AllMsg() []*models.FilscanMsg {
	entry.Lock()
	defer entry.Unlock()
	len := entry.Size()
	var res []*models.FilscanMsg
	for key, _ := range entry.element {
		oneTipset := entry.element[len-1-key]
		for _, b := range oneTipset.blocks {
			for _, m := range b.msg {
				res = append(res, m)
			}
		}
	}
	return res
}

//msg Size
func (entry *SliceEntry) MsgSize() int {
	entry.Lock()
	defer entry.Unlock()
	var msgSize int
	for _, value := range entry.element {
		for _, b := range value.blocks {
			msgSize += len(b.msg)
		}
	}
	return msgSize
}

func (entry *SliceEntry) IsEmpty() bool {
	entry.Lock()
	defer entry.Unlock()
	if len(entry.element) == 0 {
		return true
	}
	return false
}

//height is have
func (entry *SliceEntry) Have(e *Element) int {
	entry.Lock()
	defer entry.Unlock()
	if entry.Size() < 1 {
		return -1
	}
	for key, value := range entry.element {
		if value.tipset.Height() == e.tipset.Height() {
			return key
		}
	}
	return -1
}

func (entry *SliceEntry) AllElement() []*Element {
	//if !entry.isLock {
	//	entry.isLock = true
	//	defer entry.Unlock()
	//	defer func() {
	//		entry.isLock = false
	//	}()
	//	entry.Lock()
	//}
	return entry.element
}

//获取 tipsetElement begindex , count
func (entry *SliceEntry) TipSetElement(start, end int) []*Element {
	entry.Lock()
	defer entry.Unlock()
	len := entry.Size()
	if len < start {
		return nil
	}
	if start+end >= len {
		return entry.element[start : len-1]
	} else {
		return entry.element[len-1-start-end : len-1-start] //因为[]从小到大排列  获取应该 从尾 向前 推
	}
}

//获取其中 部分 待优化
func (entry *SliceEntry) BlockByIndex(start, end int) []*BlockAndMsg {
	entry.Lock()
	defer entry.Unlock()
	all := entry.AllBlock()
	if len(all) < start {
		return nil
	}
	if start+end >= len(all) {
		return all[start:]
	}
	return all[start : start+end]
}

//BlockByMinerArr sort by height desc
func (entry *SliceEntry) SortBlockByMinerArr(minerArr []string) []*models.FilscanBlock {
	entry.Lock()
	defer entry.Unlock()
	var res []*models.FilscanBlock
	len := entry.Size()
	for i := 0; i < len; i++ {
		for _, b := range entry.element[len-1-i].blocks { // queue is height desc
			for _, miner := range minerArr {
				if b.block.BlockHeader.Miner.String() == miner {
					res = append(res, b.block)
					break // not be other miner
				}
			}
		}
	}
	return res
}

//获取其中 部分 待优化
func (entry *SliceEntry) MsgByIndex(start, end int) []*models.FilscanMsg {
	entry.Lock()
	defer entry.Unlock()
	all := entry.AllMsg()
	if len(all) < start {
		return nil
	}
	if start+end >= len(all) {
		return all[start:]
	}
	return all[start : start+end]
}

func (entry *SliceEntry) MsgByAddressFromToMethodName(address, fromTo, methodName string) []*models.FilscanMsg {
	entry.Lock()
	defer entry.Unlock()
	lenS := entry.Size()
	var res []*models.FilscanMsg
	for key, _ := range entry.element {
		oneTipset := entry.element[lenS-1-key]
		for _, b := range oneTipset.blocks {
			for _, m := range b.msg {
				switch fromTo {
				case "from":
					if m.Message.From.String() == address {
						res = append(res, m)
					}
				case "to":
					if m.Message.To.String() == address {
						res = append(res, m)
					}
				default:
					if m.Message.From.String() == address || m.Message.To.String() == address {
						res = append(res, m)
					}
				}
			}
		}
	}

	if len(methodName) > 0 {
		var tmp []*models.FilscanMsg
		for _, value := range res {
			//methodInt, _ := strconv.Atoi(method)
			if value.MethodName == methodName {
				tmp = append(tmp, value)
			}
		}
		res = tmp
	}
	return res
}

func (entry *SliceEntry) BlockByCid(cid string) *BlockAndMsg {
	entry.Lock()
	defer entry.Unlock()
	//var res []*BlockAndMsg
	for _, value := range entry.element {
		for _, b := range value.blocks {
			if b.block.Cid == cid {
				return b
			}
		}
	}
	return nil
}

func (entry *SliceEntry) MsgByCid(cid string) *models.FilscanMsg {
	entry.Lock()
	defer entry.Unlock()
	//var res []*BlockAndMsg
	for _, value := range entry.element {
		for _, b := range value.blocks {
			for _, m := range b.msg {
				if m.Cid == cid {
					return m
				}
			}
		}
	}
	return nil
}

func (entry *SliceEntry) MsgByBlockCid(blockCids []string) []*models.FilscanMsg {
	entry.Lock()
	defer entry.Unlock()
	var res []*models.FilscanMsg
	for _, cid := range blockCids {
		for _, value := range entry.element {
			for _, b := range value.blocks {
				if b.block.Cid == cid {
					res = append(res, b.msg...)
					break
				}
			}
			break
		}
	}
	return res
}

func (entry *SliceEntry) TipsetByOneHeight(Height uint64) *Element {
	entry.Lock()
	defer entry.Unlock()
	if len(entry.element) < 1 || entry.element[0].tipset.Height() > Height || entry.element[len(entry.element)-1].tipset.Height() < Height {
		return nil
	}
	for _, value := range entry.element {
		if value.tipset.Height() == Height {
			return value
		}
	}
	return nil
}

func (entry *SliceEntry) DelTipsetByHeight(Height uint64) {
	entry.Lock()
	defer entry.Unlock()
	if len(entry.element) < 1 || entry.element[0].tipset.Height() > Height || entry.element[len(entry.element)-1].tipset.Height() < Height {
		return
	}
	for key, value := range entry.element {
		if value.tipset.Height() == Height {
			entry.element = append(entry.element[:key], entry.element[key+1:]...)
		}
	}
	return
}

func (entry *SliceEntry) TipsetByHeight(startHeight, endHeight uint64) []*Element {
	entry.Lock()
	defer entry.Unlock()
	var res []*Element
	for _, value := range entry.element {
		if value.tipset.Height() > endHeight {
			return res
		}
		if value.tipset.Height() >= startHeight && value.tipset.Height() <= endHeight {
			res = append(res, value)
		}
	}
	return res
}

//func (entry *SliceEntry) MsgByBlockCidMethod(blockCid, method string) []*models.FilscanMsg {
//	entry.Lock()
//	defer entry.Unlock()
//	var res []*models.FilscanMsg
//	if len(blockCid) > 1 {
//		for _, value := range entry.element {
//			for _, b := range value.blocks {
//				if b.block.Cid == blockCid {
//					if len(method) > 0 {
//						for _, msg := range b.msg {
//							res = append(res, msg)
//						}
//					} else {
//						res = append(res, b.msg...)
//					}
//				}
//			}
//		}
//	} else {
//		for _, value := range entry.element {
//			for _, b := range value.blocks {
//				if len(method) > 0 {
//					for _, msg := range b.msg {
//						methodInt, _ := strconv.Atoi(method)
//						if msg.Message.Method == uint64(methodInt) {
//							res = append(res, msg)
//						}
//					}
//				} else {
//					res = append(res, b.msg...)
//				}
//			}
//		}
//	}
//	return res
//}

func (entry *SliceEntry) MsgByBlockCidMethodName(blockCid, methodName string) []*models.FilscanMsg {
	entry.Lock()
	defer entry.Unlock()
	var res []*models.FilscanMsg
	if len(blockCid) > 1 {
		for _, value := range entry.element {
			for _, b := range value.blocks {
				if b.block.Cid == blockCid {
					if len(methodName) > 0 {
						for _, msg := range b.msg {
							if msg.MethodName == methodName {
								res = append(res, msg)
							}
						}
					} else {
						res = append(res, b.msg...)
					}
				}
			}
		}
	} else {
		for _, value := range entry.element {
			for _, b := range value.blocks {
				if len(methodName) > 0 {
					for _, msg := range b.msg {
						if msg.MethodName == methodName {
							res = append(res, msg)
						}
					}
				} else {
					res = append(res, b.msg...)
				}
			}
		}
	}
	return res
}

func (entry *SliceEntry) MsgUpdateReceipt(msg []api.Message, msgReceipt []*types.MessageReceipt, height uint64, loop int) {
	entry.Lock()
	defer entry.Unlock()

	if len(entry.element) == 1 {
		return
	}
	go func(ms []api.Message, msR []*types.MessageReceipt, h uint64) {
		oneEle := entry.TipsetByOneHeight(h)
		if loop > 4 { //循环次数 >5 说明height 很可能不在cash中 在数据库
			models.UpdateMsgReceipts(ms, msR, 0)
			return
		}
		if oneEle == nil || oneEle.blocks == nil {
			time.Sleep(3 * time.Second)
			go entry.MsgUpdateReceipt(ms, msR, h, loop+1)
			return
		}
		flag := true
		for _, b := range oneEle.blocks {
			for _, m := range b.msg {
				for k, value := range ms {
					if m.Cid == value.Cid.String() {
						flag = false

						rebyte, _ := json.Marshal(msR[k])
						var returnS models.MsgReceipt
						err := json.Unmarshal(rebyte, &returnS)
						if err != nil {
							fmt.Sprintf("err =%v", err)
						}
						m.GasUsed = msR[k].GasUsed.String()
						m.Return = returnS.Return
						m.ExitCode = strconv.Itoa(int(returnS.ExitCode))
						//m.ExitCode = returnS.ExitCode
					}
				}
			}
		}
		if flag {
			time.Sleep(3 * time.Second)
			go entry.MsgUpdateReceipt(ms, msR, h, loop+1)
			return
		}
	}(msg, msgReceipt, height)
}
