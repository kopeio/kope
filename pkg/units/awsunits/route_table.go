package awsunits

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
)

type RouteTable struct {
	fi.SimpleUnit

	Name *string
	ID   *string
	VPC  *VPC
}

func (s *RouteTable) Key() string {
	return *s.Name
}

func (s *RouteTable) GetID() *string {
	return s.ID
}

func (e *RouteTable) find(c *fi.RunContext) (*RouteTable, error) {
	cloud := c.Cloud().(*fi.AWSCloud)

	request := &ec2.DescribeRouteTablesInput{}
	if e.ID != nil {
		request.RouteTableIds = []*string{e.ID}
	} else {
		request.Filters = cloud.BuildFilters(e.Name)
	}

	response, err := cloud.EC2.DescribeRouteTables(request)
	if err != nil {
		return nil, fmt.Errorf("error listing RouteTables: %v", err)
	}
	if response == nil || len(response.RouteTables) == 0 {
		return nil, nil
	}

	if len(response.RouteTables) != 1 {
		return nil, fmt.Errorf("found multiple RouteTables matching tags")
	}
	rt := response.RouteTables[0]

	actual := &RouteTable{}
	actual.ID = rt.RouteTableId
	actual.VPC = &VPC{ID: rt.VpcId}
	actual.Name = e.Name
	glog.V(2).Infof("found matching RouteTable %q", *actual.ID)

	return actual, nil
}

func (e *RouteTable) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	if a != nil && e.ID == nil {
		e.ID = a.ID
	}

	changes := &RouteTable{}
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

func (s *RouteTable) checkChanges(a, e, changes *RouteTable) error {
	if a != nil {
		if changes.VPC != nil && changes.VPC.ID != nil {
			return fi.InvalidChangeError("Cannot change RouteTable VPC", changes.VPC.ID, e.VPC.ID)
		}
	}
	return nil
}

func (_ *RouteTable) RenderAWS(t *fi.AWSAPITarget, a, e, changes *RouteTable) error {
	if a == nil {
		vpcID := e.VPC.ID
		if vpcID == nil {
			return fi.MissingValueError("Must specify VPC for RouteTable create")
		}

		glog.V(2).Infof("Creating RouteTable with VPC: %q", *vpcID)

		request := &ec2.CreateRouteTableInput{}
		request.VpcId = vpcID

		response, err := t.Cloud.EC2.CreateRouteTable(request)
		if err != nil {
			return fmt.Errorf("error creating RouteTable: %v", err)
		}

		rt := response.RouteTable
		e.ID = rt.RouteTableId
	}

	return t.AddAWSTags(*e.ID, t.Cloud.BuildTags(e.Name))
}

func (_ *RouteTable) RenderBash(t *fi.BashTarget, a, e, changes *RouteTable) error {
	t.CreateVar(e)
	if a == nil {
		vpcID := t.ReadVar(e.VPC)

		glog.V(2).Infof("Creating RouteTable with VPC: %q", vpcID)

		t.AddEC2Command("create-route-table", "--vpc-id", vpcID, "--query", "RouteTable.RouteTableId").AssignTo(e)
	} else {
		t.AddAssignment(e, fi.StringValue(a.ID))
	}

	return t.AddAWSTags(e, t.Cloud.BuildTags(e.Name))
}
