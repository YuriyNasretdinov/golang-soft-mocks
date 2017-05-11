package soft

type (
	funcPtr uintptr
	flagPtr *int32 // flag pointer indicates that there exists a mock and is atomically read and written
)

var pkgFlags = make(map[funcPtr]flagPtr)
var pkgFuncs = make(map[funcPtr]interface{})

// callback that is used in rewritten files
func RegisterFunc(fun interface{}, p *int32) {
	f := GetFuncPtr(fun)
	pkgFlags[f] = flagPtr(p)
	pkgFuncs[f] = fun
}
