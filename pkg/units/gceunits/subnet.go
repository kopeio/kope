package gceunits

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
	"github.com/kopeio/kope/pkg/fi/gce"
	"google.golang.org/api/compute/v1"
)

type Subnet struct {
	fi.SimpleUnit

	Name    *string
	Network *Network
	Region  *string
	CIDR    *string
}

func (s *Subnet) Key() string {
	return *s.Name
}

func (s *Subnet) GetID() *string {
	return s.Name
}

func (e *Subnet) find(c *fi.RunContext) (*Subnet, error) {
	cloud := c.Cloud().(*gce.GCECloud)

	s, err := cloud.Compute.Subnetworks.Get(cloud.Project, cloud.Region, *e.Name).Do()
	if err != nil {
		if gce.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error listing Subnets: %v", err)
	}

	actual := &Subnet{}
	actual.Name = &s.Name
	actual.Network = &Network{Name: &s.Network}
	actual.Region = &s.Region
	actual.CIDR = &s.IpCidrRange

	return actual, nil
}

func (e *Subnet) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	changes := &Subnet{}
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

func (s *Subnet) checkChanges(a, e, changes *Subnet) error {
	if a != nil {
	}
	return nil
}

func (_ *Subnet) RenderGCE(t *gce.GCEAPITarget, a, e, changes *Subnet) error {
	if a == nil {
		glog.V(2).Infof("Creating Subnet with CIDR: %q", *e.CIDR)

		subnet := &compute.Subnetwork{
			IpCidrRange: *e.CIDR,
			Name:        *e.Name,
			Network:     *e.Network.Name,
		}
		_, err := t.Cloud.Compute.Subnetworks.Insert(t.Cloud.Project, t.Cloud.Region, subnet).Do()
		if err != nil {
			return fmt.Errorf("error creating Subnet: %v", err)
		}
	}

	return nil
}

//func (_*Subnet) RenderBash(t *fi.BashTarget, a, e, changes *Subnet) error {
//	t.CreateVar(e)
//	if a == nil {
//		vpcID := t.ReadVar(e.VPC)
//
//		//gcloud compute Subnets create --project "${PROJECT}" "${Subnet}" --range "10.240.0.0/16"
//
//
//		args := []string{"create-Subnet", "--cidr-block", *e.CIDR, "--vpc-id", vpcID, "--query", "Subnet.SubnetId"}
//		if e.AvailabilityZone != nil {
//			args = append(args, "--availability-zone", *e.AvailabilityZone)
//		}
//
//		t.AddEC2Command(args...).AssignTo(e)
//	} else {
//		t.AddAssignment(e, StringValue(a.ID))
//	}
//
//	return t.AddAWSTags(e, t.Cloud.BuildTags(e.Name))
//}
