package encoding

import (
	"reflect"

	"gopkg.in/mgo.v2/bson"
)

func anyInto(v interface{}) (interface{}, error) {
	if qv, can := quickInto(v); can {
		return qv, nil
	}

	vt := reflect.TypeOf(v)

	into, exist := intos[vt]
	if exist {
		return into(v)
	}

	return valueInto(reflect.ValueOf(v))
}

func valueInto(v reflect.Value) (interface{}, error) {
	if !v.IsValid() {
		return nil, nil
	}

	if into, exist := intos[v.Type()]; exist {
		return into(v.Interface())
	}

	vk := v.Kind()
	switch vk {
	case reflect.Interface, reflect.Ptr:
		if v.IsNil() {
			return nil, nil
		}

		return valueInto(v.Elem())

	case reflect.Array, reflect.Slice:
		return sliceInto(v)

	case reflect.Map:
		return mapInto(v)

	case reflect.Struct:
		return structInto(v)

	default:
		return v.Interface(), nil
	}
}

func sliceInto(v reflect.Value) ([]interface{}, error) {
	size := v.Len()
	result := make([]interface{}, 0, size)

	for i := 0; i < size; i++ {
		eleV, err := anyInto(v.Index(i).Interface())
		if err != nil {
			return nil, err
		}

		result = append(result, eleV)
	}

	return result, nil
}

func mapInto(v reflect.Value) (bson.D, error) {
	doc := bson.D{}

	iter := v.MapRange()
	for iter.Next() {
		eleV, err := anyInto(iter.Value().Interface())
		if err != nil {
			return nil, err
		}

		doc = append(doc, bson.DocElem{
			Name:  iter.Key().String(),
			Value: eleV,
		})
	}

	return doc, nil
}

func structInto(v reflect.Value) (bson.D, error) {
	vt := v.Type()
	si, has := structs[vt]
	if !has {
		if err := RegisterStruct(vt); err != nil {
			return nil, err
		}

		si, _ = structs[vt]
	}

	return si.into(v)
}
