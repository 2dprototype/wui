package check_test

import (
	"fmt"
	"testing"
	"unsafe"

	"github.com/2dprototype/wui/check"
)

type mockTester struct {
	err      string
	isHelper bool
}

func (t *mockTester) Errorf(format string, a ...interface{}) {
	t.err = fmt.Sprintf(format, a...)
}

func (t *mockTester) Helper() {
	t.isHelper = true
}

func TestEqAndNeq(t *testing.T) {
	// eq and neq are helper functions that let us state facts about values that
	// are equal and not equal. eq asserts that check.Eq is true for both
	// arguments, independent of the order (a,b or b,a) and that check.Neq is
	// false. neq does the opposite of that.
	eq := func(a, b interface{}) {
		t.Helper()
		var ttEqAB mockTester
		check.Eq(&ttEqAB, a, b)
		if ttEqAB.err != "" {
			t.Errorf("%v == %v but error for Eq was %q", a, b, ttEqAB.err)
		}
		var ttEqBA mockTester
		check.Eq(&ttEqBA, b, a)
		if ttEqBA.err != "" {
			t.Errorf("%v == %v but error for Eq was %q", b, a, ttEqBA.err)
		}
		var ttNeqAB mockTester
		check.Neq(&ttNeqAB, a, b)
		if ttNeqAB.err == "" {
			t.Errorf("%v != %v but have no error for Neq", a, b)
		}
		var ttNeqBA mockTester
		check.Neq(&ttNeqBA, b, a)
		if ttNeqBA.err == "" {
			t.Errorf("%v != %v but have no error for Neq", b, a)
		}
	}
	neq := func(a, b interface{}) {
		t.Helper()
		var ttEqAB mockTester
		check.Eq(&ttEqAB, a, b)
		if ttEqAB.err == "" {
			t.Errorf("%v != %v but have no error for Eq", a, b)
		}
		var ttEqBA mockTester
		check.Eq(&ttEqBA, b, a)
		if ttEqBA.err == "" {
			t.Errorf("%v != %v but have no error for Eq", b, a)
		}
		var ttNeqAB mockTester
		check.Neq(&ttNeqAB, a, b)
		if ttNeqAB.err != "" {
			t.Errorf("%v == %v but error for Neq was %q", a, b, ttNeqAB.err)
		}
		var ttNeqBA mockTester
		check.Neq(&ttNeqBA, b, a)
		if ttNeqBA.err != "" {
			t.Errorf("%v == %v but error for Neq was %q", b, a, ttNeqBA.err)
		}
	}

	// numbers
	eq(0, 0)
	neq(1, 0)
	neq(1, nil)
	eq(int8(2), uint64(2))
	eq(float32(1.2), float64(1.2))
	eq(complex(1, 2), complex(1, 2))
	neq(complex(1, 2), complex(1, 456))
	eq(complex64(complex(1, 2)), complex128(complex(1, 2)))
	eq(complex(1, 0), float32(1))
	neq(complex(1, 9), float32(1))
	eq(complex(1, 0), 1)
	neq(complex(1, 9), 1)
	neq(complex(1, 2), false)
	eq(4, 4.0)
	eq(-4.0, -4)
	neq(1.2, "1.2")
	var i, j int
	eq(&i, &i)
	eq(&i, &j)
	eq(1.2, 1.20000001)
	neq(uint64(0xFFFFFFFFFFFFFFFF), int64(-1))
	eq(int32(5), int8(5))
	eq(uint32(5), uint64(5))
	eq(uint64(5), uint64(5))
	neq(uint64(999), uint64(5))
	eq(2.0, uint64(2))

	// boolean values
	eq(true, true)
	neq(true, false)

	// strings
	eq("", "")
	eq("abc", "abc")
	eq("abc", []byte("abc"))
	eq("abc", []rune("abc"))

	neq("abc", "ABC")
	neq("abc", []byte("ABC"))
	neq("abc", []rune("ABC"))

	// functions
	eq(eq, eq)
	neq(eq, neq)
	var nilF func()
	eq(nilF, nilF)

	// arrays
	var ints2 [2]int
	var ints3 [3]int
	eq(ints2, ints2)
	neq(ints2, ints3)
	neq([2]int{1, 2}, [2]int{1, 3})

	// slices
	eq([]int{1, 2, 3}, []int{1, 2, 3})
	neq([]int{1, 2, 3}, []int{1, 2})
	neq([]int{1, 2, 3}, []int{1, 2, 4})
	neq([]int{1, 2, 3}, []int(nil))
	slice := []int{1, 2, 3}
	eq(slice, slice)
	eq([]interface{}{7.0, 6.0, 5.0}, []interface{}{7, uint8(6), 5})
	neq([]interface{}{""}, []interface{}{5})

	// maps
	var nilMap map[int]string
	m1 := map[int]string{1: "abc"}
	m2 := map[int]string{1: "abc", 2: "def"}
	m22 := map[int]string{1: "abc", 2: "def"}
	m3 := map[int]string{1: "abc", 2: "DEF"}
	eq(m1, m1)
	neq(m1, m2)
	neq(nilMap, m1)
	eq(nilMap, nilMap)
	neq(m2, m3)
	eq(m2, m22)

	// structs and interfaces
	var s1, s2 struct{ a int }
	eq(s1, s1)
	eq(s1, s2)
	s2.a = 111
	neq(s1, s2)

	type bbb struct{}
	type aaa struct {
		bbb
		u unsafe.Pointer
		i interface {
			a()
		}
	}
	var aa1, aa2 aaa
	eq(aa1, aa2)
	aa1.u = unsafe.Pointer(&aa1)
	aa2.u = unsafe.Pointer(&aa2)
	eq(aa1, aa1)
	neq(aa1, aa2)

	aa2.u = aa1.u
	eq(aa1, aa2)

	aa1.i = aer{i: 1}
	aa2.i = aer{i: 1}
	eq(aa1, aa2)

	aa1.i = aer{i: 1}
	aa2.i = aer{i: 2}
	neq(aa1, aa2)

	// pointers
	p1 := unsafe.Pointer(&s1)
	p2 := unsafe.Pointer(&s2)
	eq(p1, p1)
	neq(p1, p2)
	neq(p1, &p1)
	eq(&p1, &p1)
	eq(nil, nil)
	var intPtr *int
	eq(nil, intPtr)
	var integer int
	neq(&integer, nil)

	// empty slices are nil
	s := make([]int, 0)
	eq(s, nil)
	eq(nil, s)
	eq(nil, []int{})
	eq([]int{}, nil)
	eq([]int{}, s)
	eq(s, []int{})

	// Fuzz-test different types of values, make sure Eq and Neq do not panic.
	values := []interface{}{
		false, true,
		0, -1, 1, integer, &integer,
		complex(1, 2),
		uintptr(0), uintptr(1),
		unsafe.Pointer(uintptr(0)), unsafe.Pointer(uintptr(1)),
		"", "string",
		[]byte{}, []byte("bytes"),
		[]rune{}, []rune("bytes"),
		nil,
		make(map[int]string),
		make(chan bool),
		aer{}, &aer{},
		nilF, func() {}, aer{}.a,
		[]int{}, []int{1, 2, 3},
	}
	for i := range values {
		for j := i + 1; j < len(values); j++ {
			a, b := values[i], values[j]
			var ttEqAB mockTester
			check.Eq(&ttEqAB, a, b)
			check.Eq(&ttEqAB, b, a)
			check.Neq(&ttEqAB, a, b)
			check.Neq(&ttEqAB, b, a)
		}
	}
}

