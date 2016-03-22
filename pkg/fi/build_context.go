package fi

import (
	"github.com/golang/glog"
	"reflect"
)

type BuildContext struct {
	*Context
	node *node
}

type Builder interface {
	Add(*BuildContext)
}

func GetTypeName(unit interface{}) string {
	t := reflect.TypeOf(unit)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

func (b *BuildContext) Add(unit Unit) {
	childNode := b.newNode(unit)

	key := ""
	hk, ok := unit.(HasKey)
	if ok {
		key = hk.Key()
	}

	if key == "" {
		glog.Exitf("could not determine key for %T", unit)
	}

	key = GetTypeName(unit) + "-" + key
	ka, ok := unit.(KeyAware)
	if ok {
		ka.SetKey(key)
	} else {
		glog.Exitf("could not determine set key for %T", unit)
	}

	builder, ok := unit.(Builder)
	if ok {
		childContext := b.createChildContext(childNode)
		builder.Add(childContext)
	}

	b.node.Add(childNode)
}

func (b *BuildContext) createChildContext(childNode *node) *BuildContext {
	bc := &BuildContext{
		Context: b.Context,
		node:    childNode,
	}
	return bc
}
