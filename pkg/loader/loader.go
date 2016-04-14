package loader

import (
	"text/template"
	"fmt"
	"bytes"
	"reflect"
	"gopkg.in/yaml.v2"
	"github.com/golang/glog"
	"strings"
	"os"
	"github.com/kopeio/kope/pkg/fi"
	"path"
	"io/ioutil"
)

type deferredType int

const (
	deferredUnit deferredType = iota
	deferredResource
)

type Loader struct {
	typeMap  map[string]reflect.Type
	options  map[string]interface{}
	objects  []interface{}
	deferred []*deferredBinding
}

type deferredBinding struct {
	name         string
	dest         reflect.Value
	src          string
	deferredType deferredType
}

func NewLoader(options map[string]interface{}) *Loader {
	l := &Loader{}
	l.options = options
	l.typeMap = make(map[string]reflect.Type)
	return l
}

func (l*Loader) AddType(key string, t reflect.Type) {
	_, exists := l.typeMap[key]
	if exists {
		glog.Fatalf("duplicate type key: %q", key)
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	l.typeMap[key] = t
}

func (l*Loader) Load(key string, d string) (error) {
	data, err := l.executeTemplate(key, d)
	if err != nil {
		return err
	}
	objects, err := l.loadYaml(key, data)
	if err != nil {
		return err
	}

	l.objects = append(l.objects, objects...)
	return nil
}

func (l*Loader) executeTemplate(key string, d string) ([]byte, error) {
	t := template.New(key)

	funcMap := make(template.FuncMap)

	t.Funcs(funcMap)

	_, err := t.Parse(d)
	if err != nil {
		return nil, fmt.Errorf("error parsing template %q: %v", key, err)
	}

	t.Option("missingkey=zero")

	var buffer bytes.Buffer
	err = t.ExecuteTemplate(&buffer, key, l.options)
	if err != nil {
		return nil, fmt.Errorf("error executing template %q: %v", key, err)
	}

	return buffer.Bytes(), nil
}

func (l*Loader) Build() ([]interface{}, error) {
	if len(l.deferred) != 0 {
		unitMap := make(map[string]fi.Unit)
		resourceMap := make(map[string]fi.Resource)

		for _, o := range l.objects {
			if unit, ok := o.(fi.Unit); ok {
				path, err := buildPath(unit)
				if err != nil {
					return nil, err
				}
				path = strings.ToLower(path)
				unitMap[path] = unit
			}
		}

		for _, d := range l.deferred {
			src := strings.ToLower(d.src)

			switch d.deferredType {
			case deferredUnit:
				unit, found := unitMap[src]
				if !found {
					glog.Infof("Known targets:")
					for k, _ := range unitMap {
						glog.Infof("  %s", k)
					}
					return nil, fmt.Errorf("cannot resolve link at %q to %q", d.name, d.src)
				}

				d.dest.Set(reflect.ValueOf(unit))

			case deferredResource:
				resource, found := resourceMap[src]
				if !found {
					glog.Infof("Known resources:")
					for k, _ := range resourceMap {
						glog.Infof("  %s", k)
					}
					return nil, fmt.Errorf("cannot resolve resource link at %q to %q", d.name, d.src)
				}

				d.dest.Set(reflect.ValueOf(resource))

			default:
				panic("unhandled deferred type")
			}
		}
	}

	return l.objects, nil
}

func isOptions(name string) bool {
	return strings.HasSuffix(name, ".options") || strings.HasSuffix(name, ".options.yaml")
}

func (l*Loader) WalkDirectory(baseDir string) error {
	files, err := ioutil.ReadDir(baseDir)
	if err != nil {
		return fmt.Errorf("error reading directory %q: %v", baseDir, err)
	}

	for _, f := range files {
		if isOptions(f.Name()) {
			continue
		}
		p := path.Join(baseDir, f.Name())
		contents, err := ioutil.ReadFile(p)
		if err != nil {
			return fmt.Errorf("error loading file %q: %v", p, err)
		}
		err = l.Load(f.Name(), string(contents))
		if err != nil {
			return fmt.Errorf("error processing file %q: %v", p, err)
		}
	}
	return nil
}

func buildPath(u fi.Unit) (string, error) {
	t := reflect.TypeOf(u)
	if t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface {
		t = t.Elem()
	}
	typeName := t.Name()
	name, err := getName(u)
	if err != nil {
		return "", err
	}
	return typeName + "/" + name, nil
}

func getName(u fi.Unit) (string, error) {
	v := reflect.ValueOf(u)
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	nameField := v.FieldByName("Name")
	if !nameField.IsValid() {
		return "", fmt.Errorf("cannot determine Name for %T", u)
	}
	if nameField.Kind() == reflect.Ptr || nameField.Kind() == reflect.Interface {
		nameField = nameField.Elem()
	}
	return fmt.Sprintf("%s", nameField.Interface()), nil
}

func (l*Loader) loadYaml(key string, data []byte) ([]interface{}, error) {
	var o map[string]interface{}
	err := yaml.Unmarshal(data, &o)
	if err != nil {
		// TODO: It would be nice if yaml returned us the line number here
		glog.Infof("error parsing yaml.  yaml follows:")
		for i, line := range strings.Split(string(data), "\n") {
			fmt.Fprintf(os.Stderr, "%3d: %s\n", i, line)
		}
		return nil, fmt.Errorf("error parsing yaml %q: %v", key, err)
	}

	return l.loadMap(key, o)
}

func (l*Loader) loadMap(key string, data map[string]interface{}) ([]interface{}, error) {
	var loaded []interface{}
	for k, v := range data {
		t, found := l.typeMap[k]
		if !found {
			return nil, fmt.Errorf("unknown type %q (in %q)", k, key)
		}

		o := reflect.New(t)
		err := l.populateObject(key + ":" + k, o, v)
		if err != nil {
			return nil, err
		}

		loaded = append(loaded, o.Interface())
	}
	return loaded, nil
}

func (l*Loader) populateObject(name string, dest reflect.Value, src interface{}) error {
	m, ok := src.(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("unexpected type of source data for %q: %T", name, src)
	}

	if dest.Kind() == reflect.Ptr && !dest.IsNil() {
		dest = dest.Elem()
	}

	// TODO: Pre-calculate
	destType := dest.Type()
	fieldMap := map[string]reflect.StructField{}
	for i := 0; i < destType.NumField(); i++ {
		f := destType.Field(i)
		fieldName := f.Name
		fieldName = strings.ToLower(fieldName)
		_, exists := fieldMap[fieldName]
		if exists {
			glog.Fatalf("ambiguous field name in %q: %q", destType.Name(), fieldName)
		}
		fieldMap[fieldName] = f
	}

	//t := dest.Type()
	for k, v := range m {
		kString, ok := k.(string)
		if !ok {
			return fmt.Errorf("unexpected key type %T in %q", k, name)
		}
		kString = strings.ToLower(kString)
		fieldInfo, found := fieldMap[kString]
		if !found {
			return fmt.Errorf("unknown field %q in %q", k, name)
		}
		field := dest.FieldByIndex(fieldInfo.Index)

		err := l.populateValue(name + "." + kString, field, v)
		if err != nil {
			return err
		}
	}

	return nil
}

//func (l*Loader) populateField(name string, dest reflect.Value, src interface{}) error {
//	return
//	if src == nil {
//		return nil
//	}
//
//	// Divert special cases
//	switch dest.Type().Kind() {
//	case reflect.Map:
//		return l.populateMap(name, dest, src)
//	}
//
//	var destTypeName string
//	switch dest.Type().Kind() {
//	case reflect.Ptr:
//		destTypeName = "*" + dest.Type().Elem().Name()
//	case reflect.Slice:
//		destTypeName = "[]" + dest.Type().Elem().Name()
//	default:
//		glog.Fatalf("unhandled destination type: %v", dest.Type())
//	}
//
//	switch src := src.(type) {
//	case bool:
//		dest.SetBool(src)
//	case int:
//		switch destTypeName {
//		case "*int64":
//			s := int64(src)
//			val := reflect.ValueOf(&s)
//			dest.Set(val)
//		case "*int":
//			s := src
//			val := reflect.ValueOf(&s)
//			dest.Set(val)
//		default:
//			return fmt.Errorf("unhandled destination type for %q: %s", name, destTypeName)
//		}
//	case string:
//		switch destTypeName {
//		case "*string":
//			s := src
//			val := reflect.ValueOf(&s)
//			dest.Set(val)
//		case "[]string":
//			// We allow a single string to populate an array
//			s := []string{src}
//			val := reflect.ValueOf(s)
//			dest.Set(val)
//		default:
//			if unit, ok := dest.Interface().(fi.Unit); ok {
//				glog.Errorf("unit linking not yet implemented %q %T", name, unit)
//			} else {
//				return fmt.Errorf("unhandled destination type for %q: %s", name, destTypeName)
//			}
//		}
//	default:
//		return fmt.Errorf("unhandled type for %q: %T", name, src)
//	}
//
//	return nil
//}

func (l*Loader) populateMap(name string, dest reflect.Value, src interface{}) error {
	if src == nil {
		return nil
	}

	glog.Infof("populateMap on type %s", BuildTypeName(dest.Type()))

	destType := dest.Type()

	if destType.Kind() == reflect.Ptr {
		destType = destType.Elem()
		dest = dest.Elem()
	}

	if dest.IsNil() {
		dest = reflect.MakeMap(dest.Type())
	}

	srcMap, ok := src.(map[interface{}]interface{})
	if ok {
		for k, v := range srcMap {
			kString, ok := k.(string)
			if !ok {
				return fmt.Errorf("unexpected type for map key in %q: %T", name, k)
			}

			entryValue := reflect.New(destType.Elem()).Elem()
			err := l.populateValue(name + "." + kString, entryValue, v)
			if err != nil {
				return err
			}
			dest.SetMapIndex(reflect.ValueOf(k), entryValue)
		}
		return nil
	}

	return fmt.Errorf("unexpected source type for map %q: %T", name, src)
}

func (l*Loader) populateValue(name string, dest reflect.Value, src interface{}) error {
	if src == nil {
		return nil
	}

	// Divert special cases
	switch dest.Type().Kind() {
	case reflect.Map:
		return l.populateMap(name, dest, src)
	case reflect.Ptr:
		elemType := dest.Type().Elem()
		switch (elemType.Kind()) {
		case reflect.Map:
			return l.populateMap(name, dest.Elem(), src)
		}
	}

	destTypeName := BuildTypeName(dest.Type())

	switch destTypeName {
	case "*string": {
		switch src := src.(type) {
		case string:
			v := src
			dest.Set(reflect.ValueOf(&v))
			return nil
		default:
			return fmt.Errorf("unhandled conversion for %q: %T -> %s", name, src, destTypeName)
		}
	}

	case "[]string": {
		switch src := src.(type) {
		case string:
			// We allow a single string to populate an array
			v := []string{src}
			dest.Set(reflect.ValueOf(v))
			return nil
		case []interface{}:
			v := []string{}
			for _, i := range src {
				v = append(v, fmt.Sprintf("%v", i))
			}
			dest.Set(reflect.ValueOf(v))
			return nil
		default:
			return fmt.Errorf("unhandled conversion for %q: %T -> %s", name, src, destTypeName)
		}
	}

	case "*int64": {
		switch src := src.(type) {
		case int:
			v := int64(src)
			dest.Set(reflect.ValueOf(&v))
			return nil
		default:
			return fmt.Errorf("unhandled conversion for %q: %T -> %s", name, src, destTypeName)
		}
	}

	case "*bool": {
		switch src := src.(type) {
		case bool:
			v := bool(src)
			dest.Set(reflect.ValueOf(&v))
			return nil
		default:
			return fmt.Errorf("unhandled conversion for %q: %T -> %s", name, src, destTypeName)
		}
	}

	case "Resource": {
		d := &deferredBinding{
			name: name,
			dest: dest,
			deferredType: deferredResource,
		}
		switch src := src.(type) {
		case string:
			d.src = src
		default:
			return fmt.Errorf("unhandled conversion for %q: %T -> %s", name, src, destTypeName)
		}
		l.deferred = append(l.deferred, d)
		return nil
	}

	default:
		if _, ok := dest.Interface().(fi.Unit); ok {
			d := &deferredBinding{
				name: name,
				dest: dest,
				deferredType: deferredUnit,
			}
			switch src := src.(type) {
			case string:
				d.src = src
			default:
				return fmt.Errorf("unhandled conversion for %q: %T -> %s", name, src, destTypeName)
			}
			l.deferred = append(l.deferred, d)
			return nil
		} else {
			return fmt.Errorf("unhandled destination type for %q: %s", name, destTypeName)
		}
	}

}

func BuildTypeName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Ptr:
		return "*" + BuildTypeName(t.Elem())
	case reflect.Slice:
		return "[]" + BuildTypeName(t.Elem())
	case reflect.Struct, reflect.Interface:
		return t.Name()
	case reflect.String, reflect.Bool, reflect.Int64:
		return t.Name()
	case reflect.Map:
		return "map[" + BuildTypeName(t.Key()) + "]" + BuildTypeName(t.Elem())
	default:
		glog.Errorf("cannot find type name for: %v, assuming %s", t, t.Name())
		return t.Name()
	}
}


