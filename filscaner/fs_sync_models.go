package filscaner

import (
	"container/list"
	"filscan_lotus/models"
	"fmt"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/globalsign/mgo/bson"
	"sync"
)

const collection_min_synced_height = "min_synced_height"
const collection_chian_synced_tipset = "synced_tipset"

type models_min_synced_height struct {
	MinHeight uint64 `bson:"min_height"`
}

func models_get_synced_min_height() (uint64, error) {
	ms, c := models.Connect(collection_min_synced_height)
	defer ms.Close()

	min_height := &models_min_synced_height{}

	err := c.Find(nil).One(min_height)
	return min_height.MinHeight, err
}

func models_upsert_synced_height(height uint64) error {
	ms, c := models.Connect(collection_min_synced_height)
	defer ms.Close()

	_, err := c.Upsert(nil, &models_min_synced_height{MinHeight: height})
	return err
}

// 同步状态信息
type fs_synced_tipset struct {
	Key       string `bson:"key"`
	ParentKey string `bson:"parent_key"`
	Height    uint64 `bson:"height"`
}

func (self *fs_synced_tipset) update_with_tipset(tipset *types.TipSet) *fs_synced_tipset {
	if tipset == nil {
		return self
	}
	self.Key = tipset.Key().String()
	self.ParentKey = tipset.Parents().String()
	self.Height = tipset.Height()
	return self
}

func (self *fs_synced_tipset) update_with_parent(tipset *types.TipSet) bool {
	if self.is_my_parent(tipset) {
		self.update_with_tipset(tipset)
		return true
	}
	return false
}

func (self *fs_synced_tipset) update_with_child(tipset *types.TipSet) bool {
	if self.is_my_child_(tipset) {
		self.update_with_tipset(tipset)
		return true
	}
	return false
}

func (self *fs_synced_tipset) is_my_child_(tipset *types.TipSet) bool {
	if self == nil || tipset == nil {
		return false
	}

	fmt.Printf("info:is_my_child, self.key:%s, tipset.parents.key:%s\n", self.Key, tipset.Parents().String())

	return self.Key == tipset.Parents().String()
}

func (self *fs_synced_tipset) is_my_parent(tipset *types.TipSet) bool {
	if self == nil || tipset == nil {
		return false
	}
	return self.ParentKey == tipset.Key().String()
}

func (self *fs_synced_tipset) is_my_child__(tipset *fs_synced_tipset) bool {
	if self == nil || tipset == nil {
		return false
	}
	return (self.Key == tipset.ParentKey && self.Height < tipset.Height)
}

type fs_synced_tipset_path struct {
	Head        *fs_synced_tipset `bson:"head"`
	Tail        *fs_synced_tipset `bson:"tail"`
	upsert_selc bson.M            `bson:"-"`
}

var fs_new_stp = fs_new_synced_tipset_path

func fs_new_synced_tipset_path(head, tail *fs_synced_tipset) *fs_synced_tipset_path {
	path := &fs_synced_tipset_path{head, tail, nil}
	return path.refresh_selc()
}

func (self *fs_synced_tipset_path) try_merge_with(fstp *fs_synced_tipset_path) bool {
	var merged = false
	if self.Head.Height > fstp.Head.Height {
		if self.Tail.ParentKey == fstp.Head.Key || self.Tail.Height <= fstp.Head.Height {
			self.Tail = fstp.Tail
			merged = true
		}
	} else if fstp.Head.Height > self.Head.Height {
		if fstp.Tail.ParentKey == self.Head.Key || fstp.Tail.Height <= self.Head.Height {
			self.Head = fstp.Head
			merged = true
		}
	} else if self.Head.Height == fstp.Head.Height {
		if fstp.Tail.Height < self.Tail.Height {
			self.Tail = fstp.Tail
		}
		merged = true
	}
	return merged
}

func (self *fs_synced_tipset_path) models_del() error {
	ms, c := models.Connect(collection_chian_synced_tipset)
	defer ms.Close()
	return c.Remove(self.upsert_selc)
}

func (self *fs_synced_tipset_path) refresh_selc() *fs_synced_tipset_path {
	self.upsert_selc = bson.M{
		"head.height": self.Head.Height,
		"tail.height": self.Tail.Height}
	return self
}

