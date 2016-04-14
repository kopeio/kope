package awsunits

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
)

type SecurityGroup struct {
	fi.SimpleUnit

	ID          *string
	Name        *string
	Description *string
	VPC         *VPC
}

func (s *SecurityGroup) Key() string {
	return *s.Name
}

func (s *SecurityGroup) GetID() *string {
	return s.ID
}

func (e *SecurityGroup) find(c *fi.RunContext) (*SecurityGroup, error) {
	cloud := c.Cloud().(*fi.AWSCloud)

	var vpcID *string
	if e.VPC != nil {
		vpcID = e.VPC.ID
	}

	if vpcID == nil || e.Name == nil {
		return nil, nil
	}

	filters := cloud.BuildFilters(nil) // TODO: Do we need any filters here - done by group-name
	filters = append(filters, fi.NewEC2Filter("vpc-id", *vpcID))
	filters = append(filters, fi.NewEC2Filter("group-name", *e.Name))

	request := &ec2.DescribeSecurityGroupsInput{
		Filters: filters,
	}

	response, err := cloud.EC2.DescribeSecurityGroups(request)
	if err != nil {
		return nil, fmt.Errorf("error listing SecurityGroups: %v", err)
	}
	if response == nil || len(response.SecurityGroups) == 0 {
		return nil, nil
	}

	if len(response.SecurityGroups) != 1 {
		return nil, fmt.Errorf("found multiple SecurityGroups matching tags")
	}
	sg := response.SecurityGroups[0]
	actual := &SecurityGroup{}
	actual.ID = sg.GroupId
	actual.Name = sg.GroupName
	actual.Description = sg.Description
	actual.VPC = &VPC{ID: sg.VpcId}
	glog.V(2).Infof("found matching SecurityGroup %q", *actual.ID)
	return actual, nil
}

func (e *SecurityGroup) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	if a != nil && e.ID == nil {
		e.ID = a.ID
	}

	changes := &SecurityGroup{}
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

func (s *SecurityGroup) checkChanges(a, e, changes *SecurityGroup) error {
	if a != nil {
		if changes.ID != nil {
			return fi.InvalidChangeError("Cannot change SecurityGroup ID", changes.ID, e.ID)
		}
		if changes.Name != nil {
			return fi.InvalidChangeError("Cannot change SecurityGroup Name", changes.Name, e.Name)
		}
		if changes.VPC != nil {
			return fi.InvalidChangeError("Cannot change SecurityGroup VPC", changes.VPC, e.VPC)
		}
	}
	return nil
}

func (_ *SecurityGroup) RenderAWS(t *fi.AWSAPITarget, a, e, changes *SecurityGroup) error {
	if a == nil {
		vpcID := e.VPC.ID

		glog.V(2).Infof("Creating SecurityGroup with Name:%q VPC:%q", *e.Name, *vpcID)

		request := &ec2.CreateSecurityGroupInput{}
		request.VpcId = vpcID
		request.GroupName = e.Name
		request.Description = e.Description

		response, err := t.Cloud.EC2.CreateSecurityGroup(request)
		if err != nil {
			return fmt.Errorf("error creating SecurityGroup: %v", err)
		}

		e.ID = response.GroupId
	}

	return t.AddAWSTags(*e.ID, t.Cloud.BuildTags(e.Name))
}

func (_ *SecurityGroup) RenderBash(t *fi.BashTarget, a, e, changes *SecurityGroup) error {
	t.CreateVar(e)
	if a == nil {
		glog.V(2).Infof("Creating SecurityGroup with Name:%q", *e.Name)

		t.AddEC2Command("create-security-group", "--group-name", *e.Name,
			"--description", fi.BashQuoteString(*e.Description),
			"--vpc-id", t.ReadVar(e.VPC),
			"--query", "GroupId").AssignTo(e)
	} else {
		t.AddAssignment(e, fi.StringValue(a.ID))
	}

	return t.AddAWSTags(e, t.Cloud.BuildTags(e.Name))
}

func (s *SecurityGroup) AllowFrom(source *SecurityGroup) *SecurityGroupIngress {
	return &SecurityGroupIngress{SecurityGroup: s, SourceGroup: source}
}

func (s *SecurityGroup) AllowTCP(cidr string, fromPort int, toPort int) *SecurityGroupIngress {
	fromPort64 := int64(fromPort)
	toPort64 := int64(toPort)
	protocol := "tcp"
	return &SecurityGroupIngress{
		SecurityGroup: s,
		CIDR:          &cidr,
		Protocol:      &protocol,
		FromPort:      &fromPort64,
		ToPort:        &toPort64,
	}
}
