// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pc

type PDH_FMT_COUNTERVALUE_ITEM struct {
	Name     *uint16
	padding  uint32 // TODO: could well be broken on amd64
	FmtValue PDH_FMT_COUNTERVALUE
}