func (self *fs_synced_tipset_path) models_upsert() error {
	ms, c := models.Connect(collection_chian_synced_tipset)
	defer ms.Close()
	if _, err := c.Upsert(self.upsert_selc, self); err != nil {
		return err
	}
	self.refresh_selc()
	return nil
}

type fs_synced_tipset_path_list struct {
	Path_list *list.List
	mutx      sync.Mutex
}

func models_new_synced_tipset_list() (*fs_synced_tipset_path_list, error) {
	synced := &fs_synced_tipset_path_list{}
	err := synced.models_load()
	return synced, err
}

func (self *fs_synced_tipset_path_list) lock() {
	self.mutx.Lock()
}

func (self *fs_synced_tipset_path_list) unlock() {
	self.mutx.Unlock()
}

func (self *fs_synced_tipset_path_list) push_new_path(tipset *types.TipSet) error {
	self.lock()
	defer self.unlock()

	stp := fs_new_stp((&fs_synced_tipset{}).update_with_tipset(tipset),
		(&fs_synced_tipset{}).update_with_tipset(tipset))

	self.Path_list.PushFront(stp)

	return stp.models_upsert()
}

func (self *fs_synced_tipset_path_list) try_merge_front_with_next(lock bool) *fs_synced_tipset_path {
	if lock {
		self.lock()
		defer self.unlock()
	}

	if e_front := self.Path_list.Front(); e_front != nil {
		if e_next := e_front.Next(); e_next != nil {
			front := e_front.Value.(*fs_synced_tipset_path)
			next := e_next.Value.(*fs_synced_tipset_path)
			if front.try_merge_with(next) {
				self.Path_list.Remove(e_next)
				return next
			}

		}
	}
	return nil
}

func (self *fs_synced_tipset_path_list) models_upsert_front(lock bool) error {
	if lock {
		self.lock()
		defer self.unlock()
	}
	return self.Path_list.Front().Value.(*fs_synced_tipset_path).models_upsert()
}

func (self *fs_synced_tipset_path_list) front_synced_path() bson.M {
	self.lock()
	defer self.unlock()
	if self.Path_list.Len() != 0 {
		return self.Path_list.Front().Value.(*fs_synced_tipset_path).upsert_selc
	}
	return nil
}

func (self *fs_synced_tipset_path_list) front_tail() *fs_synced_tipset {
	self.lock()
	defer self.unlock()
	if self.Path_list.Len() > 0 {
		return self.Path_list.Front().Value.(*fs_synced_tipset_path).Tail
	}
	return nil
}

func (self *fs_synced_tipset_path_list) models_load() error {
	self.lock()
	defer self.unlock()

	ms, c := models.Connect(collection_chian_synced_tipset)
	defer ms.Close()

	var patharr []*fs_synced_tipset_path
	if err := c.Find(nil).Sort("-head.tipset_height").Limit(20).All(&patharr); err != nil {
		return err
	}

	self.Path_list = list.New()

	for _, path := range patharr {
		// 加载的时候, 刷新一下更新要使用的selc
		path.refresh_selc()
		self.Path_list.PushFront(path)
	}

	changed := false
	for merged := self.try_merge_front_with_next(false); merged != nil; {
		changed = true
		if err := merged.models_del(); err != nil {
			return err
		}
	}
	if changed {
		return self.models_upsert_front(false)
	}
	return nil
}

func (self *fs_synced_tipset_path_list) insert_tail_parent(tipset *types.TipSet) bool {
	self.lock()
	defer self.unlock()

	e_f := self.Path_list.Front()

	if e_f == nil {
		e_f = self.Path_list.PushFront(fs_new_synced_tipset_path(
			(&fs_synced_tipset{}).update_with_tipset(tipset),
			(&fs_synced_tipset{}).update_with_tipset(tipset)))
		return true
	}
	return e_f.Value.(*fs_synced_tipset_path).Tail.update_with_parent(tipset)
}

func (self *fs_synced_tipset_path_list) insert_head_child(tipset *types.TipSet) bool {
	self.lock()
	defer self.unlock()

	e_f := self.Path_list.Front()

	if e_f == nil {
		e_f = self.Path_list.PushFront(fs_new_synced_tipset_path(
			(&fs_synced_tipset{}).update_with_tipset(tipset),
			(&fs_synced_tipset{}).update_with_tipset(tipset)))
		return true
	}

	return e_f.Value.(*fs_synced_tipset_path).Head.update_with_child(tipset)
}
