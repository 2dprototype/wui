/*
Package check implements easy to use functions to write your tests in a concise
way.

In the past I have used if-statements to compare values for equality, for
example:

	func Test(t *testing.T) {
		sum := add(1, 2)
		if sum != 3 {
			t.Errorf("1+2 = 3 but got %v", sum)
		}
	}

This would soon get tedious and copy paste errors for the error messages would
creep in. Thus I would create a helper function to compare two integers, like
this:

	func checkInts(t *testing.T, msg string, have, want int) {
		if have != want {
			t.Errorf("%s: %v != %v", msg, have, want)
		}
	}

Since there are no generics in Go, I would have to write a helper for all types
of values that I wanted to compare: int, uint, byte, float32, []byte, map, etc.
The functions to implement comparison for equality of two values can get more
complex depeding on the type.
Floating point values can usually not be compared with == because of rounding
errors. You want to have an epsilon by which the values might diverge from each
other and still be considered equal. Furthermore, what about INF and NAN?
Comparing slices and maps means you must compare their lengths and iterate over
them to check each item for equality.
Nested structs and interfaces can get hairy pretty quickly.

To get over these problems once and for all I created this package. It aims at a
minimal API with maximum usability. You can only check for equality or
non-equality with the Eq and Neq functions.

The above example becomes:

	func Test(t *testing.T) {
		sum := add(1, 2)
		check.Eq(t, sum, 3, "1+2")
	}

It does not matter whether the add function returns an int, a uint32, a byte or
a float64. Eq and Neq compare values in a deep way while handling different
integer types, floating point accuracy, INF and NAN and comparison between
string, []byte and []rune.

This package will not solve all your testing needs but probably 95% of it. You
can still write if-statements or special helpers for the cases where simple
equality of values does not fit your needs.
*/
package check

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"unsafe"
)

// Tester wraps the Errorf message which is used to print error messages in case
// a check fails.
// *testing.T fulfills the Tester interface.
type Tester interface {
	Errorf(format string, a ...interface{})
}

// *testing.T only started supporting the Helper() function in Go 1.9. To
// support older versions we query the helper interface at runtime and only call
// Helper if it is available.
type helper interface {
	Helper()
}

// Eq compares a and b and calls Errorf on t if they differ. Values are compared
// in a deep way, similar to reflect.DeepEqual, only that float and complex
// values are compared using an epsilon of 1e-6.
// If there are any msg parameters, they are printed in concatenation before the
// error message, e.g. if you pass ["input ", 5] as msg, errors will be printed
// as: "input 5: <error>".
func Eq(t Tester, a, b interface{}, msg ...interface{}) {
	if h, ok := t.(helper); ok {
		h.Helper()
	}
	EqEps(t, a, b, 1e-6, msg...)
}

// EqExact compares a and b and calls Errorf on t if they differ. Values are
// compared in a deep way, similar to reflect.DeepEqual, float and complex
// values must match exactly.
// If there are any msg parameters, they are printed in concatenation before the
// error message, e.g. if you pass ["input ", 5] as msg, errors will be printed
// as: "input 5: <error>".
func EqExact(t Tester, a, b interface{}, msg ...interface{}) {
	if h, ok := t.(helper); ok {
		h.Helper()
	}
	EqEps(t, a, b, 0, msg...)
}

// EqEps compares a and b and calls Errorf on t if they differ. Values are
// compared in a deep way, similar to reflect.DeepEqual, only that float and
// complex values are compared using epsilon. Values are considered equal if
// their absolute difference is less than or equal to epsilon. Set epsilon to
// zero to compare for exact equality (or use EqExact).
// If there are any msg parameters, they are printed in concatenation before the
// error message, e.g. if you pass ["input ", 5] as msg, errors will be printed
// as: "input 5: <error>".
func EqEps(t Tester, a, b interface{}, epsilon float64, msg ...interface{}) {
	if h, ok := t.(helper); ok {
		h.Helper()
	}
	if !deepEqual(a, b, epsilon) {
		errorf(t, "!=", a, b, msg...)
	}
}

