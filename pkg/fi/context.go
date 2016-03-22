package fi

type Context struct {
	roles     []string
	state     map[string]interface{}

	//os        *OS
	cloud     Cloud
	castore   CAStore
	//config    Config
	resources *ResourcesList

	root      *node
}

//type Context struct {
//	Cloud  *AWSCloud
//	Target Target
//}
//
//func NewContext(target Target, cloud *AWSCloud) *Context {
//	c := &Context{
//		Target: target,
//		Cloud:  cloud,
//	}
//	return c
//}


func NewContext(cloud Cloud, castore CAStore) (*Context, error) {
	c := &Context{
		state:     make(map[string]interface{}),
		//os:        &OS{},
		cloud:     cloud,
		castore: castore,
		resources: &ResourcesList{},
	}

	c.root = &node{}

	//err := c.os.init()
	//if err != nil {
	//	return nil, err
	//}

	return c, nil
}

func (c *Context) NewRunContext(target Target, runMode RunMode) *RunContext {
	rc := &RunContext{
		Context: c,
		Target: target,
		node:    c.root,
		mode:    runMode,
	}
	return rc
}

func (c *Context) NewBuildContext() *BuildContext {
	bc := &BuildContext{
		Context: c,
		node:    c.root,
	}
	return bc
}

func (c *Context) AddRole(role string) {
	c.roles = append(c.roles, role)
}

func (c *Context) HasRole(role string) bool {
	for _, r := range c.roles {
		if r == role {
			return true
		}
	}
	return false
}

//func (c *Context) OS() *OS {
//	return c.os
//}

func (c *Context) Cloud() Cloud {
	return c.cloud
}

func (c *Context) CAStore() CAStore {
	return c.castore
}

func (c *Context) GetState(key string, builder func() (interface{}, error)) (interface{}, error) {
	v := c.state[key]
	if v == nil {
		var err error
		v, err = builder()
		if err != nil {
			return nil, err
		}
		c.state[key] = v
	}
	return v, nil
}

func (c *Context) newNode(unit Unit) *node {
	childNode := &node{
		unit: unit,
	}

	return childNode
}

func (c *Context) Resource(key string) Resource {
	r, found := c.resources.Get(key)
	if !found {
		panic("Resource not found: " + key)
	}
	return r
}

func (c *Context) AddResources(r Resources) {
	c.resources.Add(r)
}
