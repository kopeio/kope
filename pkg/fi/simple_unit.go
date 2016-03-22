package fi

type SimpleUnit struct {
	parent Unit
	key    string
}

var _ Unit = &SimpleUnit{}
var _ KeyAware = &SimpleUnit{}

func (u *SimpleUnit) Run(c *RunContext) error {
	return nil
}

func (u *SimpleUnit) Path() string {
	if u.parent == nil {
		return u.key
	}
	return u.parent.Path() + "/" + u.key
}

func (u *SimpleUnit) SetKey(key string) {
	u.key = key
}