package utils

import (
	"reflect"
	"sync"
)

func maparr(in interface{}) interface{} {
	rin := reflect.ValueOf(in)
	rout := reflect.MakeSlice(reflect.SliceOf(rin.Type().Elem()), rin.Len(), rin.Len())
	var i int

	it := rin.MapRange()
	for it.Next() {
		rout.Index(i).Set(it.Value())
		i++
	}

	return rout.Interface()
}

func kmaparr(in interface{}) interface{} {
	rin := reflect.ValueOf(in)
	rout := reflect.MakeSlice(reflect.SliceOf(rin.Type().Key()), rin.Len(), rin.Len())
	var i int

	it := rin.MapRange()
	for it.Next() {
		rout.Index(i).Set(it.Key())
		i++
	}

	return rout.Interface()
}

// map[k]v => []func() (k, v)
func kvmaparr(in interface{}) interface{} {
	rin := reflect.ValueOf(in)

	t := reflect.FuncOf([]reflect.Type{}, []reflect.Type{
		rin.Type().Key(),
		rin.Type().Elem(),
	}, false)

	rout := reflect.MakeSlice(reflect.SliceOf(t), rin.Len(), rin.Len())
	var i int

	it := rin.MapRange()
	for it.Next() {
		k := it.Key()
		v := it.Value()

		rout.Index(i).Set(reflect.MakeFunc(t, func(args []reflect.Value) (results []reflect.Value) {
			return []reflect.Value{k, v}
		}))
		i++
	}

	return rout.Interface()
}

func par(concurrency int, arr interface{}, f interface{}) {
	throttle := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	varr := reflect.ValueOf(arr)
	l := varr.Len()

	rf := reflect.ValueOf(f)

	wg.Add(l)
	for i := 0; i < l; i++ {
		throttle <- struct{}{}

		go func(i int) {
			defer wg.Done()
			defer func() {
				<-throttle
			}()
			rf.Call([]reflect.Value{varr.Index(i)})
		}(i)
	}

	wg.Wait()
}

func field_exsit(tin reflect.Type, field string) (reflect.Type, bool) {
	var kind reflect.Kind
for_exit:
	for {
		kind = tin.Kind()
		switch kind {
		case reflect.Interface:
			tin = reflect.TypeOf(tin)
		case reflect.Ptr:
			tin = tin.Elem()
		default:
			break for_exit
		}
	}

	if kind == reflect.Struct {
		if f, exist := tin.FieldByName(field); exist {
			return f.Type, true
		}
	}

	return nil, false
}

func in_value(vin reflect.Value) reflect.Value {
	for {
		if vin.Kind() == reflect.Interface {
			vin = reflect.ValueOf(vin)
		} else if vin.Kind() == reflect.Ptr {
			vin = vin.Elem()
		} else {
			return vin
		}
	}
}

func field_value(vin reflect.Value, field string) reflect.Value {
	vin = in_value(vin)
	kind := vin.Kind()

	var vout reflect.Value

	if kind == reflect.Struct {
		vout = vin.FieldByName(field)
	} else if kind == reflect.Map {
		ks := vin.MapKeys()
		for _, k := range ks {
			if k.String() == field {
				vout = vin.MapIndex(k)
				break
			}
		}
	}

	return vout
}

func SlcObjToSlc(in interface{}, field_name string) interface{} {
	vin := in_value(reflect.ValueOf(in))
	if vin.Kind() != reflect.Slice && vin.Kind() != reflect.Array {
		return nil
	}

	ft, exist := field_exsit(vin.Type().Elem(), field_name)
	if !exist {
		return nil
	}

	slice_size := vin.Len()

	out := reflect.MakeSlice(reflect.SliceOf(ft), slice_size, slice_size)

	for i := 0; i < slice_size; i++ {
		out.Index(i).Set(field_value(vin.Index(i), field_name))
	}

	return out.Interface()
}

func SlcToMap(in interface{}, kf string, use_value bool) interface{} {
	vin := in_value(reflect.ValueOf(in))
	if vin.Kind() != reflect.Slice && vin.Kind() != reflect.Array {
		return nil
	}

	var kt reflect.Type
	var vt reflect.Type
	var is = interface{}(struct{}{})

	ele := vin.Type().Elem()

	if kf != "" {
		var exist bool
		if kt, exist = field_exsit(ele, kf); !exist {
			return nil
		}
	} else {
		kt = ele
	}

	if use_value {
		vt = ele
	} else {
		vt = reflect.TypeOf(is)
	}

	out := reflect.MakeMap(reflect.MapOf(kt, vt))

	slice_size := vin.Len()

	var vk, vv reflect.Value

	for i := 0; i < slice_size; i++ {
		vv = vin.Index(i)
		if kf != "" {
			vk = field_value(vv, kf)
		} else {
			vk = vv
		}

		if !use_value {
			vv = reflect.ValueOf(is)
		}

		out.SetMapIndex(vk, vv)
	}

	return out.Interface()
}
