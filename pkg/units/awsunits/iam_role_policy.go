package awsunits

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/golang/glog"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/kopeio/kope/pkg/fi"
)

type IAMRolePolicy struct {
	fi.SimpleUnit

	ID             *string
	Name           *string
	Role           *IAMRole
	PolicyDocument fi.Resource
}

func (s *IAMRolePolicy) Key() string {
	return *s.Name
}

func (s *IAMRolePolicy) GetID() *string {
	return s.ID
}

func (e *IAMRolePolicy) find(c *fi.RunContext) (*IAMRolePolicy, error) {
	cloud := c.Cloud().(*fi.AWSCloud)

	request := &iam.GetRolePolicyInput{
		RoleName:   e.Role.Name,
		PolicyName: e.Name,
	}

	response, err := cloud.IAM.GetRolePolicy(request)
	if err != nil {
		return nil, fmt.Errorf("error getting role: %v", err)
	}

	p := response
	actual := &IAMRolePolicy{}
	actual.Role = &IAMRole{Name: p.RoleName}
	if p.PolicyDocument != nil {
		actual.PolicyDocument = fi.NewStringResource(*p.PolicyDocument)
	}
	actual.Name = p.PolicyName
	return actual, nil
}

func (e *IAMRolePolicy) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	changes := &IAMRolePolicy{}
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

func (s *IAMRolePolicy) checkChanges(a, e, changes *IAMRolePolicy) error {
	if a != nil {
		if e.Name == nil {
			return fi.MissingValueError("Name is required when creating IAMRolePolicy")
		}
	}
	return nil
}

func (_*IAMRolePolicy) RenderAWS(t *fi.AWSAPITarget, a, e, changes *IAMRolePolicy) error {
	if a == nil {
		glog.V(2).Infof("Creating IAMRolePolicy")

		policy, err := fi.ResourceAsString(e.PolicyDocument)
		if err != nil {
			return fmt.Errorf("error rendering PolicyDocument: %v", err)
		}

		request := &iam.PutRolePolicyInput{}
		request.PolicyDocument = aws.String(policy)
		request.RoleName = e.Name
		request.PolicyName = e.Name

		_, err = t.Cloud.IAM.PutRolePolicy(request)
		if err != nil {
			return fmt.Errorf("error creating IAMRolePolicy: %v", err)
		}
	}

	return nil //return output.AddAWSTags(cloud.Tags(), v, "vpc")
}

func (_*IAMRolePolicy) RenderBash(t *fi.BashTarget, a, e, changes *IAMRolePolicy) error {
	t.CreateVar(e)
	if a == nil {
		glog.V(2).Infof("Creating IAMRolePolicy with Name:%q", *e.Name)

		rolePolicyDocument, err := t.AddLocalResource(e.PolicyDocument)
		if err != nil {
			return err
		}

		t.AddIAMCommand("put-role-policy",
			"--role-name", *e.Role.Name,
			"--policy-name", *e.Name,
			"--policy-document", rolePolicyDocument)
	}

	return nil
}

