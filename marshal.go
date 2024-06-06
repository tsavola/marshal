// Copyright (c) 2024 Timo Savola
// SPDX-License-Identifier: BSD-3-Clause

package marshal

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"

	"import.name/pan"
)

func Marshal(x any, types *Types, ignoreUnsupportedTypes bool) ([]any, error) {
	v := reflect.ValueOf(x)
	if v.Kind() == reflect.Struct {
		return nil, errors.New("marshal: struct passed as value")
	}

	m := &marshaler{
		strict: !ignoreUnsupportedTypes,
		types:  types,
		refs:   make(map[unsafe.Pointer]int),
	}

	if err := pan.Recover(func() {
		if _, ok := m.marshal(v, true); !ok {
			pan.Panic(errors.New("marshal: type not supported"))
		}
	}); err != nil {
		return nil, err
	}

	return m.objects, nil
}

type marshaler struct {
	strict  bool
	types   *Types
	refs    map[unsafe.Pointer]int
	objects []any
}

func (m *marshaler) marshal(v reflect.Value, init bool) (any, bool) {
	switch v.Kind() {
	case reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if v.IsNil() {
			if init {
				m.objects = append(m.objects, nil)
			}
			return nil, true
		}
	}

	switch v.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String:
		if init {
			m.objects = append(m.objects, v.Interface())
		}
		return v.Interface(), true

	case reflect.Struct:
		fields := reflect.VisibleFields(v.Type())
		marshaled := make(map[string]any, len(fields))

		for _, f := range fields {
			if f.IsExported() {
				if x, ok := m.marshal(v.FieldByIndex(f.Index), false); ok && x != nil {
					marshaled[f.Name] = x
				}
			}
		}

		if init {
			m.objects = append(m.objects, marshaled)
		}
		return marshaled, true

	case reflect.Array, reflect.Slice:
		t := reflect.SliceOf(reflect.TypeFor[any]())
		n := v.Len()
		marshaled := reflect.MakeSlice(t, n, n)

		for i := range n {
			x, ok := m.marshal(v.Index(i), false)
			if !ok {
				if i > 0 {
					panic("failed to marshal secondary slice element")
				}
				return nil, false
			}

			if x != nil {
				marshaled.Index(i).Set(reflect.ValueOf(x))
			}
		}

		if init {
			m.objects = append(m.objects, marshaled.Interface())
		}
		return marshaled.Interface(), true

	case reflect.Map:
		keyType := v.Type().Key()
		if !isMapKeyTypeSupported(keyType) {
			if m.strict {
				pan.Panic(fmt.Errorf("marshal: type not supported: %s", v.Type()))
			}
			return nil, false
		}

		elemType := reflect.TypeFor[any]()
		mapType := reflect.MapOf(keyType, elemType)
		marshaled := reflect.MakeMapWithSize(mapType, v.Len())

		for iter := v.MapRange(); iter.Next(); {
			if x, ok := m.marshal(iter.Value(), false); ok {
				if x == nil {
					marshaled.SetMapIndex(iter.Key(), reflect.Zero(elemType))
				} else {
					marshaled.SetMapIndex(iter.Key(), reflect.ValueOf(x))
				}
			}
		}

		if init {
			m.objects = append(m.objects, marshaled.Interface())
		}
		return marshaled.Interface(), true

	case reflect.Interface:
		v := v.Elem()
		t := v.Type()

		name, found := m.types.typeNames[t]
		if !found {
			pan.Panic(fmt.Errorf("marshal: type not registered: %s", t))
		}

		x, ok := m.marshal(v, false)
		if !ok {
			panic("failed to marshal registered type")
		}

		marshaled := map[string]any{name: x}
		if init {
			m.objects = append(m.objects, marshaled)
		}
		return marshaled, true

	case reflect.Pointer:
		ptr := v.UnsafePointer()
		if index, found := m.refs[ptr]; found {
			return index, true
		}

		index := len(m.objects)
		m.refs[ptr] = index
		m.objects = append(m.objects, nil) // Placeholder.

		if x, ok := m.marshal(v.Elem(), false); ok {
			m.objects[index] = x
			return index, true
		}

		m.objects = m.objects[:index]
		return nil, false

	default:
		if m.strict {
			pan.Panic(fmt.Errorf("marshal: type not supported: %s", v.Type()))
		}
		return nil, false
	}
}
