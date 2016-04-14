package gce

import (
	"encoding/base64"
	"fmt"
	"github.com/golang/glog"
	"google.golang.org/api/storage/v1"
	"io"
)

type GCSObject struct {
	Bucket *GCSBucket
	Key    string
	meta   *storage.Object
}

func (o *GCSObject) String() string {
	return fmt.Sprintf("gs://%s/%s", o.Bucket.Name, o.Key)
}

func (o *GCSObject) IsPublic() (bool, error) {
	glog.V(2).Infof("Getting GCS object ACL: %s", o)

	service := o.Bucket.service

	entity := "allUsers"
	acl, err := service.ObjectAccessControls.Get(o.Bucket.Name, o.Key, entity).Do()
	if err != nil {
		if IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("error reading GCS ACL for %q: %v", o, err)
	}

	if acl.Role == "READER" {
		return true, nil
	}
	if acl.Role == "OWNER" {
		// TODO: Return a richer model
		return true, nil
	}
	return false, fmt.Errorf("unkonwn role for %s: %s", o, acl.Role)
}

func (o *GCSObject) Md5Hash() ([]byte, error) {
	h := o.meta.Md5Hash
	hashBytes, err := base64.StdEncoding.DecodeString(h)
	if err != nil {
		return nil, fmt.Errorf("cannot decode MD5Hash %q for object %q", h, o.Key)
	}
	return hashBytes, nil
}

func (o *GCSObject) PublicURL() string {
	bucketBase := o.Bucket.PublicURL()
	return bucketBase + o.Key
}

func (o *GCSObject) SetPublicACL() error {
	glog.V(2).Infof("Setting GCS object ACL: %s", o)

	service := o.Bucket.service

	acl := &storage.ObjectAccessControl{
		Role: "READER",
	}
	entity := "allUsers"
	_, err := service.ObjectAccessControls.Update(o.Bucket.Name, o.Key, entity, acl).Do()
	if err != nil {
		return fmt.Errorf("error setting GCS ACL for %q: %v", o, err)
	}

	return nil
}

func (o *GCSObject) putObject(body io.ReadSeeker) error {
	glog.Infof("Uploading object to %q", o)
	meta := &storage.Object{
		Name: o.Key,
	}
	response, err := o.Bucket.service.Objects.Insert(o.Bucket.Name, meta).Media(body).Do()
	if err != nil {
		return fmt.Errorf("error uploading GCS object %q: %v", o, err)
	}
	o.meta = response
	return nil
}
