package log

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"sort"
)

var Default = new(Logger)

func init() {
	Default.SetHandler(&StreamHandler{
		W:   os.Stderr,
		Fmt: LineFormatter,
	})
}

func groupExtras(values map[interface{}]struct{}, fields map[string][]interface{}) (ret map[interface{}][]interface{}) {
	ret = make(map[interface{}][]interface{})
	for v := range values {
		ret[reflect.TypeOf(v)] = append(ret[reflect.TypeOf(v)], v)
	}
	for f, vs := range fields {
		ret[f] = append(ret[f], vs...)
	}
	return
}

type extra struct {
	Key    interface{}
	Values []interface{}
}

// TODO: This might not be necessary soon? Go introduces sorted printing of maps?
func sortExtras(extras map[interface{}][]interface{}) (ret []extra) {
	for k, v := range extras {
		ret = append(ret, extra{k, v})
	}
	sort.Slice(ret, func(i, j int) bool {
		return fmt.Sprint(ret[i].Key) < fmt.Sprint(ret[j].Key)
	})
	return
}

func Call() Msg {
	var pc [1]uintptr
	n := runtime.Callers(4, pc[:])
	fs := runtime.CallersFrames(pc[:n])
	f, _ := fs.Next()
	return Fmsg("called %q", f.Function)
}
