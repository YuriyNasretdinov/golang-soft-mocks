package soft

import "reflect"

func GetFuncPtr(f interface{}) funcPtr {
	return funcPtr(reflect.ValueOf(f).Pointer())
}
