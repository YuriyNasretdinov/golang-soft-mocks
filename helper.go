package soft

import "reflect"

func GetFuncPtr(f interface{}) FuncPtr {
	return FuncPtr(reflect.ValueOf(f).Pointer())
}
