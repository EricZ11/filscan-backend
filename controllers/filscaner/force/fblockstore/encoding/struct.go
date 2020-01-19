package encoding

import (
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/mgo.v2/bson"
)

var structs = map[reflect.Type]structInfo{}

type structInfo struct {
	fields []fieldInfo
}

func (s *structInfo) into(v reflect.Value) (bson.D, error) {
	doc := bson.D{}

	for _, field := range s.fields {
		fv, err := valueInto(v.FieldByIndex(field.Index))
		if err != nil {
			return nil, err
		}

		if !field.Anonymous {
			doc = append(doc, bson.DocElem{
				Name:  field.BsonName,
				Value: fv,
			})
			continue
		}

		// map or struct
		if innerDoc, ok := fv.(bson.D); ok {
			doc = append(doc, innerDoc...)
		} else {
			doc = append(doc, bson.DocElem{
				Name:  field.BsonName,
				Value: fv,
			})
		}
	}

	return doc, nil
}

type fieldInfo struct {
	BsonName string
	reflect.StructField
}

// RegisterStruct parse and register struct info
func RegisterStruct(vt reflect.Type) error {
	if _, has := structs[vt]; has {
		return nil
	}

	vk := vt.Kind()
	switch vk {
	case reflect.Ptr:
		return RegisterStruct(vt.Elem())

	case reflect.Struct:

	default:
		return fmt.Errorf("expected type of struct or ptr, get %s", vt.String())
	}

	fnum := vt.NumField()
	fields := make([]fieldInfo, 0, fnum)

	for i := 0; i < fnum; i++ {
		f := vt.Field(i)

		// private field
		if f.PkgPath != "" {
			continue
		}

		bsonName := f.Name
		if tag := f.Tag.Get("bson"); tag != "" {
			if tag == "-" {
				continue
			}

			pieces := strings.Split(tag, ",")
			if len(pieces[0]) > 0 {
				bsonName = pieces[0]
			}
		}

		RegisterStruct(f.Type)

		fields = append(fields, fieldInfo{
			BsonName:    bsonName,
			StructField: f,
		})
	}

	structs[vt] = structInfo{
		fields: fields,
	}

	return nil
}
