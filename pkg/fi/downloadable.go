package fi

import ()

type Downloadable interface {
	Resolve(hashAlgorithm HashAlgorithm) (url string, hash string, err error)
}

type CloudDownloadable struct {
	filestore FileStore
	resource  Resource
	key       string

	url       string
	hash      string
}

func NewDownloadableFromResource(filestore FileStore,  key string, resource Resource) *CloudDownloadable {
	return &CloudDownloadable{filestore: filestore, resource: resource, key: key}
}

func (c *CloudDownloadable) Resolve(hashAlgorithm HashAlgorithm) (string, string, error) {
	if c.url != "" {
		return c.url, c.hash, nil
	}
	url, hash, err := c.filestore.PutResource(c.key, c.resource, hashAlgorithm)
	if err != nil {
		return "", "", err
	}
	c.url = url
	c.hash = hash
	return url, hash, err
}
