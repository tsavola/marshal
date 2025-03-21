// Copyright (c) 2025 Timo Savola
// SPDX-License-Identifier: BSD-3-Clause

package marshal

import (
	"import.name/pan"
)

var z = new(pan.Zone)

func must[T any](x T, err error) T {
	z.Check(err)
	return x
}