// Neq compares a and b and calls Errorf on t if they are equal. Values are
// compared in a deep way, similar to reflect.DeepEqual, only that float and
// complex values are compared using an epsilon of 1e-6.
// If there are any msg parameters, they are printed in concatenation before the
// error message, e.g. if you pass ["input ", 5] as msg, errors will be printed
// as: "input 5: <error>".
func Neq(t Tester, a, b interface{}, msg ...interface{}) {
	if h, ok := t.(helper); ok {
		h.Helper()
	}
	NeqEps(t, a, b, 1e-6, msg...)
}

// NeqExact compares a and b and calls Errorf on t if they are equal. Values are
// compared in a deep way, similar to reflect.DeepEqual, float and complex
// values must match exactly.
// If there are any msg parameters, they are printed in concatenation before the
// error message, e.g. if you pass ["input ", 5] as msg, errors will be printed
// as: "input 5: <error>".
func NeqExact(t Tester, a, b interface{}, msg ...interface{}) {
	if h, ok := t.(helper); ok {
		h.Helper()
	}
	NeqEps(t, a, b, 0, msg...)
}

// NeqEps compares a and b and calls Errorf on t if they are equal. Values are
// compared in a deep way, similar to reflect.DeepEqual, only that float and
// complex values are compared using epsilon. Values are considered equal if
// their absolute difference is less than or equal to epsilon. Set epsilon to
// zero to compare for exact equality (or use EqExact).
// If there are any msg parameters, they are printed in concatenation before the
// error message, e.g. if you pass ["input ", 5] as msg, errors will be printed
// as: "input 5: <error>".
func NeqEps(t Tester, a, b interface{}, epsilon float64, msg ...interface{}) {
	if h, ok := t.(helper); ok {
		h.Helper()
	}
	if deepEqual(a, b, epsilon) {
		errorf(t, "==", a, b, msg...)
	}
}

func errorf(t Tester, op string, a, b interface{}, msg ...interface{}) {
	if h, ok := t.(helper); ok {
		h.Helper()
	}
	var prefix string
	if len(msg) > 0 {
		prefix = fmt.Sprint(msg...) + ": "
	}
	t.Errorf("%s%#v %s %#v", prefix, a, op, b)
}

// deepEqual is a modified version of reflect.DeepEqual. deepEqual compares
// float and complex values using epsilon.
func deepEqual(x, y interface{}, epsilon float64) bool {
	if x == nil && y == nil {
		return true
	}
	if y == nil {
		x, y = y, x // make sure y is not nil
	}
	if x == nil {
		// y is not nil
		y := reflect.ValueOf(y)
		if y.Kind() == reflect.Slice {
			return y.IsNil() || y.Len() == 0
		}
		if y.Kind() == reflect.Ptr {
			return y.IsNil()
		}
		return false
	}
	return deepValueEqual(
		reflect.ValueOf(x),
		reflect.ValueOf(y),
		epsilon,
		make(map[visit]bool),
	)
}