type aer struct{ i int }

func (aer) a() {}

func TestEqHasMessage(t *testing.T) {
	var tt mockTester
	check.Eq(&tt, 1, 2, "message")
	if tt.err != "message: 1 != 2" {
		t.Error(tt.err)
	}

	tt.err = ""
	check.Neq(&tt, 1, 1, "wat")
	if tt.err != "wat: 1 == 1" {
		t.Error(tt.err)
	}
}

func TestEqExact(t *testing.T) {
	var tt mockTester
	check.EqExact(&tt, 1.0, 1.0)
	if tt.err != "" {
		t.Error("no error expected")
	}

	tt.err = ""
	check.EqExact(&tt, 1.0, 1.00000001)
	if tt.err == "" {
		t.Error("error expected")
	}

	tt.err = ""
	check.NeqExact(&tt, 1.0, 1.0)
	if tt.err == "" {
		t.Error("error expected")
	}

	tt.err = ""
	check.NeqExact(&tt, 1.0, 1.00000001)
	if tt.err != "" {
		t.Error("no error expected", tt.err)
	}
}

func TestHelperFunctionIsDeclared(t *testing.T) {
	var tt mockTester
	check.Eq(&tt, 0, 0)
	if !tt.isHelper {
		t.Error("Eq does not declare itself as Helper()")
	}
}
