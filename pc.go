// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Windows performance counters.
// (MANY THINGS ARE BROKEN)
package pc

import (
	"syscall"
	"time"
	"unsafe"
)

//go:generate go run mksyscall_windows.go -output zapi.go pc.go

// Performance query.
type Query struct {
	Handle PDH_HQUERY
}

type PDH_HQUERY syscall.Handle

//sys	PdhOpenQuery(datasrc *uint16, userdata uint32, query *PDH_HQUERY) (pdherr error) = pdh.PdhOpenQueryW
//sys	PdhCloseQuery(query PDH_HQUERY) (pdherr error) = pdh.PdhCloseQuery
//sys	PdhCollectQueryData(query PDH_HQUERY) (pdherr error) = pdh.PdhCollectQueryData

func toptr(s string) *uint16 {
	if len(s) == 0 {
		return nil
	}
	return &(syscall.StringToUTF16(s))[0]
}

// OpenQuery creates a new query that is used to manage
// the collection of performance data.
func OpenQuery(datasrc string, userdata uint32) (*Query, error) {
	var h PDH_HQUERY
	err := PdhOpenQuery(toptr(datasrc), userdata, &h)
	if err != nil {
		return nil, err
	}
	return &Query{h}, nil
}

// CollectData collects the current raw data value for all counters
// in the query q and updates the status code of each counter.
func (q *Query) CollectData() error {
	return PdhCollectQueryData(q.Handle)
}

// Close closes all counters contained in the query q,
// closes all handles related to the query, and frees all memory
// associated with the query.
func (q *Query) Close() error {
	return PdhCloseQuery(q.Handle)
}

// Performance counter.
type Counter struct {
	Handle PDH_HCOUNTER
}

const (
	// Determines the data type of the formatted value.
	// Specify one of the following values.
	PDH_FMT_DOUBLE = 0x00000200 // Return data as a double-precision floating point real.
	PDH_FMT_LARGE  = 0x00000400 // Return data as a 64-bit integer.
	PDH_FMT_LONG   = 0x00000100 // Return data as a long integer.

	// You can use the bitwise inclusive OR operator to combine
	// the data type with one of the following scaling factors.
	PDH_FMT_NOSCALE  = 0x00001000 // Do not apply the counter's default scaling factor.
	PDH_FMT_NOCAP100 = 0x00008000 // Counter values greater than 100 (for example, counter values measuring the processor load on multiprocessor computers) will not be reset to 100. The default behavior is that counter values are capped at a value of 100.
	PDH_FMT_1000     = 0x00002000 // Multiply the actual value by 1,000.
)

type PDH_HCOUNTER syscall.Handle

type PDH_RAW_COUNTER struct {
	CStatus     uint32
	TimeStamp   syscall.Filetime
	padding1    uint32
	FirstValue  int64
	SecondValue int64
	MultiCount  uint32
	padding2    uint32
}

func (c *PDH_RAW_COUNTER) Time() time.Time {
	t := time.Unix(0, c.TimeStamp.Nanoseconds()).UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
}

type PDH_FMT_COUNTERVALUE struct {
	CStatus uint32
	padding uint32
	Value   uint64 // largest value of the union possible
}

func (i *PDH_FMT_COUNTERVALUE_ITEM) NameString() string {
	return syscall.UTF16ToString((*[20]uint16)(unsafe.Pointer(i.Name))[:])
}

//sys	PdhAddCounter(query PDH_HQUERY, fullpath *uint16, userdata uint32, counter *PDH_HCOUNTER) (pdherr error) = pdh.PdhAddCounterW
//sys	PdhRemoveCounter(counter PDH_HCOUNTER) (pdherr error) = pdh.PdhRemoveCounter
//sys	PdhGetRawCounterValue(counter PDH_HCOUNTER, ctype *uint32, value *PDH_RAW_COUNTER) (pdherr error) = pdh.PdhGetRawCounterValue
//sys	PdhGetFormattedCounterValue(counter PDH_HCOUNTER, format uint32, ctype *uint32, value *PDH_FMT_COUNTERVALUE) (pdherr error) = pdh.PdhGetFormattedCounterValue
//sys	PdhGetFormattedCounterArray(counter PDH_HCOUNTER, format uint32, bufsize *uint32, bufcnt *uint32, item *PDH_FMT_COUNTERVALUE_ITEM) (pdherr error) = pdh.PdhGetFormattedCounterArrayW

// AddCounter adds the specified counter to the query q.
func (q *Query) AddCounter(fullpath string, userdata uint32) (*Counter, error) {
	var h PDH_HCOUNTER
	err := PdhAddCounter(q.Handle, toptr(fullpath), userdata, &h)
	if err != nil {
		return nil, err
	}
	return &Counter{h}, nil
}

// Remove removes counter c from a query.
func (c *Counter) Remove() error {
	return PdhRemoveCounter(c.Handle)
}

func (c *Counter) GetFmtValue(format uint32) (uint32, *PDH_FMT_COUNTERVALUE, error) {
	var t uint32
	var v PDH_FMT_COUNTERVALUE
	err := PdhGetFormattedCounterValue(c.Handle, format, &t, &v)
	if err != nil {
		return 0, nil, err
	}
	return t, &v, nil
}

func (c *Counter) GetRawValue() (uint32, *PDH_RAW_COUNTER, error) {
	var t uint32
	var v PDH_RAW_COUNTER
	err := PdhGetRawCounterValue(c.Handle, &t, &v)
	if err != nil {
		return 0, nil, err
	}
	return t, &v, nil
}

// FmtArray.
type FmtArray struct {
	buf   []byte
	Items []PDH_FMT_COUNTERVALUE_ITEM
}

func (a *FmtArray) Get(c *Counter, format uint32) error {
	var size, count uint32
	var p *PDH_FMT_COUNTERVALUE_ITEM
	if len(a.buf) > 0 {
		size = uint32(len(a.buf))
		p = (*PDH_FMT_COUNTERVALUE_ITEM)(unsafe.Pointer(&a.buf[0]))
	}
	err := PdhGetFormattedCounterArray(c.Handle, format, &size, &count, p)
	if err != nil {
		if err != PDH_MORE_DATA {
			return err
		}
		a.buf = make([]byte, size)
		p = (*PDH_FMT_COUNTERVALUE_ITEM)(unsafe.Pointer(&a.buf[0]))
		err = PdhGetFormattedCounterArray(c.Handle, format, &size, &count, p)
		if err != nil {
			return err
		}
	}
	a.Items = (*[1 << 20]PDH_FMT_COUNTERVALUE_ITEM)(unsafe.Pointer(p))[:count]
	return nil

}

func (a *FmtArray) Clean() {
	a.buf = nil
	a.Items = nil
}