func deepValueEqual(v1, v2 reflect.Value, eps float64, visited map[visit]bool) bool {
	if v1.Type() != v2.Type() {
		if canBeString(v1) && canBeString(v2) {
			return bytes.Equal(toBytes(v1), toBytes(v2))
		}
		if isInteger(v1) && isInteger(v2) {
			// We might need to compare signed with unsigned.
			v1Signed := isSignedInteger(v1)
			if v1Signed != isSignedInteger(v2) {
				// One signed, one unsigned; make sure the unsigned type is v1.
				if v1Signed {
					v1, v2 = v2, v1
				}
				// v1: unsigned type
				// v2: signed type
				i2 := v2.Int()
				if i2 < 0 {
					return false // v2 is unsigned, thus always >= 0
				}
				u1 := v1.Uint()
				return u1 == uint64(i2)
			}
			// At this point either both are signed or both are unsigned, thus
			// their uint64 bit patterns should match.
			return toUint64(v1) == toUint64(v2)
		}
		if isFloat(v1) && isFloat(v2) {
			return floatEq(v1.Float(), v2.Float(), eps)
		}
		if isComplex(v1) && isComplex(v2) {
			c1 := v1.Complex()
			c2 := v2.Complex()
			return floatEq(real(c1), real(c2), eps) &&
				floatEq(imag(c1), imag(c2), eps)
		}
		// check for int/float to complex comparison, make the complex be v2
		if isComplex(v1) {
			v1, v2 = v2, v1
		}
		if isComplex(v2) {
			c2 := v2.Complex()
			if imag(c2) != 0 {
				// imaginary part is non-zero, but for integer and floats it
				// must be zero
				return false
			}
			f2 := real(c2)
			if isInteger(v1) {
				f1 := intToFloat64(v1)
				return floatEq(f1, f2, eps)
			} else if isFloat(v1) {
				f1 := v1.Float()
				return floatEq(f1, f2, eps)
			} else {
				return false
			}
		}
		// check for integer to float comparison, make the integer be v1
		if isInteger(v2) {
			v1, v2 = v2, v1
		}
		if isInteger(v1) && isFloat(v2) {
			f1 := intToFloat64(v1)
			f2 := v2.Float()
			return floatEq(f1, f2, eps)
		}
		return false
	}

	// At this point we compare two values of the same type.

	// We want to avoid putting more in the visited map than we need to.
	// For any possible reference cycle that might be encountered,
	// hard(t) needs to return true for at least one of the types in the cycle.
	hard := func(k reflect.Kind) bool {
		switch k {
		case reflect.Map, reflect.Slice, reflect.Ptr, reflect.Interface:
			return true
		}
		return false
	}

	if v1.CanAddr() && v2.CanAddr() && hard(v1.Kind()) {
		addr1 := unsafe.Pointer(v1.UnsafeAddr())
		addr2 := unsafe.Pointer(v2.UnsafeAddr())
		if uintptr(addr1) > uintptr(addr2) {
			// Canonicalize order to reduce number of entries in visited.
			// Assumes non-moving garbage collector.
			addr1, addr2 = addr2, addr1
		}

		// Short circuit if references are already seen.
		typ := v1.Type()
		v := visit{addr1, addr2, typ}
		if visited[v] {
			return true
		}

		// Remember for later.
		visited[v] = true
	}

	switch v1.Kind() {
	case reflect.Array:
		// v1 and v2 have the same type, the length of the array is part of its
		// type, thus we need not compare their lengths.
		for i := 0; i < v1.Len(); i++ {
			if !deepValueEqual(v1.Index(i), v2.Index(i), eps, visited) {
				return false
			}
		}
		return true
	case reflect.Slice:
		if v1.IsNil() && v2.Len() == 0 {
			return true
		}
		if v1.Len() == 0 && v2.IsNil() {
			return true
		}
		if v1.IsNil() != v2.IsNil() {
			return false
		}
		if v1.Len() != v2.Len() {
			return false
		}
		if v1.Pointer() == v2.Pointer() {
			return true
		}
		for i := 0; i < v1.Len(); i++ {
			if !deepValueEqual(v1.Index(i), v2.Index(i), eps, visited) {
				return false
			}
		}
		return true
	case reflect.Interface:
		if v1.IsNil() || v2.IsNil() {
			return v1.IsNil() == v2.IsNil()
		}
		return deepValueEqual(v1.Elem(), v2.Elem(), eps, visited)
	case reflect.Ptr:
		if v1.Pointer() == v2.Pointer() {
			return true
		}
		return deepValueEqual(v1.Elem(), v2.Elem(), eps, visited)
	case reflect.Struct:
		for i, n := 0, v1.NumField(); i < n; i++ {
			if !deepValueEqual(v1.Field(i), v2.Field(i), eps, visited) {
				return false
			}
		}
		return true
	case reflect.Map:
		if v1.IsNil() != v2.IsNil() {
			return false
		}
		if v1.Len() != v2.Len() {
			return false
		}
		if v1.Pointer() == v2.Pointer() {
			return true
		}
		for _, k := range v1.MapKeys() {
			val1 := v1.MapIndex(k)
			val2 := v2.MapIndex(k)
			if !val1.IsValid() ||
				!val2.IsValid() ||
				!deepValueEqual(v1.MapIndex(k), v2.MapIndex(k), eps, visited) {
				return false
			}
		}
		return true
	case reflect.Func:
		if v1.IsNil() && v2.IsNil() {
			return true
		}
		return v1.Pointer() == v2.Pointer()
	case reflect.Bool:
		return v2.Kind() == reflect.Bool && v1.Bool() == v2.Bool()
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr:
		k2 := v2.Kind()
		return (k2 == reflect.Uint ||
			k2 == reflect.Uint8 ||
			k2 == reflect.Uint16 ||
			k2 == reflect.Uint32 ||
			k2 == reflect.Uint64 ||
			k2 == reflect.Uintptr) && v1.Uint() == v2.Uint()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		k2 := v2.Kind()
		return (k2 == reflect.Int ||
			k2 == reflect.Int8 ||
			k2 == reflect.Int16 ||
			k2 == reflect.Int32 ||
			k2 == reflect.Int64) && v1.Int() == v2.Int()
	case reflect.Float32, reflect.Float64:
		k2 := v2.Kind()
		return (k2 == reflect.Float32 || k2 == reflect.Float64) &&
			floatEq(v1.Float(), v2.Float(), eps)
	case reflect.Complex64, reflect.Complex128:
		k2 := v2.Kind()
		return (k2 == reflect.Complex64 || k2 == reflect.Complex128) &&
			floatEq(real(v1.Complex()), real(v2.Complex()), eps) &&
			floatEq(imag(v1.Complex()), imag(v2.Complex()), eps)
	case reflect.String:
		return v2.Kind() == reflect.String && v1.String() == v2.String()
	case reflect.UnsafePointer:
		return v2.Kind() == reflect.UnsafePointer && v1.Pointer() == v2.Pointer()
	default:
		return false
	}
}

