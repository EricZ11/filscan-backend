package encoding

import (
	"encoding/json"
	"net/url"
	"reflect"
	"time"

	"gopkg.in/mgo.v2/bson"
)

func init() {
	RegisterInto(typeOrderKey, asIs)
}

var (
	intos = map[reflect.Type]Into{}

	typeOrderKey = reflect.TypeOf(bson.MinKey)
)

// RegisterInto 注册特定转换函数
func RegisterInto(v reflect.Type, into Into) {
	if _, has := intos[v]; !has {
		intos[v] = into
	}
}

var asIs = Into(func(v interface{}) (interface{}, error) {
	return v, nil
})

// Into 转换函数
type Into = func(interface{}) (interface{}, error)

var _ bson.Getter = (*MarshalWrapper)(nil)

// MarshalWrap wraps a specific value
func MarshalWrap(v interface{}) *MarshalWrapper {
	return &MarshalWrapper{
		inner: v,
	}
}

// MarshalWrapper wrapper struct
type MarshalWrapper struct {
	inner interface{}
}

// GetBSON impl bson.Getter
func (bw *MarshalWrapper) GetBSON() (interface{}, error) {
	return anyInto(bw.inner)
}

func quickInto(v interface{}) (interface{}, bool) {
	if _, ok := v.(bson.Getter); ok {
		return v, true
	}

	switch v.(type) {
	case nil:

	case string, []byte:

	case *string:

	case int8, int16, int32, int64, int:

	case *int8, *int16, *int32, *int64, *int:

	case uint8, uint16, uint32, uint:

	case *uint8, *uint16, *uint32, *uint:

	case bool:

	case *bool:

	case float32, float64:

	case *float32, *float64:

	case bson.ObjectId, bson.Raw, bson.M,
		bson.D, bson.DocElem,
		bson.RawD, bson.RawDocElem,
		bson.Binary, bson.DBPointer,
		bson.Symbol, bson.MongoTimestamp, bson.JavaScript, bson.RegEx:

	case *bson.ObjectId, *bson.Raw, *bson.M,
		*bson.D, *bson.DocElem,
		*bson.RawD, *bson.RawDocElem,
		*bson.Binary, *bson.DBPointer,
		*bson.Symbol, *bson.MongoTimestamp, *bson.JavaScript, *bson.RegEx:

	case time.Time, url.URL, json.Number:

	case *time.Time, *url.URL, *json.Number:

	default:
		return nil, false
	}

	return v, true
}
