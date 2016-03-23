package awsunits

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
)

type Subnet struct {
	fi.SimpleUnit

	Name             *string
	ID               *string
	VPC              *VPC
	AvailabilityZone *string
	CIDR             *string
}

func (s *Subnet) Key() string {
	return *s.Name
}

func (s *Subnet) GetID() *string {
	return s.ID
}

func (e *Subnet) find(c *fi.RunContext) (*Subnet, error) {
	cloud := c.Cloud().(*fi.AWSCloud)

	request := &ec2.DescribeSubnetsInput{}
	if e.ID != nil {
		request.SubnetIds = []*string{e.ID }
	} else {
		request.Filters = cloud.BuildFilters(e.Name)
	}

	response, err := cloud.EC2.DescribeSubnets(request)
	if err != nil {
		return nil, fmt.Errorf("error listing Subnets: %v", err)
	}
	if response == nil || len(response.Subnets) == 0 {
		return nil, nil
	}

	if len(response.Subnets) != 1 {
		glog.Fatalf("found multiple Subnets matching tags")
	}
	subnet := response.Subnets[0]

	actual := &Subnet{}
	actual.ID = subnet.SubnetId
	actual.AvailabilityZone = subnet.AvailabilityZone
	actual.VPC = &VPC{ID: subnet.VpcId}
	actual.CIDR = subnet.CidrBlock

	glog.V(2).Infof("found matching subnet %q", *actual.ID)

	return actual, nil
}

func (e *Subnet) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	if a != nil && e.ID == nil {
		e.ID = a.ID
	}

	changes := &Subnet{}
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

func (s *Subnet) checkChanges(a, e, changes *Subnet) error {
	if a != nil {
		if changes.VPC != nil {
			// TODO: Do we want to destroy & recreate the CIDR?
			return InvalidChangeError("Cannot change subnet VPC", changes.VPC.ID, e.VPC.ID)
		}
		if changes.AvailabilityZone != nil {
			// TODO: Do we want to destroy & recreate the CIDR?
			return InvalidChangeError("Cannot change subnet AvailabilityZone", changes.AvailabilityZone, e.AvailabilityZone)
		}
		if changes.CIDR != nil {
			// TODO: Do we want to destroy & recreate the CIDR?
			return InvalidChangeError("Cannot change subnet CIDR", changes.CIDR, e.CIDR)
		}
	}
	return nil
}

func (_*Subnet) RenderAWS(t *fi.AWSAPITarget, a, e, changes *Subnet) error {
	if a == nil {
		if e.CIDR == nil {
			// TODO: Auto-assign CIDR
			return MissingValueError("Must specify CIDR for Subnet create")
		}

		glog.V(2).Infof("Creating Subnet with CIDR: %q", *e.CIDR)

		var vpcID *string
		if e.VPC != nil {
			vpcID = e.VPC.ID
		}

		request := &ec2.CreateSubnetInput{}
		request.CidrBlock = e.CIDR
		request.AvailabilityZone = e.AvailabilityZone
		request.VpcId = vpcID

		response, err := t.Cloud.EC2.CreateSubnet(request)
		if err != nil {
			return fmt.Errorf("error creating subnet: %v", err)
		}

		subnet := response.Subnet
		e.ID = subnet.SubnetId
	}

	return t.AddAWSTags(*e.ID, t.Cloud.BuildTags(e.Name))
}

func (_*Subnet) RenderBash(t *fi.BashTarget, a, e, changes *Subnet) error {
	t.CreateVar(e)
	if a == nil {
		if e.CIDR == nil {
			// TODO: Auto-assign CIDR
			return MissingValueError("Must specify CIDR for Subnet create")
		}

		vpcID := t.ReadVar(e.VPC)

		args := []string{"create-subnet", "--cidr-block", *e.CIDR, "--vpc-id", vpcID, "--query", "Subnet.SubnetId"}
		if e.AvailabilityZone != nil {
			args = append(args, "--availability-zone", *e.AvailabilityZone)
		}

		t.AddEC2Command(args...).AssignTo(e)
	} else {
		t.AddAssignment(e, StringValue(a.ID))
	}

	return t.AddAWSTags(e, t.Cloud.BuildTags(e.Name))
}