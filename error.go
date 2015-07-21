// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pc

import (
	"unicode/utf16"

	"golang.org/x/sys/windows"
)

const (
	PDH_MORE_DATA    = Errno(0x800007d2)
	PDH_INVALID_DATA = Errno(0xC0000BC6)
)

type Errno uintptr

func langid(pri, sub uint16) uint32 { return uint32(sub)<<10 | uint32(pri) }

func itoa(val int) string { // do it here rather than with fmt to avoid dependency
	if val < 0 {
		return "-" + itoa(-val)
	}
	var buf [32]byte // big enough for int64
	i := len(buf) - 1
	for val >= 10 {
		buf[i] = byte(val%10 + '0')
		i--
		val /= 10
	}
	buf[i] = byte(val + '0')
	return string(buf[i:])
}

func (e Errno) Error() string {
	var flags uint32 = windows.FORMAT_MESSAGE_FROM_HMODULE | windows.FORMAT_MESSAGE_ARGUMENT_ARRAY | windows.FORMAT_MESSAGE_IGNORE_INSERTS
	b := make([]uint16, 300)
	h := uintptr(modpdh.Handle())
	n, err := windows.FormatMessage(flags, h, uint32(e), langid(windows.LANG_ENGLISH, windows.SUBLANG_ENGLISH_US), b, nil)
	if err != nil {
		n, err = windows.FormatMessage(flags, h, uint32(e), 0, b, nil)
		if err != nil {
			return "pdh error #" + itoa(int(e))
		}
	}
	// trim terminating \r and \n
	for ; n > 0 && (b[n-1] == '\n' || b[n-1] == '\r'); n-- {
	}
	return string(utf16.Decode(b[:n]))
}
