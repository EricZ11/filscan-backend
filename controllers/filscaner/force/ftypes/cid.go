package ftypes

import (
	"sort"

	"github.com/ipfs/go-cid"
)

// CopySortCids 非原址排序
func CopySortCids(src []cid.Cid) []cid.Cid {
	dst := make([]cid.Cid, len(src))
	copy(dst, src)
	SortCids(dst)
	return dst
}

// SortCids 对 cids 进行原址排序
func SortCids(cids []cid.Cid) {
	sort.Slice(cids, func(i, j int) bool {
		return cids[i].KeyString() < cids[j].KeyString()
	})
}
