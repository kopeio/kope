package fi

import "github.com/golang/glog"

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

func (c*RunContext) Render(a, e, changes interface{}) error {
	panic("not implemented")
}