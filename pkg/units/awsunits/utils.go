package awsunits

import (
	"reflect"
	crypto_rand "crypto/rand"

	"github.com/golang/glog"
	"encoding/json"
	"fmt"
	"encoding/base64"
	"bytes"
	"github.com/kopeio/kope/pkg/fi"
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
		fve := ve.Field(i)
		if fve.Kind() == reflect.Ptr && fve.IsNil() {
			// No expected value means 'don't change'
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

	if (a.Kind() == reflect.Ptr || a.Kind() == reflect.Interface) &&  !a.IsNil() {
		aHasID, ok := a.Interface().(fi.HasID)
		if ok && (e.Kind() == reflect.Ptr || e.Kind() == reflect.Interface) &&  !e.IsNil() {
			eHasID, ok := e.Interface().(fi.HasID)
			if ok {
				aID := aHasID.GetID()
				eID := eHasID.GetID()
				if aID != nil && eID != nil && *aID == *eID {
					return true
				}
			}
		}

		aResource, ok := a.Interface().(fi.Resource)
		if ok && (e.Kind() == reflect.Ptr || e.Kind() == reflect.Interface) && !e.IsNil() {
			eResource, ok := e.Interface().(fi.Resource)
			if ok {
				same, err := fi.ResourcesMatch(aResource, eResource)
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

func RandomToken(length int) string {
	// This is supposed to be the same algorithm as the old bash algorithm
	// KUBELET_TOKEN=$(dd if=/dev/urandom bs=128 count=1 2>/dev/null | base64 | tr -d "=+/" | dd bs=32 count=1 2>/dev/null)
	// KUBE_PROXY_TOKEN=$(dd if=/dev/urandom bs=128 count=1 2>/dev/null | base64 | tr -d "=+/" | dd bs=32 count=1 2>/dev/null)

	for {
		buffer := make([]byte, length * 4)
		_, err := crypto_rand.Read(buffer)
		if err != nil {
			glog.Fatalf("error generating random token: %v", err)
		}
		s := base64.StdEncoding.EncodeToString(buffer)
		var trimmed bytes.Buffer
		for _, c := range s {
			switch c {
			case '=', '+', '/':
				continue
			default:
				trimmed.WriteRune(c)
			}
		}

		s = string(trimmed.Bytes())
		if len(s) >= length {
			return s[0:length]
		}
	}
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


