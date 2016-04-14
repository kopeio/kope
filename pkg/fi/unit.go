package fi

type Unit interface {
	Run(c *RunContext) error
	Path() string
}

type HasKey interface {
	Key() string
}

type HasID interface {
	GetID() *string
}

type KeyedUnit interface {
	Unit
	HasKey
}

type KeyAware interface {
	SetKey(key string)
}
