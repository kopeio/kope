package gceunits

import (
	"fmt"

	"github.com/kopeio/kope/pkg/fi"
	"google.golang.org/api/compute/v1"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi/gce"
)

type IPAddress struct {
	fi.SimpleUnit

	Name    *string
	Address *string

	actual  *IPAddress
}

func (s *IPAddress) Key() string {
	return *s.Name
}

func (s *IPAddress) GetID() *string {
	return s.Name
}

func (e *IPAddress) find(cloud *gce.GCECloud) (*IPAddress, error) {
	r, err := cloud.Compute.Addresses.Get(cloud.Project, cloud.Region, *e.Name).Do()
	if err != nil {
		if gce.IsNotFound(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("error listing IPAddresss: %v", err)
	}

	actual := &IPAddress{}
	actual.Address = &r.Address
	actual.Name = &r.Name

	return actual, nil
}

func (e*IPAddress) FindAddress(cloud fi.Cloud) (*string, error) {
	actual, err := e.find(cloud.(*gce.GCECloud))
	if err != nil {
		// TODO: Race here if the address isn't immediately created?
		return nil, fmt.Errorf("error querying for IPAddress: %v", err)
	}
	return actual.Address, nil
}

func (e *IPAddress) Run(c *fi.RunContext) error {
	a, err := e.find(c.Cloud().(*gce.GCECloud))
	if err != nil {
		return err
	}

	changes := &IPAddress{}
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

func (s *IPAddress) checkChanges(a, e, changes *IPAddress) error {
	if a != nil {
		if changes.Name != nil {
			return fi.CannotChangeField("Name")
		}
		if changes.Address != nil {
			return fi.CannotChangeField("Address")
		}
	}
	return nil
}

func (_*IPAddress) RenderGCE(t *gce.GCEAPITarget, a, e, changes *IPAddress) error {
	addr := &compute.Address{
		Name: *e.Name,
		Address: fi.StringValue(e.Address),
		Region: t.Cloud.Region,
	}

	if a == nil {
		glog.Infof("GCE creating address: %q", addr.Name)

		_, err := t.Cloud.Compute.Addresses.Insert(t.Cloud.Project, t.Cloud.Region, addr).Do()
		if err != nil {
			return fmt.Errorf("error creating IPAddress: %v", err)
		}
	} else {
		return fmt.Errorf("Cannot apply changes to IPAddress: %v", changes)
	}

	return nil
}

//func (_*IPAddress) RenderBash(t *fi.BashTarget, a, e, changes *IPAddress) error {
//	t.CreateVar(e)
//	if a == nil {
//		//if gcloud compute addresses create "$1" \
//		//--project "${PROJECT}" \
//		//--region "${REGION}" -q > /dev/null; then
//		//# successful operation
//		//break
//		//fi
//
//		args := []string{"create-IPAddress", "--cidr-block", *e.CIDR, "--vpc-id", vpcID, "--query", "IPAddress.IPAddressId"}
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