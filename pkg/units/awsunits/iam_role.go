package awsunits

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/golang/glog"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/kopeio/kope/pkg/fi"
)

type IAMRole struct {
	fi.SimpleUnit

	ID                 *string
	Name               *string
	RolePolicyDocument fi.Resource // "inline" IAM policy
}

func (s *IAMRole) Key() string {
	return *s.Name
}

func (s *IAMRole) GetID() *string {
	return s.ID
}

func (e *IAMRole) find(c *fi.RunContext) (*IAMRole, error) {
	cloud := c.Cloud().(*fi.AWSCloud)

	request := &iam.GetRoleInput{RoleName: e.Name}

	response, err := cloud.IAM.GetRole(request)
	if err != nil {
		return nil, fmt.Errorf("error getting role: %v", err)
	}

	r := response.Role
	actual := &IAMRole{}
	actual.ID = r.RoleId
	actual.Name = r.RoleName
	if r.AssumeRolePolicyDocument != nil {
		actual.RolePolicyDocument = fi.NewStringResource(*r.AssumeRolePolicyDocument)
	}
	glog.V(2).Infof("found matching IAMRole %q", *actual.ID)
	return actual, nil
}

func (e *IAMRole) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	if a != nil && e.ID == nil {
		e.ID = a.ID
	}

	changes := &IAMRole{}
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

func (s *IAMRole) checkChanges(a, e, changes *IAMRole) error {
	if a != nil {
		if e.Name == nil {
			return fi.MissingValueError("Name is required when creating IAMRole")
		}
	}
	return nil
}

func (_*IAMRole) RenderAWS(t *fi.AWSAPITarget, a, e, changes *IAMRole) error {
	if a == nil {
		glog.V(2).Infof("Creating IAMRole with Name:%q", *e.Name)

		policy, err := fi.ResourceAsString(e.RolePolicyDocument)
		if err != nil {
			return fmt.Errorf("error rendering PolicyDocument: %v", err)
		}

		request := &iam.CreateRoleInput{}
		request.AssumeRolePolicyDocument = aws.String(policy)
		request.RoleName = e.Name

		response, err := t.Cloud.IAM.CreateRole(request)
		if err != nil {
			return fmt.Errorf("error creating IAMRole: %v", err)
		}

		e.ID = response.Role.RoleId
	}

	return nil //return output.AddAWSTags(cloud.Tags(), v, "vpc")
}

func (_*IAMRole) RenderBash(t *fi.BashTarget, a, e, changes *IAMRole) error {
	t.CreateVar(e)
	if a == nil {
		glog.V(2).Infof("Creating IAMRole with Name:%q", *e.Name)

		rolePolicyDocument, err := t.AddLocalResource(e.RolePolicyDocument)
		if err != nil {
			return err
		}

		t.AddIAMCommand("create-role",
			"--role-name", *e.Name,
			"--assume-role-policy-document", rolePolicyDocument)
	} else {
		t.AddAssignment(e, *e.ID)
	}

	return nil
}
