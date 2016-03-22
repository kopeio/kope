package fi

type FileStore interface {
	PutResource(key string, resource Resource, hashAlgorithm HashAlgorithm) (url string, hash string, err error)
}