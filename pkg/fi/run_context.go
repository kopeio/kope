package fi

import (
	"fmt"
	"github.com/golang/glog"
	"reflect"
	"strings"
)

type RunMode int

const (
	ModeConfigure RunMode = iota
	ModeValidate
)

type RunContext struct {
	*Context

	Target Target

	parent *RunContext
	node   *node
	mode   RunMode

	dirty bool
}

func (c *RunContext) MarkDirty() {
	c.dirty = true
	if c.mode == ModeValidate {
		glog.Infof("Configuration needed: %v", c.node.unit)
	}
}

func (c *RunContext) IsConfigure() bool {
	return c.mode == ModeConfigure
}

func (c *RunContext) IsValidate() bool {
	return c.mode == ModeValidate
}

func (c *RunContext) buildChildContext(n *node) *RunContext {
	child := &RunContext{
		Context: c.Context,
		Target:  c.Target,
		parent:  c,
		node:    n,
		mode:    c.mode,
	}
	return child
}

func (c *RunContext) Run() error {
	return c.node.Run(c)
}

func (c *RunContext) Render(a, e, changes Unit) error {
	if _, ok := c.Target.(*DryRunTarget); ok {
		return c.Target.(*DryRunTarget).Render(a, e, changes)
	}

	v := reflect.ValueOf(e)
	vType := v.Type()

	targetType := reflect.ValueOf(c.Target).Type()

	var renderer *reflect.Method
	for i := 0; i < vType.NumMethod(); i++ {
		method := vType.Method(i)
		if !strings.HasPrefix(method.Name, "Render") {
			continue
		}
		match := true
		for j := 0; j < method.Type.NumIn(); j++ {
			arg := method.Type.In(j)
			if arg.ConvertibleTo(vType) {
				continue
			}
			if arg.ConvertibleTo(targetType) {
				continue
			}
			match = false
			break
		}
		if match {
			if renderer != nil {
				return fmt.Errorf("Found multiple Render methods that could be invokved on %T", e)
			}
			renderer = &method
		}

	}
	if renderer == nil {
		return fmt.Errorf("Could not find Render method on type %T (target %T)", e, c.Target)
	}
	var args []reflect.Value
	args = append(args, reflect.ValueOf(c.Target))
	args = append(args, reflect.ValueOf(a))
	args = append(args, reflect.ValueOf(e))
	args = append(args, reflect.ValueOf(changes))
	glog.V(4).Infof("Calling method %s on %T", renderer.Name, e)
	m := v.MethodByName(renderer.Name)
	rv := m.Call(args)
	var rvErr error
	if !rv[0].IsNil() {
		rvErr = rv[0].Interface().(error)
	}
	return rvErr
}
