package awsunits

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
)

type S3File struct {
	fi.SimpleUnit

	Bucket    *S3Bucket
	Key       *string
	Source    fi.Resource
	Public    *bool

	//rendered  bool
	publicURL *string

	etag      *string
}

func (s *S3File) Prefix() string {
	return "S3File"
}

func (e *S3File) find(c *fi.RunContext) (*S3File, error) {
	bucket, err := e.Bucket.findBucketIfExists(c)
	if err != nil {
		return nil, err
	}
	if bucket == nil {
		return nil, nil
	}

	o, err := bucket.FindObjectIfExists(*e.Key)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, nil
	}

	isPublic, err := o.IsPublic()
	if err != nil {
		return nil, err
	}
	etag, err := o.Etag()
	if err != nil {
		return nil, err
	}

	actual := &S3File{}
	actual.Public = &isPublic
	actual.Bucket = e.Bucket
	actual.Key = e.Key
	actual.etag = &etag
	return actual, nil
}

func (e *S3File) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	changes := &S3File{}
	changed := BuildChanges(a, e, changes)
	if !changed {
		return nil
	}

	err = e.checkChanges(a, e, changes)
	if err != nil {
		return err
	}

	return c.Render(a, e, changes)
}

func (s *S3File) checkChanges(a, e, changes *S3File) error {
	if a != nil {
		if e.Key == nil {
			return MissingValueError("Key is required when creating S3File")
		}
	}
	return nil
}

func (_*S3File) RenderAWS(t *fi.AWSAPITarget, a, e, changes *S3File) error {
	panic("S3 Render to AWSAPITarget not implemented")
}

func (_*S3File) RenderBash(t *fi.BashTarget, a, e, changes *S3File) error {
	needToUpload := true

	localPath, err := t.AddLocalResource(e.Source)
	if err != nil {
		return err
	}

	if e.Bucket.Region == nil {
		panic("Bucket region not set")
	}
	region := *e.Bucket.Region

	if a != nil {
		hasher := md5.New()
		f, err := os.Open(localPath)
		if err != nil {
			return err
		}
		defer func() {
			err := f.Close()
			if err != nil {
				glog.Warning("error closing local resource: ", err)
			}
		}()
		if _, err := io.Copy(hasher, f); err != nil {
			return fmt.Errorf("error while hashing local file: %v", err)
		}
		localHash := hex.EncodeToString(hasher.Sum(nil))
		s3Hash := aws.StringValue(a.etag)
		s3Hash = strings.Replace(s3Hash, "\"", "", -1)
		if localHash == s3Hash {
			glog.V(2).Info("s3 files match; skipping upload")
			needToUpload = false
		} else {
			glog.V(2).Infof("s3 file mismatch; will upload (%s vs %s)", localHash, s3Hash)
		}
	}
	if needToUpload {
		// We use put-object instead of cp so that we don't do multipart, so the etag is the simple md5
		args := []string{"put-object"}
		args = append(args, "--bucket", *e.Bucket.Name)
		args = append(args, "--key", *e.Key)
		args = append(args, "--body", localPath)
		t.AddS3APICommand(region, args...)
	}

	publicURL := ""
	if changes.Public != nil {
		// TODO: Check existing?

		if !*changes.Public {
			panic("Only change to make S3File public is implemented")
		}

		args := []string{"put-object-acl"}
		args = append(args, "--bucket", *e.Bucket.Name)
		args = append(args, "--key", *e.Key)
		args = append(args, "--grant-read", "uri=\"http://acs.amazonaws.com/groups/global/AllUsers\"")
		t.AddS3APICommand(region, args...)

		publicURLBase := "https://s3-" + region + ".amazonaws.com"
		if region == "us-east-1" {
			// US Classic does not follow the pattern
			publicURLBase = "https://s3.amazonaws.com"
		}

		publicURL = publicURLBase + "/" + *e.Bucket.Name + "/" + *e.Key
	}

	//e.rendered = true
	e.publicURL = &publicURL

	return nil
}

func (f *S3File) PublicURL() string {
	if f.publicURL == nil {
		panic("S3File not rendered or not public")
	}
	return *f.publicURL
}

func (f *S3File) String() string {
	return fmt.Sprintf("S3File (s3://%s/%s)", *f.Bucket.Name, *f.Key)
}



