package fi

import (
	"reflect"

	"github.com/golang/glog"
	"encoding/json"
	"fmt"
)

func BuildChanges(a, e, changes interface{}) bool {
	changed := false

	ve := reflect.ValueOf(e)
	vc := reflect.ValueOf(changes)

	ve = ve.Elem()
	vc = vc.Elem()

	t := vc.Type()
	if t != ve.Type() {
		panic("mismatched types in BuildChanges")
	}

	va := reflect.ValueOf(a)
	aIsNil := false

	if va.IsNil() {
		aIsNil = true
	}
	if !aIsNil {
		va = va.Elem()

		if t != va.Type() {
			panic("mismatched types in BuildChanges")
		}
	}

	for i := 0; i < ve.NumField(); i++ {
		if t.Field(i).PkgPath != "" {
			// unexported
			continue
		}

		fve := ve.Field(i)
		if fve.Kind() == reflect.Ptr && fve.IsNil() {
			// No expected value means 'don't change'
			continue
		}

		if ignoreField(fve) {
			continue
		}

		if !aIsNil {
			fva := va.Field(i)

			if equalFieldValues(fva, fve) {
				continue
			}

			glog.V(8).Infof("Field changed %q actual=%q expected=%q", t.Field(i).Name, DebugPrint(fva.Interface()), DebugPrint(fve.Interface()))
		}
		changed = true
		vc.Field(i).Set(fve)
	}

	return changed
}

func equalFieldValues(a, e reflect.Value) bool {
	//if a.Kind() == reflect.Ptr && !a.IsNil() && !e.IsNil() {
	//	a = a.Elem()
	//	e = e.Elem()
	//}

	if a.Kind() == reflect.Map {
		return equalMapValues(a, e)
	}
	if (a.Kind() == reflect.Ptr || a.Kind() == reflect.Interface) &&  !a.IsNil() {
		aHasID, ok := a.Interface().(HasID)
		if ok && (e.Kind() == reflect.Ptr || e.Kind() == reflect.Interface) &&  !e.IsNil() {
			eHasID, ok := e.Interface().(HasID)
			if ok {
				aID := aHasID.GetID()
				eID := eHasID.GetID()
				if aID != nil && eID != nil && *aID == *eID {
					return true
				}
			}
		}

		aResource, ok := a.Interface().(Resource)
		if ok && (e.Kind() == reflect.Ptr || e.Kind() == reflect.Interface) && !e.IsNil() {
			eResource, ok := e.Interface().(Resource)
			if ok {
				same, err := ResourcesMatch(aResource, eResource)
				if err != nil {
					glog.Fatalf("error while comparing resources: %v", err)
				} else {
					return same
				}
			}
		}
	}
	//if a.Kind() == reflect.Ptr && !a.IsNil() && e.Kind() == reflect.Ptr && !e.IsNil() {
	//	if reflect.DeepEqual(a.Elem().Interface(), e.Elem().Interface()) {
	//		return true
	//	}
	//}
	if reflect.DeepEqual(a.Interface(), e.Interface()) {
		return true
	}
	return false
}

func equalMapValues(a, e reflect.Value) bool {
	if a.IsNil() != e.IsNil() {
		return false
	}
	if a.IsNil() && e.IsNil() {
		return true
	}
	if a.Len() != e.Len() {
		return false
	}
	for _, k := range a.MapKeys() {
		valA := a.MapIndex(k)
		valE := e.MapIndex(k)

		glog.Infof("comparing maps: %v %v %v", k, valA, valE)

		if !equalFieldValues(valA, valE) {
			glog.Infof("unequals map value: %v %v %v", k, valA, valE)
			return false
		}
	}
	return true
}

func ignoreField(e reflect.Value) bool {
	if e.Kind() == reflect.Struct {
		_, ok := e.Addr().Interface().(*SimpleUnit)
		if ok {
			return true
		}
	}
	return false
}

func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func JsonString(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("error marshalling: %v", err)
	}
	return string(data)
}

func String(s string) *string {
	return &s
}

func Bool(v bool) *bool {
	return &v
}

func BoolValue(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}

func Int(v int) *int {
	return &v
}

func Int64(v int64) *int64 {
	return &v
}

func DebugPrint(o interface{}) string {
	if o == nil {
		return "<nil>"
	}
	v := reflect.ValueOf(o)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "<nil>"
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return "<?>"
	}
	o = v.Interface()
	stringer, ok := o.(fmt.Stringer)
	if ok {
		return stringer.String()
	}
	return fmt.Sprint(o)
}


