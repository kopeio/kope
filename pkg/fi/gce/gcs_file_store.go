package gce

import (
	"encoding/hex"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
)

type GCSFileStore struct {
	bucket *GCSBucket
	prefix string
}

func NewGCSFileStore(bucket *GCSBucket, prefix string) *GCSFileStore {
	return &GCSFileStore{
		bucket: bucket,
		prefix: prefix,
	}
}

func (s *GCSFileStore) PutResource(key string, r fi.Resource, hashAlgorithm fi.HashAlgorithm) (string, string, error) {
	hashes, err := fi.HashesForResource(r, []fi.HashAlgorithm{fi.HashAlgorithmMD5, hashAlgorithm})
	if err != nil {
		return "", "", err
	}

	md5 := hashes[fi.HashAlgorithmMD5]
	userHash := hashes[hashAlgorithm]

	gcsKey := s.prefix + key + "-" + userHash
	o, err := s.bucket.FindObjectIfExists(gcsKey)
	if err != nil {
		return "", "", err
	}

	alreadyPresent := false

	objectHash := ""
	if o != nil {
		objectHashBytes, err := o.Md5Hash()
		if err != nil {
			return "", "", err
		}
		objectHash = hex.EncodeToString(objectHashBytes)
		if objectHash == md5 {
			alreadyPresent = true
		} else {
			glog.Infof("Found file, but did not match: %q (%s vs %s)", o, objectHash, md5)
		}
	}

	if !alreadyPresent {
		body, err := r.Open()
		if err != nil {
			return "", "", err
		}
		defer fi.SafeClose(body)

		o, err = s.bucket.PutObject(gcsKey, body)
		if err != nil {
			return "", "", err
		}
	}

	isPublic, err := o.IsPublic()
	if err != nil {
		return "", "", err
	}
	if !isPublic {
		err = o.SetPublicACL()
		if err != nil {
			return "", "", err
		}
	}

	return o.PublicURL(), userHash, nil
}
