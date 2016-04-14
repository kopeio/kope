package fi

import (
	"fmt"

	"bytes"
	"github.com/golang/glog"
	"io"
	"reflect"
)

type putResource struct {
	Key  string
	Hash string
}

type render struct {
	a       Unit
	aIsNil  bool
	e       Unit
	changes Unit
}

type DryRunTarget struct {
	putResources map[string]*putResource
	changes      []*render
	filestore    *dryRunFilestore
}

var _ Target = &DryRunTarget{}

type dryRunFilestore struct {
	target *DryRunTarget
}

func (t *dryRunFilestore) PutResource(key string, r Resource, hashAlgorithm HashAlgorithm) (string, string, error) {
	if r == nil {
		glog.Fatalf("Attempt to put null resource for %q", key)
	}
	url := "dryrun://" + key
	hash, err := HashForResource(r, hashAlgorithm)
	if err != nil {
		return "", "", fmt.Errorf("error hashing resource %q: %v", key, err)
	}
	t.target.putResources[key+":"+hash] = &putResource{
		Key:  key,
		Hash: hash,
	}

	return url, hash, nil
}

var _ FileStore = &dryRunFilestore{}

func NewDryRunTarget() (*DryRunTarget, error) {
	t := &DryRunTarget{}
	t.putResources = make(map[string]*putResource)
	t.filestore = &dryRunFilestore{target: t}
	return t, nil
}

func (t *DryRunTarget) Render(a, e, changes Unit) error {
	valA := reflect.ValueOf(a)
	aIsNil := valA.IsNil()

	t.changes = append(t.changes, &render{
		a:       a,
		aIsNil:  aIsNil,
		e:       e,
		changes: changes,
	})
	return nil
}

func (t *DryRunTarget) FileStore() FileStore {
	return t.filestore
}

func (t *DryRunTarget) PrintReport(out io.Writer) error {
	b := &bytes.Buffer{}

	if len(t.putResources) != 0 {
		fmt.Fprintf(b, "Upload resources:\n")
		for _, r := range t.putResources {
			fmt.Fprintf(b, "  %s\t%s\n", r.Key, r.Hash)
		}
	}

	if len(t.changes) != 0 {
		fmt.Fprintf(b, "Created resources:\n")
		for _, r := range t.changes {
			if !r.aIsNil {
				continue
			}

			fmt.Fprintf(b, "  %T\t%s\n", r.changes, r.e.Path())
		}

		fmt.Fprintf(b, "Changed resources:\n")
		for _, r := range t.changes {
			if r.aIsNil {
				continue
			}
			var changeList []string

			valC := reflect.ValueOf(r.changes)
			valA := reflect.ValueOf(r.a)
			valE := reflect.ValueOf(r.e)
			if valC.Kind() == reflect.Ptr && !valC.IsNil() {
				valC = valC.Elem()
			}
			if valA.Kind() == reflect.Ptr && !valA.IsNil() {
				valA = valA.Elem()
			}
			if valE.Kind() == reflect.Ptr && !valE.IsNil() {
				valE = valE.Elem()
			}
			if valC.Kind() == reflect.Struct {
				for i := 0; i < valC.NumField(); i++ {
					fieldValC := valC.Field(i)

					if (fieldValC.Kind() == reflect.Ptr || fieldValC.Kind() == reflect.Slice || fieldValC.Kind() == reflect.Map) && fieldValC.IsNil() {
						// No change
						continue
					}

					fieldValE := valE.Field(i)

					description := ""
					ignored := false
					if fieldValE.CanInterface() {
						fieldValA := valA.Field(i)

						switch fieldValE.Interface().(type) {
						case SimpleUnit:
							ignored = true
						default:
							description = fmt.Sprintf(" %v -> %v", asString(fieldValA), asString(fieldValE))
						}
					}
					if ignored {
						continue
					}
					changeList = append(changeList, valC.Type().Field(i).Name+description)
				}
			} else {
				return fmt.Errorf("unhandled change type: %v", valC.Type())
			}

			if len(changeList) == 0 {
				continue
			}

			fmt.Fprintf(b, "  %T\t%s\n", r.changes, r.e.Path())
			for _, f := range changeList {
				fmt.Fprintf(b, "    %s\n", f)
			}
			fmt.Fprintf(b, "\n")
		}
	}

	_, err := out.Write(b.Bytes())
	return err
}

func asString(v reflect.Value) string {
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return "<nil>"
		}
		//v = v.Elem()
		//if v.Kind() == reflect.Ptr && v.IsNil() {
		//	return "<nil>"
		//}
	}
	if v.CanInterface() {
		iv := v.Interface()
		_, isResource := iv.(Resource)
		if isResource {
			return "<resource>"
		}
		_, isHasID := iv.(HasID)
		if isHasID {
			id := iv.(HasID).GetID()
			if id == nil {
				return "id:<nil>"
			} else {
				return "id:" + *id
			}
		}
		switch iv.(type) {
		case *string:
			return *(iv.(*string))
		default:
			return fmt.Sprintf("%T (%v)", iv, iv)
		}

	} else {
		return fmt.Sprintf("Unhandled: %T", v.Type())

	}
}
