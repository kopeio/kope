package awsunits

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
)

type IAMInstanceProfile struct {
	fi.SimpleUnit

	ID   *string
	Name *string
}

func (s *IAMInstanceProfile) Key() string {
	return *s.Name
}

func (s *IAMInstanceProfile) GetID() *string {
	return s.ID
}

func (e *IAMInstanceProfile) find(c *fi.RunContext) (*IAMInstanceProfile, error) {
	cloud := c.Cloud().(*fi.AWSCloud)

	request := &iam.GetInstanceProfileInput{InstanceProfileName: e.Name}

	response, err := cloud.IAM.GetInstanceProfile(request)
	if err != nil {
		return nil, fmt.Errorf("error getting IAMInstanceProfile: %v", err)
	}

	ip := response.InstanceProfile
	actual := &IAMInstanceProfile{}
	actual.ID = ip.InstanceProfileId
	actual.Name = ip.InstanceProfileName
	return actual, nil
}

func (e *IAMInstanceProfile) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	if a != nil {
		if e.ID == nil {
			e.ID = a.ID
		}
	}

	changes := &IAMInstanceProfile{}
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

func (s *IAMInstanceProfile) checkChanges(a, e, changes *IAMInstanceProfile) error {
	if a != nil {
		if e.Name == nil {
			return fi.MissingValueError("Name is required when creating IAMInstanceProfile")
		}
	}
	return nil
}

func (_*IAMInstanceProfile) RenderAWS(t *fi.AWSAPITarget, a, e, changes *IAMInstanceProfile) error {
	if a == nil {
		glog.V(2).Infof("Creating IAMInstanceProfile with Name:%q", *e.Name)

		request := &iam.CreateInstanceProfileInput{}
		request.InstanceProfileName = e.Name

		response, err := t.Cloud.IAM.CreateInstanceProfile(request)
		if err != nil {
			return fmt.Errorf("error creating IAMInstanceProfile: %v", err)
		}

		e.ID = response.InstanceProfile.InstanceProfileId
	}

	return nil //return output.AddAWSTags(cloud.Tags(), v, "vpc")
}

func (_*IAMInstanceProfile) RenderBash(t *fi.BashTarget, a, e, changes *IAMInstanceProfile) error {
	t.CreateVar(e)
	if a == nil {
		glog.V(2).Infof("Creating IAMInstanceProfile with Name:%q", *e.Name)

		t.AddIAMCommand("create-instance-profile",
			"--instance-profile-name", *e.Name)
	}

	return nil
}
