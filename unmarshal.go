// Copyright (c) 2024 Timo Savola
// SPDX-License-Identifier: BSD-3-Clause

package marshal

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"import.name/pan"

	. "import.name/pan/mustcheck"
)

func Unmarshal(sources []any, ptr any, types *Types) error {
	if reflect.TypeOf(ptr).Kind() != reflect.Pointer {
		return errors.New("unmarshal: destination pointer expected")
	}
	if len(sources) == 0 {
		return errors.New("unmarshal: nothing to unmarshal")
	}

	u := &unmarshaler{
		types:   types,
		sources: sources,
		objects: make([]any, len(sources)),
	}
	u.objects[0] = ptr

	src := reflect.ValueOf(u.sources[0])
	dest := reflect.ValueOf(ptr).Elem()

	return pan.Recover(func() {
		u.unmarshal(src, dest)
	})
}

type unmarshaler struct {
	types   *Types
	sources []any
	objects []any
}

func (u *unmarshaler) unmarshal(src, dest reflect.Value) {
	switch dest.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String:
		// TODO: check src kind
		dest.Set(src)

	case reflect.Struct:
		srcType := src.Type()
		if srcType.Kind() != reflect.Map {
			panic(src) // TODO
		}
		if srcType.Key().Kind() != reflect.String {
			panic(src) // TODO
		}
		if srcType.Elem().Kind() != reflect.Interface {
			panic(src) // TODO
		}

		for _, f := range reflect.VisibleFields(dest.Type()) {
			if f.IsExported() {
				v := src.MapIndex(reflect.ValueOf(f.Name))
				if v != (reflect.Value{}) {
					u.unmarshal(v.Elem(), dest.FieldByIndex(f.Index))
				}
			}
		}

	case reflect.Array, reflect.Slice:
		srcType := src.Type()
		if srcType.Kind() != reflect.Slice {
			panic(src) // TODO
		}
		if srcType.Elem().Kind() != reflect.Interface {
			panic(src) // TODO
		}

		n := src.Len()
		if dest.Kind() == reflect.Array && n != dest.Len() {
			panic(src) // TODO
		}
		if dest.Kind() == reflect.Slice {
			dest.Set(reflect.MakeSlice(dest.Type(), n, n))
		}

		for i := range n {
			v := src.Index(i)
			if !v.IsNil() {
				u.unmarshal(v.Elem(), dest.Index(i))
			}
		}

	case reflect.Map:
		destType := dest.Type()
		keyType := destType.Key()
		elemType := destType.Elem()

		srcType := src.Type()
		if srcType.Kind() != reflect.Map {
			panic(src) // TODO
		}
		if srcType.Key().Kind() != keyType.Kind() {
			panic(src) // TODO
		}
		if srcType.Elem().Kind() != reflect.Interface {
			panic(src) // TODO
		}

		if !src.IsNil() {
			dest.Set(reflect.MakeMapWithSize(destType, src.Len()))

			for iter := src.MapRange(); iter.Next(); {
				v := iter.Value()
				if v.IsZero() {
					dest.SetMapIndex(iter.Key(), reflect.Zero(elemType))
				} else {
					tmp := reflect.New(elemType)
					u.unmarshal(v.Elem(), tmp.Elem())
					dest.SetMapIndex(iter.Key(), tmp.Elem())
				}
			}
		}

	case reflect.Interface:
		srcType := src.Type()
		if srcType.Kind() != reflect.Map {
			panic(src) // TODO
		}
		if srcType.Key().Kind() != reflect.String {
			panic(src) // TODO
		}
		if srcType.Elem().Kind() != reflect.Interface {
			panic(src) // TODO
		}
		if src.Len() != 1 {
			panic(src) // TODO
		}

		iter := src.MapRange()
		iter.Next()

		typeName := iter.Key().String()
		t, found := u.types.nameTypes[typeName]
		if !found {
			pan.Panic(fmt.Errorf("unmarshal: type name not registered: %q", typeName))
		}

		tmp := reflect.New(t)
		u.unmarshal(iter.Value().Elem(), tmp.Elem())
		dest.Set(tmp.Elem())

	case reflect.Pointer:
		var index uint64

		switch src.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			index = src.Uint()

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i := src.Int()
			if i < 0 {
				panic(src) // TODO
			}
			index = uint64(i)

		case reflect.Float32, reflect.Float64:
			f := src.Float()
			index = uint64(f)
			if f < 0 || f != float64(index) {
				panic(src) // TODO
			}

		case reflect.String:
			index = Must(strconv.ParseUint(src.String(), 0, 64))

		case reflect.Interface:
			if src.IsNil() {
				dest.Set(src)
				break
			}
			fallthrough
		default:
			panic(src) // TODO
		}

		if index >= uint64(len(u.objects)) {
			panic(src) // TODO
		}

		if x := u.objects[index]; x != nil {
			dest.Set(reflect.ValueOf(x))
			return
		}

		ptr := reflect.New(dest.Type().Elem())
		u.objects[index] = ptr.Interface()
		dest.Set(ptr)
		u.unmarshal(reflect.ValueOf(u.sources[index]), ptr.Elem())

	default:
		pan.Panic(fmt.Errorf("unmarshal: target type not supported: %s", dest.Type()))
	}
}
