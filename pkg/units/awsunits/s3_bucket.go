package awsunits

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
	"github.com/aws/aws-sdk-go/aws"
)

type S3Bucket struct {
	fi.SimpleUnit

	Name   *string
	Region *string

	//rendered bool
	//exists   bool
}

func (s *S3Bucket) Prefix() string {
	return "S3Bucket"
}

func (e*S3Bucket) findBucketIfExists(c *fi.RunContext) (*fi.S3Bucket, error) {
	cloud := c.Cloud().(*fi.AWSCloud)

	return cloud.S3.FindBucketIfExists(*e.Name)
}

func (e *S3Bucket) find(c *fi.RunContext) (*S3Bucket, error) {
	bucket, err := e.findBucketIfExists(c)
	if err != nil {
		return nil, err
	}
	if bucket == nil {
		return nil, nil
	}

	glog.V(2).Info("found existing S3 bucket")
	actual := &S3Bucket{}
	actual.Name = e.Name
	actual.Region = aws.String(bucket.Region())
	return actual, nil
}

func (e *S3Bucket) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	changes := &S3Bucket{}
	changed := fi.BuildChanges(a, e, changes)
	if !changed {
		return nil
	}

	err = e.checkChanges(a, e, changes)
	if err != nil {
		return err
	}

	return c.Render(a, e, changes)
}

func (s *S3Bucket) checkChanges(a, e, changes *S3Bucket) error {
	if a != nil {
		if e.Name == nil {
			return fi.MissingValueError("Name is required when creating S3Bucket")
		}
		if changes.Region != nil {
			return fi.InvalidChangeError("Cannot change region of existing S3Bucket", a.Region, e.Region)
		}
	}
	return nil
}

func (_*S3Bucket) RenderAWS(t *fi.AWSAPITarget, a, e, changes *S3Bucket) error {
	if a == nil {
		glog.V(2).Infof("Creating S3Bucket with Name:%q", *e.Name)

		request := &s3.CreateBucketInput{}
		request.Bucket = e.Name

		_, err := t.Cloud.GetS3(*e.Region).CreateBucket(request)
		if err != nil {
			return fmt.Errorf("error creating S3Bucket: %v", err)
		}
	}

	return nil //return output.AddAWSTags(cloud.Tags(), v, "vpc")
}

func (_*S3Bucket) RenderBash(t *fi.BashTarget, a, e, changes *S3Bucket) error {
	if a == nil {
		glog.V(2).Infof("Creating S3Bucket with Name:%q", *e.Name)

		args := []string{"mb"}
		args = append(args, "s3://" + *e.Name)

		t.AddS3Command(*e.Region, args...)
	}

	return nil
}

//func (b *S3Bucket) Region() string {
//	if !b.rendered {
//		glog.Fatalf("not yet rendered")
//	}
//	return b.region
//}

//func (b *S3Bucket) exists() bool {
//	if !b.rendered {
//		glog.Fatalf("not yet rendered")
//	}
//	return b.exists
//}
