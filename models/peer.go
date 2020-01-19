package models

import (
	"encoding/json"
	"github.com/globalsign/mgo"
	"gopkg.in/mgo.v2/bson"
)

type Peer struct {
	PeerId      string  `bson:"peer_id" json:"peer_id"`
	IpAddr      string  `bson:"ip_addr" json:"ip_addr"`
	Ip          string  `bson:"ip" json:"ip"`
	LocationCN  string  `bson:"location_cn" json:"location_cn"`
	LocationEN  string  `bson:"location_en" json:"location_en"`
	Longitude   float64 `bson:"longitude" json:"longitude"`
	Latitude    float64 `bson:"latitude" json:"latitude"`
	GmtCreate   int64   `bson:"gmt_create" json:"gmt_create"`
	GmtModified int64   `bson:"gmt_modified" json:"gmt_modified"`
}

const (
	PeerCollection = "peer"
)

/**
db.peer.ensureIndex({"gmt_modified":-1})
db.peer.ensureIndex({"peer_id":-1},{"unique":true})
db.peer.ensureIndex({"longitude":1,"latitude":1})
*/
func Create_peer_index() {
	ms, c := Connect(PeerCollection)
	defer ms.Close()
	ms.SetMode(mgo.Monotonic, true)

	indexs := []mgo.Index{
		{Key: []string{"peer_id"}, Unique: true, Background: true},
		//{Key: []string{"peer_id"}, Unique: false, Background: true},
		{Key: []string{"gmt_modified"}, Unique: false, Background: true},
		{Key: []string{"longitude", "latitude"}, Unique: false, Background: true},
	}
	for _, index := range indexs {
		if err := c.EnsureIndex(index); err != nil {
			panic(err)
		}
	}
}

func InsertPeer(p *Peer) (err error) {
	p.GmtModified = TimeNow
	p.GmtCreate = TimeNow
	err = Insert(PeerCollection, p)
	return err
}
func InsertPeerMulti(p []*Peer) (err error) {
	var docs []interface{}
	for _, value := range p {
		value.GmtModified = TimeNow
		value.GmtCreate = TimeNow
		docs = append(docs, value)
	}
	err = Insert(PeerCollection, docs...)
	return err
}

func GetPeerByPeerId(peer string) (res *Peer, err error) {
	q := bson.M{"peer_id": peer}
	var r []*Peer
	err = FindAll(PeerCollection, q, nil, &r)
	if err != nil {
		return
	}
	if len(r) < 1 {
		return nil, nil
	} else {
		return r[0], nil
	}
}
func UpdatePeerGmtModifiedByPeerId(peerId string) error {
	q := bson.M{"peer_id": peerId}
	u := bson.M{"$set": bson.M{"gmt_modified": TimeNow}}
	return Update(PeerCollection, q, u)
}

type PeerGroupResult struct {
	ID struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"_id"`
	LocationCn []string `json:"location_cn"`
	LocationEn []string `json:"location_en"`
	PeerID     []string `json:"peer_id"`
	IP         []string `json:"ip"`
}

/**
db.peer.aggregate([
    {
        $group: {
            _id: {
                longitude: "$longitude",
                latitude: "$latitude",
            },
            "total": {
                $sum: 1
            },
            "location": {$push:"$location"},
        }
    }
])
*/

func GetPeerGroup() (ress []*PeerGroupResult, err error) {
	math := bson.M{"$match": bson.M{"gmt_modified": bson.M{"$gte": TimeNow - 60*60*24}}}
	group := bson.M{"$group": bson.M{"_id": bson.M{"longitude": "$longitude", "latitude": "$latitude"}, "location_cn": bson.M{"$push": "$location_cn"}, "location_en": bson.M{"$push": "$location_en"}, "peer_id": bson.M{"$push": "$peer_id"}, "ip": bson.M{"$push": "$ip"}}}

	op := []bson.M{math, group}
	var res []interface{}
	err = AggregateAll(PeerCollection, op, &res)
	if err != nil {
		return
	}
	if len(res) > 365 {
		res = res[0:364]
	}
	b, _ := json.Marshal(res)
	err = json.Unmarshal(b, &ress)
	return
}

func GetActivePeerCountByTime(t int64) (int, error) {
	q := bson.M{"gmt_modified": bson.M{"$gte": t}}
	total, err := FindCount(PeerCollection, q, nil)
	if err != nil {
		return 0, err
	}
	return total, err

}
