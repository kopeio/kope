package awsunits

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
)

type IAMInstanceProfileRole struct {
	fi.SimpleUnit

	InstanceProfile *IAMInstanceProfile
	Role            *IAMRole
}

func (s *IAMInstanceProfileRole) Key() string {
	return s.InstanceProfile.Key() + "-" + s.Role.Key()
}

func (e *IAMInstanceProfileRole) find(c *fi.RunContext) (*IAMInstanceProfileRole, error) {
	cloud := c.Cloud().(*fi.AWSCloud)

	if e.Role == nil || e.Role.ID == nil {
		glog.V(2).Infof("Role/RoleID not set")
		return nil, nil
	}
	roleID := *e.Role.ID

	request := &iam.GetInstanceProfileInput{InstanceProfileName: e.InstanceProfile.Name}

	response, err := cloud.IAM.GetInstanceProfile(request)
	if err != nil {
		return nil, fmt.Errorf("error getting IAMInstanceProfile: %v", err)
	}

	ip := response.InstanceProfile
	for _, role := range ip.Roles {
		if aws.StringValue(role.RoleId) != roleID {
			continue
		}
		actual := &IAMInstanceProfileRole{}
		actual.InstanceProfile = &IAMInstanceProfile{ID: ip.InstanceProfileId}
		actual.Role = &IAMRole{ID: role.RoleId}
		return actual, nil
	}
	return nil, nil
}

func (e *IAMInstanceProfileRole) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	changes := &IAMInstanceProfileRole{}
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

func (s *IAMInstanceProfileRole) checkChanges(a, e, changes *IAMInstanceProfileRole) error {
	if a != nil {
		if e.Role == nil {
			return fi.MissingValueError("Role is required when creating IAMInstanceProfileRole")
		}
		if e.InstanceProfile == nil {
			return fi.MissingValueError("InstanceProfile is required when creating IAMInstanceProfileRole")
		}
	}
	return nil
}

func (_*IAMInstanceProfileRole) RenderAWS(t *fi.AWSAPITarget, a, e, changes *IAMInstanceProfileRole) error {
	if a == nil {
		request := &iam.AddRoleToInstanceProfileInput{}
		request.InstanceProfileName = e.InstanceProfile.Name
		request.RoleName = e.Role.Name

		_, err := t.Cloud.IAM.AddRoleToInstanceProfile(request)
		if err != nil {
			return fmt.Errorf("error creating IAMInstanceProfileRole: %v", err)
		}
	}

	return nil //return output.AddAWSTags(cloud.Tags(), v, "vpc")
}

func (_*IAMInstanceProfileRole) RenderBash(t *fi.BashTarget, a, e, changes *IAMInstanceProfileRole) error {
	if a == nil {
		glog.V(2).Infof("Creating IAMInstanceProfileRole")

		t.AddIAMCommand("add-role-to-instance-profile",
			"--instance-profile-name", *e.InstanceProfile.Name,
			"--role-name", *e.Role.Name)
	}

	return nil
}
