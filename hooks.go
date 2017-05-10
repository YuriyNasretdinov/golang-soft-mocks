package soft

type (
	FuncPtr uintptr
	FlagPtr *int32 // flag pointer indicates that there exists a mock and is atomically read and written
)

var pkgFlags = make(map[FuncPtr]FlagPtr)
var pkgFuncs = make(map[FuncPtr]interface{})

// callback that is used in rewritten files
func RegisterFunc(fun interface{}, flagPtr *int32) {
	ptr := GetFuncPtr(fun)
	println("Registering func ", ptr)

	pkgFlags[ptr] = FlagPtr(flagPtr)
	pkgFuncs[ptr] = fun
}
