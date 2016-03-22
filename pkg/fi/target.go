package fi

type Target interface {
	PutResource(key string, resource Resource, hashAlgorithm HashAlgorithm) (url string, hash string, err error)
}
