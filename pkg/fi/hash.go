package fi

import (
	"hash"
	"crypto/md5"
	"crypto/sha256"
	"github.com/golang/glog"
	"crypto/sha1"
)

type  HashAlgorithm string

const (
	HashAlgorithmSHA256 = "sha256"
	HashAlgorithmSHA1 = "sha1"
	HashAlgorithmMD5 = "md5"
)

func NewHasher(hashAlgorithm HashAlgorithm) hash.Hash {
	switch hashAlgorithm {
	case HashAlgorithmMD5:
		return md5.New()

	case HashAlgorithmSHA1:
		return sha1.New()

	case HashAlgorithmSHA256:
		return sha256.New()
	}

	glog.Exitf("Unknown hash algorithm: %v", hashAlgorithm)
	return nil
}