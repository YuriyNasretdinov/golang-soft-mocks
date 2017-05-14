package soft

import "reflect"

type (
	funcPtr uintptr
	flagPtr *int32 // flag pointer indicates that there exists a mock and is atomically read and written
)

var pkgFlags = make(map[funcPtr]flagPtr)
var pkgFuncs = make(map[funcPtr]interface{})

func getFuncPtr(f interface{}) funcPtr {
	return funcPtr(reflect.ValueOf(f).Pointer())
}

// callback that is used in rewritten files
func RegisterFunc(fun interface{}, p *int32) {
	f := getFuncPtr(fun)
	pkgFlags[f] = flagPtr(p)
	pkgFuncs[f] = fun
}
