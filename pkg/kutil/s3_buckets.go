package kutil

import (
	"fmt"
	"crypto/md5"
	"encoding/hex"
	"strings"
	"github.com/kopeio/kope/pkg/fi"
)

func GetDefaultS3Bucket(cloud *fi.AWSCloud) (string, error) {
	credentials, err := cloud.EC2.Config.Credentials.Get()
	if err != nil {
		return "", fmt.Errorf("error fetching EC2 credentials")
	}

	hash := md5.Sum([]byte(credentials.AccessKeyID))
	hashString := hex.EncodeToString(hash[:])
	hashString = strings.ToLower(hashString)
	return "kubernetes-staging-" + hashString, nil
}
