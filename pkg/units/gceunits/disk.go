package gceunits

import (
	"fmt"

	"github.com/kopeio/kope/pkg/fi"
	"github.com/kopeio/kope/pkg/fi/gce"
	"google.golang.org/api/compute/v1"
	"strings"
)

type PersistentDisk struct {
	fi.SimpleUnit

	Name       *string
	VolumeType *string
	SizeGB     *int64
	Zone       *string
}

func (s *PersistentDisk) Key() string {
	return *s.Name
}

func (s *PersistentDisk) GetID() *string {
	return s.Name
}

func (d *PersistentDisk) String() string {
	return fi.JsonString(d)
}

// Returns the last component of a URL, i.e. anything after the last slash
// If there is no slash, returns the whole string
func lastComponent(s string) string {
	lastSlash := strings.LastIndex(s, "/")
	if lastSlash != -1 {
		s = s[lastSlash+1:]
	}
	return s
}

func (e *PersistentDisk) find(c *fi.RunContext) (*PersistentDisk, error) {
	cloud := c.Cloud().(*gce.GCECloud)

	r, err := cloud.Compute.Disks.Get(cloud.Project, *e.Zone, *e.Name).Do()
	if err != nil {
		if gce.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error listing PersistentDisks: %v", err)
	}

	actual := &PersistentDisk{}
	actual.Name = &r.Name
	actual.VolumeType = fi.String(lastComponent(r.Type))
	actual.Zone = fi.String(lastComponent(r.Zone))
	actual.SizeGB = &r.SizeGb

	return actual, nil
}

func (e *PersistentDisk) URL(project string) string {
	u := &gce.GoogleCloudURL{
		Project: project,
		Zone:    *e.Zone,
		Type:    "disks",
		Name:    *e.Name,
	}
	return u.BuildURL()
}

func (e *PersistentDisk) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	changes := &PersistentDisk{}
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

func (s *PersistentDisk) checkChanges(a, e, changes *PersistentDisk) error {
	if a != nil {
		if changes.SizeGB != nil {
			return fi.CannotChangeField("SizeGB")
		}
		if changes.Zone != nil {
			return fi.CannotChangeField("Zone")
		}
		if changes.VolumeType != nil {
			return fi.CannotChangeField("VolumeType")
		}
	} else {
		if e.Zone == nil {
			return fi.RequiredField("Zone")
		}
	}
	return nil
}

func (_ *PersistentDisk) RenderGCE(t *gce.GCEAPITarget, a, e, changes *PersistentDisk) error {
	typeURL := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/diskTypes/%s",
		t.Cloud.Project,
		*e.Zone,
		*e.VolumeType)

	disk := &compute.Disk{
		Name:   *e.Name,
		SizeGb: *e.SizeGB,
		Type:   typeURL,
	}

	if a == nil {
		_, err := t.Cloud.Compute.Disks.Insert(t.Cloud.Project, *e.Zone, disk).Do()
		if err != nil {
			return fmt.Errorf("error creating PersistentDisk: %v", err)
		}
	} else {
		return fmt.Errorf("Cannot apply changes to PersistentDisk: %v", changes)
	}

	return nil
}

//func (_*PersistentDisk) RenderBash(t *fi.BashTarget, a, e, changes *PersistentDisk) error {
//	t.CreateVar(e)
//	if a == nil {
//		//	# We have to make sure the disk is created before creating the master VM, so
//		//# run this in the foreground.
//		//gcloud compute disks create "${MASTER_NAME}-pd" \
//		//--project "${PROJECT}" \
//		//--zone "${ZONE}" \
//		//--type "${MASTER_DISK_TYPE}" \
//		//--size "${MASTER_DISK_SIZE}"
//
//		args := []string{"create-PersistentDisk", "--cidr-block", *e.CIDR, "--vpc-id", vpcID, "--query", "PersistentDisk.PersistentDiskId"}
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
