package awsunits

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
	"github.com/aws/aws-sdk-go/aws"
)

type InstanceVolumeAttachment struct {
	fi.SimpleUnit

	Instance *Instance
	Volume   *PersistentVolume
	Device   *string
}

func (s *InstanceVolumeAttachment) Key() string {
	return s.Instance.Key() + "-" + s.Volume.Key()
}

func (e *InstanceVolumeAttachment) find(c *fi.RunContext) (*InstanceVolumeAttachment, error) {
	cloud := c.Cloud().(*fi.AWSCloud)

	instanceID := e.Instance.ID
	volumeID := e.Volume.ID

	if instanceID == nil || volumeID == nil {
		return nil, nil
	}

	instance, err := cloud.DescribeInstance(*instanceID)
	if err != nil {
		return nil, err
	}

	for _, bdm := range instance.BlockDeviceMappings {
		if bdm.Ebs == nil {
			continue
		}
		if aws.StringValue(bdm.Ebs.VolumeId) != *volumeID {
			continue
		}

		actual := &InstanceVolumeAttachment{}
		actual.Instance = &Instance{ID: e.Instance.ID }
		actual.Volume = &PersistentVolume{ID: e.Volume.ID }
		actual.Device = bdm.DeviceName
		glog.V(2).Infof("found matching InstanceVolumeAttachment %q", *actual.Device)
		return actual, nil
	}

	return nil, nil
}

func (e *InstanceVolumeAttachment) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	changes := &InstanceVolumeAttachment{}
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

func (s *InstanceVolumeAttachment) checkChanges(a, e, changes *InstanceVolumeAttachment) error {
	if a != nil {
		if changes.Device != nil {
			// TODO: Support this?
			return InvalidChangeError("Cannot change InstanceVolumeAttachment Device", changes.Device, e.Device)
		}
	}

	if a == nil {
		if e.Device == nil {
			return MissingValueError("Must specify Device for InstanceVolumeAttachment create")
		}
	}
	return nil
}

func (_*InstanceVolumeAttachment) RenderAWS(t *fi.AWSAPITarget,a, e, changes *InstanceVolumeAttachment) error {
	if a == nil {
		err := t.WaitForInstanceRunning(*e.Instance.ID)
		if err != nil {
			return err
		}

		request := &ec2.AttachVolumeInput{}
		request.InstanceId = e.Instance.ID
		request.VolumeId = e.Volume.ID
		request.Device = e.Device

		_, err = t.Cloud.EC2.AttachVolume(request)
		if err != nil {
			return fmt.Errorf("error creating InstanceVolumeAttachment: %v", err)
		}
	}

	return nil // no tags
}

func (_*InstanceVolumeAttachment) RenderBash(t *fi.BashTarget, a, e, changes *InstanceVolumeAttachment) error {
	//t.CreateVar(e)
	if a == nil {
		t.WaitForInstanceRunning(e.Instance)

		instanceID := t.ReadVar(e.Instance)
		volumeID := t.ReadVar(e.Volume)

		t.AddEC2Command("attach-volume", "--volume-id", volumeID, "--instance-id", instanceID, "--device", *e.Device)
	} else {
		//t.AddAssignment(e, StringValue(a.ID))
	}

	return nil // no tags
}
