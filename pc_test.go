// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pc

import (
	"testing"
	"time"
)

func printFmtArray(t *testing.T, c *Counter, format uint32) {
	a := new(FmtArray)
	err := a.Get(c, format)
	switch err {
	case PDH_INVALID_DATA:
		t.Logf("FmtArray.Get(): specified counter instance does not exist, skipping")
	case nil:
		for i, v := range a.Items {
			t.Logf("%d: name=%s %+v", i, v.NameString(), v.FmtValue)
		}
	default:
		t.Fatalf("FmtArray.Get() failed: %v", err)
	}
}

func TestPC(t *testing.T) {
	q, err := OpenQuery("", 0)
	if err != nil {
		t.Fatalf("OpenQuery failed: %v", err)
	}
	defer q.Close()

	const cname = `\Processor(*)\% Processor Time`
	//const cname = `\Processor(0)\% Processor Time`

	t.Logf("Checking %v ...", cname)

	c, err := q.AddCounter(cname, 0)
	if err != nil {
		t.Fatalf("AddCounter failed: %v", err)
	}
	defer c.Remove()

	const format = PDH_FMT_DOUBLE

	err = q.CollectData()
	if err != nil {
		t.Fatalf("CollectData failed: %v", err)
	}
	for i := 0; i < 3; i++ {
		time.Sleep(time.Second)

		err = q.CollectData()
		if err != nil {
			t.Fatalf("CollectData failed: %v", err)
		}
		printFmtArray(t, c, format)
	}
}

var memPCNames = []string{
	`\Memory\Available Mbytes`,
	`\Memory\Pages Input/sec`,
	`\Memory\Pages/sec`,
	`\Memory\Committed Bytes`,
	`\Memory\Commit Limit`,
	`\Memory\% Committed Bytes in Use`,
	`\Process(_Total)\Private Bytes`,
}

func TestMemory(t *testing.T) {
	q, err := OpenQuery("", 0)
	if err != nil {
		t.Fatalf("OpenQuery failed: %v", err)
	}
	defer q.Close()

	cs := make(map[string]*Counter)
	for _, name := range memPCNames {
		c, err := q.AddCounter(name, 0)
		if err != nil {
			t.Fatalf("AddCounter(%s) failed: %v", name, err)
		}
		cs[name] = c
	}

	err = q.CollectData()
	if err != nil {
		t.Fatalf("CollectData failed: %v", err)
	}

	//const format = PDH_FMT_DOUBLE
	//const format = PDH_FMT_LONG
	const format = PDH_FMT_LARGE

	for name, c := range cs {
		t.Logf("Checking %v ...", name)

		ctype, rval, err := c.GetRawValue()
		if err != nil {
			t.Fatalf("GetRawValue() failed: %v", err)
		}
		t.Logf("GetRawValue(): ctype=%v rval=%+v time=%v", ctype, rval, rval.Time())

		ctype, cval, err := c.GetFmtValue(format)
		switch err {
		case PDH_INVALID_DATA:
			t.Logf("GetFmtValue(): specified counter instance does not exist, skipping")
		case nil:
			t.Logf("GetFmtValue(): ctype=%v cval=%+v", ctype, cval)
		default:
			t.Fatalf("GetFmtValue() failed: %v", err)
		}

		printFmtArray(t, c, format)
	}
}

func TestErrorString(t *testing.T) {
	q, err := OpenQuery("no_such_source", 0)
	if err == nil {
		q.Close()
		t.Fatal("OpenQuery succeeded, but expected to fail.")
	}
	if err.Error() != "Unable to find the specified file." {
		t.Fatalf("'Unable to find the specified file.' error message expected, but %q found.", err.Error())
	}
}
