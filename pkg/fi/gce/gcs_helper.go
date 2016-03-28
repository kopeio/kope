package gce

import (
	"fmt"
	"github.com/golang/glog"
	storage "google.golang.org/api/storage/v1"
)

//type GCSHelper struct {
//	//config    *aws.Config
//	//defaultS3 *s3.S3
//	//regions   map[string]*storage.Service
//	service *storage.Service
//}
//
//func NewGCSHelper(service *storage.Service) *GCSHelper {
//	s := &GCSHelper{
//		service: service,
//		//config: defaultConfig.Copy(),
//		//defaultS3: s3.New(session.New(), defaultConfig),
//		//regions: make(map[string]*s3.S3),
//	}
//
//	return s
//}

//func (s*GCSHelper) GetService(region string) *storage.Service {
//	client, found := s.regions[region]
//	if !found {
//		config := s.config.Copy().WithRegion(region)
//		client = s3.New(session.New(), config)
//		s.regions[region] = client
//	}
//	return client
//}

func FindGCSBucketIfExists(service *storage.Service, name string) (*GCSBucket, error) {
	glog.V(2).Infof("Getting location of GCS bucket: %s", name)

	meta, err := service.Buckets.Get(name).Do()
	if err != nil {
		return nil, fmt.Errorf("error listing GCS bucket: %v", err)
	}

	bucket := &GCSBucket{
		service: service,
		Name: name,
		meta: meta,
	}
	return bucket, nil
}

func EnsureGCSBucket(service *storage.Service, project string, name string, location string) (*GCSBucket, error) {
	bucket, err := FindGCSBucketIfExists(service, name)
	if err != nil {
		return nil, err
	}
	if bucket == nil {
		glog.V(2).Infof("Creating GCE bucket: %s", name)
		request := &storage.Bucket{
			Name: name,
			Location: location,
		}
		meta, err := service.Buckets.Insert(project, request).Do()
		if err != nil {
			return nil, fmt.Errorf("error creating GCS bucket: %v", err)
		}
		bucket = &GCSBucket{
			service: service,
			Name : name,
			meta: meta,
		}
	}
	return bucket, nil
}
