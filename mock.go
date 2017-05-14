package soft

import (
	"reflect"
	"sync"
	"sync/atomic"
)

var mocksMutex sync.Mutex
var mocks = make(map[funcPtr]interface{})

func Mock(src interface{}, dst interface{}) {
	fHash := getFuncPtr(src)

	fl, ok := pkgFlags[fHash]
	if !ok {
		panic("Function cannot be mocked, it is not registered")
	}

	if !reflect.TypeOf(dst).ConvertibleTo(reflect.TypeOf(pkgFuncs[fHash])) {
		panic("Function signatures do not match")
	}

	atomic.StoreInt32((*int32)(fl), 1)

	mocksMutex.Lock()
	defer mocksMutex.Unlock()

	mocks[fHash] = dst
}

func getFlag(h flagPtr) bool {
	return atomic.LoadInt32((*int32)(h)) != 0
}

func setFlag(h flagPtr, v bool) {
	vInt := int32(0)
	if v {
		vInt = 1
	}

	atomic.StoreInt32((*int32)(h), vInt)
}

func CallOriginal(f interface{}, args ...interface{}) []interface{} {
	fHash := getFuncPtr(f)

	fl, ok := pkgFlags[fHash]
	if !ok {
		panic("Function is not registered")
	}

	if getFlag(fl) {
		setFlag(fl, false)
		defer setFlag(fl, true)
	}

	in := make([]reflect.Value, 0, len(args))
	for _, arg := range args {
		in = append(in, reflect.ValueOf(arg))
	}

	out := reflect.ValueOf(pkgFuncs[fHash]).Call(in)
	res := make([]interface{}, 0, len(out))
	for _, v := range out {
		res = append(res, v.Interface())
	}

	return res
}

func reset(fHash funcPtr) {
	fl, ok := pkgFlags[fHash]
	if !ok {
		panic("Function is not registered")
	}

	delete(mocks, fHash)
	setFlag(fl, false)
}

func Reset(f interface{}) {
	reset(getFuncPtr(f))
}

func ResetAll() {
	for ptr := range mocks {
		reset(ptr)
	}
}

func GetMockFor(f interface{}) interface{} {
	fHash := getFuncPtr(f)

	mocksMutex.Lock()
	res := mocks[fHash]
	mocksMutex.Unlock()

	return res
}
