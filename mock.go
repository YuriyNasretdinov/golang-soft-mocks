package soft

import (
	"reflect"
	"sync"
	"sync/atomic"
)

var mocksMutex sync.Mutex
var mocks = make(map[FuncPtr]interface{})

func Mock(src interface{}, dst interface{}) {
	fHash := GetFuncPtr(src)

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

func CallOriginal(f interface{}, args ...interface{}) []interface{} {
	fHash := GetFuncPtr(f)

	fl, ok := pkgFlags[fHash]
	if !ok {
		panic("Function is not registered")
	}

	if atomic.LoadInt32((*int32)(fl)) != 0 {
		atomic.StoreInt32((*int32)(fl), 0)
		defer atomic.StoreInt32((*int32)(fl), 1)
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

func GetMockFor(f interface{}) interface{} {
	fHash := GetFuncPtr(f)

	mocksMutex.Lock()
	res := mocks[fHash]
	mocksMutex.Unlock()

	return res
}