func isInteger(v reflect.Value) bool {
	return isSignedInteger(v) || isUnsignedInteger(v)
}

func isSignedInteger(v reflect.Value) bool {
	k := v.Type().Kind()
	return k == reflect.Int ||
		k == reflect.Int8 ||
		k == reflect.Int16 ||
		k == reflect.Int32 ||
		k == reflect.Int64
}

func isUnsignedInteger(v reflect.Value) bool {
	k := v.Type().Kind()
	return k == reflect.Uint ||
		k == reflect.Uint8 ||
		k == reflect.Uint16 ||
		k == reflect.Uint32 ||
		k == reflect.Uint64 ||
		k == reflect.Uintptr
}

func toUint64(v reflect.Value) uint64 {
	if isSignedInteger(v) {
		return uint64(v.Int())
	}
	return v.Uint()
}

func intToFloat64(v reflect.Value) float64 {
	if isSignedInteger(v) {
		return float64(v.Int())
	}
	return float64(v.Uint())
}

func isFloat(v reflect.Value) bool {
	k := v.Type().Kind()
	return k == reflect.Float32 || k == reflect.Float64
}

func isComplex(v reflect.Value) bool {
	k := v.Type().Kind()
	return k == reflect.Complex64 || k == reflect.Complex128
}

type visit struct {
	a1  unsafe.Pointer
	a2  unsafe.Pointer
	typ reflect.Type
}

func floatEq(a, b, eps float64) bool {
	return math.IsInf(a, 1) && math.IsInf(b, 1) ||
		math.IsInf(a, -1) && math.IsInf(b, -1) ||
		math.IsNaN(a) && math.IsNaN(b) ||
		abs(a-b) <= eps
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func canBeString(v reflect.Value) bool {
	if v.Kind() == reflect.String {
		return true
	}
	if v.Kind() == reflect.Slice {
		// byte or rune slices can be converted to string
		return v.Type().Elem().Kind() == reflect.Uint8 || v.Type().Elem().Kind() == reflect.Int32
	}
	return false
}

func toBytes(v reflect.Value) []byte {
	if v.Kind() == reflect.String {
		return []byte(v.String())
	}
	if v.Type().Elem().Kind() == reflect.Uint8 {
		return v.Bytes()
	}
	// in this case we have a rune slice
	return []byte(string(v.Interface().([]rune)))
}
