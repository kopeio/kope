package fi

import (
	"github.com/golang/glog"
	"reflect"
	"fmt"
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

	dirty  bool
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
		Target: c.Target,
		parent:  c,
		node:    n,
		mode:    c.mode,
	}
	return child
}

func (c *RunContext) Run() error {
	return c.node.Run(c)
}

func (c*RunContext) Render(a, e, changes Unit) error {
	dryrun := false

	var methodName string
	switch c.Target.(type) {
	case *AWSAPITarget:
		methodName = "RenderAWS"
	case *BashTarget:
		methodName = "RenderBash"
	case *DryRunTarget:
		dryrun= true
	default:
		return fmt.Errorf("Unhandled target type: %T", c.Target)
	}

	if dryrun {
		return c.Target.(*DryRunTarget).Render(a, e, changes)
	}

	v := reflect.ValueOf(e)
	vType := v.Type()

	_, found := vType.MethodByName(methodName)
	if !found {
		return fmt.Errorf("Type does not support Render function %s: %T", methodName, v.Type())
	}
	var args  []reflect.Value
	args = append(args, reflect.ValueOf(c.Target))
	args = append(args, reflect.ValueOf(a))
	args = append(args, reflect.ValueOf(e))
	args = append(args, reflect.ValueOf(changes))
	glog.V(4).Infof("Calling method %s on %T", methodName, e)
	m := v.MethodByName(methodName)
	rv := m.Call(args)
	var rvErr error
	if !rv[0].IsNil() {
		rvErr = rv[0].Interface().(error)
	}
	return rvErr
}