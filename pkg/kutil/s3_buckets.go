package kutil

import (
	"fmt"
	"crypto/md5"
	"encoding/hex"
	"strings"
	"github.com/kopeio/kope/pkg/fi"
	"os"
	"github.com/kopeio/kope/pkg/fi/gce"
)

func GetDefaultS3Bucket(cloud *fi.AWSCloud) (string, error) {
	credentials, err := cloud.EC2.Config.Credentials.Get()
	if err != nil {
		return "", fmt.Errorf("error fetching EC2 credentials")
	}

	user := os.Getenv("USER")

	hasher := md5.New()
	hasher.Write([]byte(user))
	hasher.Write([]byte(" "))
	hasher.Write([]byte(credentials.AccessKeyID))
	hash := hasher.Sum(nil)
	hashString := hex.EncodeToString(hash[:])
	hashString = strings.ToLower(hashString)
	return "kubernetes-staging-" + hashString, nil
}

func GetDefaultGCSBucket(cloud *gce.GCECloud, location string) (string, error) {
	suffix := "-" + location
	if suffix == "-us" {
		suffix = ""
	}
	hasher := md5.New()
	hasher.Write([]byte(cloud.Project))
	hash := hasher.Sum(nil)
	projectHash := hex.EncodeToString(hash[:])
	projectHash = strings.ToLower(projectHash)

	bucket := "kubernetes-staging-" + projectHash[0:10] + suffix
	return bucket, nil
}