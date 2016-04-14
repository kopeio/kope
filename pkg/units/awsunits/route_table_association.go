package awsunits

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
)

type RouteTableAssociation struct {
	fi.SimpleUnit

	ID         *string
	RouteTable *RouteTable
	Subnet     *Subnet
}

func (s *RouteTableAssociation) Key() string {
	return s.RouteTable.Key() + "-" + s.Subnet.Key()
}

func (s *RouteTableAssociation) Prefix() string {
	return "RouteTableAssociation"
}

func (s *RouteTableAssociation) GetID() *string {
	return s.ID
}

func (e *RouteTableAssociation) find(c *fi.RunContext) (*RouteTableAssociation, error) {
	cloud := c.Cloud().(*fi.AWSCloud)

	routeTableID := e.RouteTable.ID
	subnetID := e.Subnet.ID

	if routeTableID == nil || subnetID == nil {
		return nil, nil
	}

	request := &ec2.DescribeRouteTablesInput{
		RouteTableIds: []*string{routeTableID},
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
	for _, rta := range rt.Associations {
		if aws.StringValue(rta.SubnetId) != *subnetID {
			continue
		}
		actual := &RouteTableAssociation{}
		actual.ID = rta.RouteTableAssociationId
		actual.RouteTable = &RouteTable{ID: rta.RouteTableId}
		actual.Subnet = &Subnet{ID: rta.SubnetId}
		glog.V(2).Infof("found matching RouteTableAssociation %q", *actual.ID)
		return actual, nil
	}

	return nil, nil
}

func (e *RouteTableAssociation) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	if a != nil && e.ID == nil {
		e.ID = a.ID
	}

	changes := &RouteTableAssociation{}
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

func (s *RouteTableAssociation) checkChanges(a, e, changes *RouteTableAssociation) error {
	if a != nil {
		if changes.RouteTable != nil {
			return fi.InvalidChangeError("Cannot change RouteTableAssociation RouteTable", changes.RouteTable.ID, e.RouteTable.ID)
		}
		if changes.Subnet != nil {
			return fi.InvalidChangeError("Cannot change RouteTableAssociation Subnet", changes.Subnet.ID, e.Subnet.ID)
		}
	}
	return nil
}

func (_ *RouteTableAssociation) RenderAWS(t *fi.AWSAPITarget, a, e, changes *RouteTableAssociation) error {
	if a == nil {
		subnetID := e.Subnet.ID
		if subnetID == nil {
			return fi.MissingValueError("Must specify Subnet for RouteTableAssociation create")
		}

		routeTableID := e.RouteTable.ID
		if routeTableID == nil {
			return fi.MissingValueError("Must specify RouteTable for RouteTableAssociation create")
		}

		glog.V(2).Infof("Creating RouteTableAssociation with RouteTable:%q Subnet:%q", *routeTableID, *subnetID)

		request := &ec2.AssociateRouteTableInput{}
		request.SubnetId = subnetID
		request.RouteTableId = routeTableID

		response, err := t.Cloud.EC2.AssociateRouteTable(request)
		if err != nil {
			return fmt.Errorf("error creating RouteTableAssociation: %v", err)
		}

		e.ID = response.AssociationId
	}

	return nil // no tags
}

func (_ *RouteTableAssociation) RenderBash(t *fi.BashTarget, a, e, changes *RouteTableAssociation) error {
	t.CreateVar(e)
	if a == nil {
		subnetID := t.ReadVar(e.Subnet)
		routeTableID := t.ReadVar(e.RouteTable)

		glog.V(2).Infof("Creating RouteTableAssociation with RouteTable:%q Subnet:%q", routeTableID, subnetID)

		t.AddEC2Command("associate-route-table", "--route-table-id", routeTableID, "--subnet-id", subnetID)
	} else {
		t.AddAssignment(e, fi.StringValue(a.ID))
	}

	return nil // no tags
}
