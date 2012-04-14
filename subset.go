// Deep subset test via reflection. Based on reflect/deepcopy.go.

package subset

import (
	"reflect"
)

// During checkSubset, must keep track of checks that are
// in progress.  The comparison algorithm assumes that all
// checks in progress are true when it reencounters them.
// Visited are stored in a map indexed by 17 * a1 + a2;
type visit struct {
	a1   uintptr
	a2   uintptr
	typ  reflect.Type
	next *visit
}

// Tests for deep equality using reflected types. The map argument tracks
// comparisons that have already been seen, which allows short circuiting on
// recursive types.
func checkSubset(expected, target reflect.Value, visited map[uintptr]*visit, depth int) (b bool) {
	if !expected.IsValid() || !target.IsValid() {
		return expected.IsValid() == target.IsValid()
	}
	if expected.Type() != target.Type() {
		return false
	}

	// if depth > 10 { panic("checkSubset") }	// for debugging

	if expected.CanAddr() && target.CanAddr() {
		addr1 := expected.UnsafeAddr()
		addr2 := target.UnsafeAddr()
		if addr1 > addr2 {
			// Canonicalize order to reduce number of entries in visited.
			addr1, addr2 = addr2, addr1
		}

		// Short circuit if references are identical ...
		if addr1 == addr2 {
			return true
		}

		// ... or already seen
		h := 17*addr1 + addr2
		seen := visited[h]
		typ := expected.Type()
		for p := seen; p != nil; p = p.next {
			if p.a1 == addr1 && p.a2 == addr2 && p.typ == typ {
				return true
			}
		}

		// Remember for later.
		visited[h] = &visit{addr1, addr2, typ, seen}
	}

	switch expected.Kind() {
	case reflect.Array:
		if expected.Len() == 0 {
			return true
		}
		if expected.Len() != target.Len() {
			return false
		}
		for i := 0; i < expected.Len(); i++ {
			if !checkSubset(expected.Index(i), target.Index(i), visited, depth+1) {
				return false
			}
		}
		return true
	case reflect.Slice:
		if expected.IsNil() {
			return true
		}
		if expected.IsNil() != target.IsNil() {
			return false
		}
		if expected.Len() != target.Len() {
			return false
		}
		for i := 0; i < expected.Len(); i++ {
			if !checkSubset(expected.Index(i), target.Index(i), visited, depth+1) {
				return false
			}
		}
		return true
	case reflect.Interface:
		if expected.IsNil() {
			return true
		}
		if expected.IsNil() || target.IsNil() {
			return expected.IsNil() == target.IsNil()
		}
		return checkSubset(expected.Elem(), target.Elem(), visited, depth+1)
	case reflect.Ptr:
		return checkSubset(expected.Elem(), target.Elem(), visited, depth+1)
	case reflect.Struct:
		for i, n := 0, expected.NumField(); i < n; i++ {
			if !checkSubset(expected.Field(i), target.Field(i), visited, depth+1) {
				return false
			}
		}
		return true
	case reflect.Map:
		if expected.IsNil() {
			return true
		}
		if expected.IsNil() != target.IsNil() {
			return false
		}
		if expected.Len() != target.Len() {
			return false
		}
		for _, k := range expected.MapKeys() {
			if !checkSubset(expected.MapIndex(k), target.MapIndex(k), visited, depth+1) {
				return false
			}
		}
		return true
	case reflect.Func:
		if expected.IsNil() && target.IsNil() {
			return true
		}
		// Can't do better than this:
		return false
	default:
		// ignore zero value expectations
		zeroValue := reflect.Zero(expected.Type())
		if reflect.DeepEqual(expected.Interface(), zeroValue.Interface()) {
			return true
		}

		// Normal equality suffices
		return reflect.DeepEqual(expected.Interface(), target.Interface())
	}

	panic("Not reached")
}

// Check tests for deep subset. It uses normal == equality where
// possible but will scan members of arrays, slices, maps, and fields of
// structs. It correctly handles recursive types. Functions are equal
// only if they are both nil.
func Check(expected, target interface{}) bool {
	if expected == nil {
		return true
	}
	if target == nil {
		return false
	}
	return checkSubset(
		reflect.ValueOf(expected),
		reflect.ValueOf(target),
		make(map[uintptr]*visit),
		0)
}
