package gceunits

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
	"github.com/kopeio/kope/pkg/fi/gce"
	"google.golang.org/api/compute/v1"
)

type Network struct {
	fi.SimpleUnit

	Name *string
	CIDR *string

	url string
}

func (s *Network) Key() string {
	return *s.Name
}

func (s *Network) GetID() *string {
	return s.Name
}

func (e *Network) find(c *fi.RunContext) (*Network, error) {
	cloud := c.Cloud().(*gce.GCECloud)

	r, err := cloud.Compute.Networks.Get(cloud.Project, *e.Name).Do()
	if err != nil {
		if gce.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error listing Networks: %v", err)
	}

	actual := &Network{}
	actual.Name = &r.Name
	actual.CIDR = &r.IPv4Range

	e.url = r.SelfLink

	return actual, nil
}

func (e *Network) URL() string {
	// TODO: We might as well just build it using GoogleCloudURL
	if e.url == "" {
		glog.Exitf("URL not set for %q", e.Path)
	}
	return e.url
}

func (e *Network) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	changes := &Network{}
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

func (s *Network) checkChanges(a, e, changes *Network) error {
	if a != nil {
	}
	return nil
}

func (_ *Network) RenderGCE(t *gce.GCEAPITarget, a, e, changes *Network) error {
	if a == nil {
		glog.V(2).Infof("Creating Network with CIDR: %q", *e.CIDR)

		network := &compute.Network{
			IPv4Range: *e.CIDR,

			//// AutoCreateSubnetworks: When set to true, the network is created in
			//// "auto subnet mode". When set to false, the network is in "custom
			//// subnet mode".
			////
			//// In "auto subnet mode", a newly created network is assigned the
			//// default CIDR of 10.128.0.0/9 and it automatically creates one
			//// subnetwork per region.
			//AutoCreateSubnetworks bool `json:"autoCreateSubnetworks,omitempty"`

			Name: *e.Name,
		}
		_, err := t.Cloud.Compute.Networks.Insert(t.Cloud.Project, network).Do()
		if err != nil {
			return fmt.Errorf("error creating Network: %v", err)
		}
	}

	return nil
}

//func (_*Network) RenderBash(t *fi.BashTarget, a, e, changes *Network) error {
//	t.CreateVar(e)
//	if a == nil {
//		vpcID := t.ReadVar(e.VPC)
//
//		//gcloud compute networks create --project "${PROJECT}" "${NETWORK}" --range "10.240.0.0/16"
//
//
//		args := []string{"create-Network", "--cidr-block", *e.CIDR, "--vpc-id", vpcID, "--query", "Network.NetworkId"}
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
