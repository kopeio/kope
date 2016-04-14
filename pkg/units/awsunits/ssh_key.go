package awsunits

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
)

type SSHKey struct {
	fi.SimpleUnit

	Name      *string
	PublicKey fi.Resource

	fingerprint *string
}

func (s *SSHKey) Key() string {
	return *s.Name
}

func (s *SSHKey) GetID() *string {
	return s.Name
}

func (k *SSHKey) String() string {
	return fmt.Sprintf("SSHKey (name=%s)", k.Name)
}

func (e *SSHKey) find(c *fi.RunContext) (*SSHKey, error) {
	cloud := c.Cloud().(*fi.AWSCloud)

	request := &ec2.DescribeKeyPairsInput{
		KeyNames: []*string{e.Name},
	}

	response, err := cloud.EC2.DescribeKeyPairs(request)
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == "InvalidKeyPair.NotFound" {
			return nil, nil
		}
	}
	if err != nil {
		return nil, fmt.Errorf("error listing SSHKeys: %v", err)
	}

	if response == nil || len(response.KeyPairs) == 0 {
		return nil, nil
	}

	if len(response.KeyPairs) != 1 {
		return nil, fmt.Errorf("Found multiple SSHKeys with Name %q", *e.Name)
	}

	k := response.KeyPairs[0]

	actual := &SSHKey{}
	actual.Name = k.KeyName
	actual.fingerprint = k.KeyFingerprint

	return actual, nil
}

func (e *SSHKey) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	changes := &SSHKey{}
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

func (s *SSHKey) checkChanges(a, e, changes *SSHKey) error {
	if a != nil {
		if changes.Name != nil {
			return fi.InvalidChangeError("Cannot change SSHKey Name", changes.Name, e.Name)
		}
	}
	return nil
}

func (_ *SSHKey) RenderAWS(t *fi.AWSAPITarget, a, e, changes *SSHKey) error {
	if a == nil {
		glog.V(2).Infof("Creating SSHKey with Name:%q", *e.Name)

		request := &ec2.ImportKeyPairInput{}
		request.KeyName = e.Name
		if e.PublicKey != nil {
			d, err := fi.ResourceAsBytes(e.PublicKey)
			if err != nil {
				return fmt.Errorf("error rendering SSHKey PublicKey: %v", err)
			}
			request.PublicKeyMaterial = d
		}

		response, err := t.Cloud.EC2.ImportKeyPair(request)
		if err != nil {
			return fmt.Errorf("error creating SSHKey: %v", err)
		}

		e.fingerprint = response.KeyFingerprint
	}

	return nil //return output.AddAWSTags(cloud.Tags(), v, "vpc")
}

func (_ *SSHKey) RenderBash(t *fi.BashTarget, a, e, changes *SSHKey) error {
	if a == nil {
		glog.V(2).Infof("Creating SSHKey with Name:%q", *e.Name)

		file, err := t.AddLocalResource(e.PublicKey)
		if err != nil {
			return err
		}
		t.AddEC2Command("import-key-pair", "--key-name", *e.Name, "--public-key-material", "file://"+file)
	}

	return nil
}
