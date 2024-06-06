// Copyright (c) 2024 Timo Savola
// SPDX-License-Identifier: BSD-3-Clause

package marshal

import (
	"encoding/json"
	"reflect"
	"testing"
	"unsafe"
)

type topLevel struct {
	Int               int
	Uint16            uint16
	AltA              alt
	AltB              alt
	StructEmbedded    subLevel
	StructIndirect    *subLevel
	Self              *topLevel
	Slice             []*topLevel
	Array             [2]int
	MapBool           map[string]bool
	MapStructEmbedded map[int]subLevel
	MapStructIndirect map[int64]*subLevel
	NilPtr            *subLevel
	NilMap            map[string]int
	UnsupportedMap    map[*int]int
	UnsupportedFunc   func()
	UnsupportedChan   chan struct{}
	UnsupportedUnsafe unsafe.Pointer
}

type subLevel struct {
	Empty  struct{}
	Parent *topLevel
}

type alt interface {
	alt()
}

type alt1 struct {
	Alt1 string
}

func (alt1) alt() {}

type alt2 struct {
	Alt2 string
}

func (*alt2) alt() {}

func TestMarshal(t *testing.T) {
	types := NewTypes()

	if err := types.Register(
		TypeName(alt1{}),
		Type("alt2ptr", &alt2{}),
	); err != nil {
		t.Fatal("type registration error:", err)
	}

	x := &topLevel{
		10,
		20,
		alt1{"ALT-1"},
		&alt2{"ALT-2"},
		subLevel{},
		&subLevel{},
		nil, // Placeholder.
		nil, // Placeholder.
		[2]int{123, 456},
		map[string]bool{"t": true, "f": false},
		map[int]subLevel{0: subLevel{}},
		map[int64]*subLevel{1: &subLevel{}, -1: nil},
		nil,
		nil,
		map[*int]int{},
		func() {},
		make(chan struct{}),
		unsafe.Pointer(&types),
	}
	x.StructEmbedded.Parent = x
	x.StructIndirect.Parent = x
	x.Self = x
	x.Slice = []*topLevel{x, nil, x}[:2]

	objects, err := Marshal(x, types, true)
	if err != nil {
		t.Fatal("marshal error:", err)
	}

	if n := len(objects); n != 4 {
		t.Error("wrong number of objects:", n)
	}

	for i, obj := range objects {
		t.Logf("objects[%d]: %v", i, obj)
	}

	if b, err := json.MarshalIndent(objects, "", "  "); err == nil {
		t.Logf("JSON: %s", b)
	} else {
		t.Error("JSON error:", err)
	}

	y := new(topLevel)
	if err := Unmarshal(objects, y, types); err != nil {
		t.Fatal("unmarshal error:", err)
	}

	x.UnsupportedMap = nil
	x.UnsupportedFunc = nil
	x.UnsupportedChan = nil
	x.UnsupportedUnsafe = nil
	if !reflect.DeepEqual(x, y) {
		t.Errorf("mismatch:\nx: %#v\ny: %#v", x, y)
	}
}
