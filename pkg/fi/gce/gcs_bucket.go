package gce

import (
	storage "google.golang.org/api/storage/v1"
	"io"
)

type GCSBucket struct {
	service *storage.Service
	Name    string
	meta    *storage.Bucket
}

func (b *GCSBucket) FindObjectIfExists(key string) (*GCSObject, error) {
	object, err := b.service.Objects.Get(b.Name, key).Do()
	if err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &GCSObject{Bucket: b, Key: key, meta: object}, nil
}

func (b *GCSBucket) PutObject(key string, body io.ReadSeeker) (*GCSObject, error) {
	o := &GCSObject{
		Bucket: b,
		Key:    key,
	}
	err := o.putObject(body)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (b *GCSBucket) PublicURL() string {
	return "https://storage.googleapis.com/" + b.Name + "/"
}
