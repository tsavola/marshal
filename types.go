// Copyright (c) 2024 Timo Savola
// SPDX-License-Identifier: BSD-3-Clause

package marshal

import (
	"errors"
	"fmt"
	"reflect"
)

type TypeParam struct {
	name string
	t    reflect.Type
}

func Type(name string, value any) TypeParam {
	return TypeParam{name, reflect.ValueOf(value).Type()}
}

func TypeName(value any) TypeParam {
	t := reflect.ValueOf(value).Type()
	return TypeParam{t.Name(), t}
}

type Types struct {
	typeNames map[reflect.Type]string
	nameTypes map[string]reflect.Type
}

func NewTypes() *Types {
	return &Types{
		make(map[reflect.Type]string),
		make(map[string]reflect.Type),
	}
}

func (ts *Types) Register(args ...TypeParam) error {
	var errs []error

	for _, arg := range args {
		if err := ts.register(arg.name, arg.t); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (ts *Types) MustRegister(args ...TypeParam) *Types {
	if err := ts.Register(args...); err != nil {
		panic(err)
	}

	return ts
}

func (ts *Types) RegisterType(name string, value any) error {
	return ts.register(name, reflect.ValueOf(value).Type())
}

func (ts *Types) RegisterTypeName(value any) error {
	t := reflect.ValueOf(value).Type()
	return ts.register(t.Name(), t)
}

func (ts *Types) register(name string, t reflect.Type) error {
	if name == "" {
		return fmt.Errorf("marshal: no name for type: %s", t)
	}
	if !isTypeSupported(t) {
		return fmt.Errorf("marshal: type not supported: %s", t)
	}
	if _, found := ts.typeNames[t]; found {
		return fmt.Errorf("marshal: type already registered: %s", t)
	}
	if _, found := ts.nameTypes[name]; found {
		return fmt.Errorf("marshal: type name already registered: %q", name)
	}

	ts.typeNames[t] = name
	ts.nameTypes[name] = t
	return nil
}

func isTypeSupported(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Map:
		return isMapKeyTypeSupported(t.Key())
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return false
	default:
		return true
	}
}

func isMapKeyTypeSupported(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.String:
		return true
	default:
		return false
	}
}
