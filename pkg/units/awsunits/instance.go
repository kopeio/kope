package awsunits

import (
	"fmt"

	"encoding/base64"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
)

const MaxUserDataSize = 16384

type Instance struct {
	fi.SimpleUnit

	ID *string
	InstanceCommonConfig
	UserData fi.Resource

	Subnet           *Subnet
	PrivateIPAddress *string

	Name *string
	Tags map[string]string
}

func (s *Instance) Key() string {
	return *s.Name
}

func (s *Instance) GetID() *string {
	return s.ID
}

func (e *Instance) find(c *fi.RunContext) (*Instance, error) {
	cloud := c.Cloud().(*fi.AWSCloud)

	filters := cloud.BuildFilters(e.Name)
	filters = append(filters, fi.NewEC2Filter("instance-state-name", "pending", "running", "stopping", "stopped"))
	request := &ec2.DescribeInstancesInput{
		Filters: filters,
	}

	response, err := cloud.EC2.DescribeInstances(request)
	if err != nil {
		return nil, fmt.Errorf("error listing instances: %v", err)
	}

	instances := []*ec2.Instance{}
	if response != nil {
		for _, reservation := range response.Reservations {
			for _, instance := range reservation.Instances {
				instances = append(instances, instance)
			}
		}
	}

	if len(instances) == 0 {
		return nil, nil
	}

	if len(instances) != 1 {
		return nil, fmt.Errorf("found multiple Instances with name: %s", *e.Name)
	}

	glog.V(2).Info("found existing instance")
	i := instances[0]
	actual := &Instance{}
	actual.ID = i.InstanceId
	actual.PrivateIPAddress = i.PrivateIpAddress
	for _, tag := range i.Tags {
		if aws.StringValue(tag.Key) == "Name" {
			actual.Name = tag.Value
		}
	}
	if i.SubnetId != nil {
		actual.Subnet = &Subnet{ID: i.SubnetId}
	}
	return actual, nil
}

func (e *Instance) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	if a != nil && e.ID == nil {
		e.ID = a.ID
	}

	changes := &Instance{}
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

func (s *Instance) checkChanges(a, e, changes *Instance) error {
	if a != nil {
		if e.Name == nil {
			return fi.MissingValueError("Name is required when creating Instance")
		}
	}
	return nil
}

func (e *Instance) buildTags(cloud *fi.AWSCloud) map[string]string {
	tags := make(map[string]string)
	for k, v := range cloud.BuildTags(e.Name) {
		tags[k] = v
	}
	for k, v := range e.Tags {
		tags[k] = v
	}
	return tags
}

func (_ *Instance) RenderAWS(t *fi.AWSAPITarget, a, e, changes *Instance) error {
	if a == nil {
		glog.V(2).Infof("Creating Instance with Name:%q", *e.Name)

		request := &ec2.RunInstancesInput{}
		request.ImageId = e.ImageID
		request.InstanceType = e.InstanceType
		if e.SSHKey != nil {
			request.KeyName = e.SSHKey.Name
		}
		securityGroupIDs := []*string{}
		for _, sg := range e.SecurityGroups {
			securityGroupIDs = append(securityGroupIDs, sg.ID)
		}
		request.NetworkInterfaces = []*ec2.InstanceNetworkInterfaceSpecification{
			{
				DeviceIndex:              aws.Int64(0),
				AssociatePublicIpAddress: e.AssociatePublicIP,
				SubnetId:                 e.Subnet.ID,
				PrivateIpAddress:         e.PrivateIPAddress,
				Groups:                   securityGroupIDs,
			},
		}

		request.MinCount = aws.Int64(1)
		request.MaxCount = aws.Int64(1)

		if e.BlockDeviceMappings != nil {
			request.BlockDeviceMappings = []*ec2.BlockDeviceMapping{}
			for _, b := range e.BlockDeviceMappings {
				request.BlockDeviceMappings = append(request.BlockDeviceMappings, b.ToEC2())
			}
		}

		if e.UserData != nil {
			d, err := fi.ResourceAsBytes(e.UserData)
			if err != nil {
				return fmt.Errorf("error rendering Instance UserData: %v", err)
			}
			if len(d) > MaxUserDataSize {
				d, err = fi.GzipBytes(d)
				if err != nil {
					return fmt.Errorf("error while gzipping UserData: %v", err)
				}
			}
			request.UserData = aws.String(base64.StdEncoding.EncodeToString(d))
		}
		if e.IAMInstanceProfile != nil {
			request.IamInstanceProfile = &ec2.IamInstanceProfileSpecification{
				Name: e.IAMInstanceProfile.Name,
			}
		}

		response, err := t.Cloud.EC2.RunInstances(request)
		if err != nil {
			return fmt.Errorf("error creating Instance: %v", err)
		}

		e.ID = response.Instances[0].InstanceId
	}

	return t.AddAWSTags(*e.ID, e.buildTags(t.Cloud))
}

func (_ *Instance) RenderBash(t *fi.BashTarget, a, e, changes *Instance) error {
	t.CreateVar(e)
	if a == nil {
		glog.V(2).Infof("Creating Instance with Name:%q", *e.Name)

		args := []string{"run-instances"}
		args = append(args, e.buildEC2CreateArgs(t)...)

		if e.UserData != nil {
			d, err := fi.ResourceAsBytes(e.UserData)
			if err != nil {
				return fmt.Errorf("error rendering Instance UserData: %v", err)
			}
			if len(d) > MaxUserDataSize {
				d, err = fi.GzipBytes(d)
				if err != nil {
					return fmt.Errorf("error while gzipping UserData: %v", err)
				}
			}

			tempFile, err := t.AddLocalResource(fi.NewBytesResource(d))
			if err != nil {
				glog.Fatalf("error adding resource: %v", err)
			}
			args = append(args, "--user-data", "file://"+tempFile)
		}

		if e.Subnet != nil {
			args = append(args, "--subnet-id", t.ReadVar(e.Subnet))
		}
		if e.PrivateIPAddress != nil {
			args = append(args, "--private-ip-address", *e.PrivateIPAddress)
		}

		args = append(args, "--query", "Instances[0].InstanceId")

		t.AddEC2Command(args...).AssignTo(e)
	} else {
		t.AddAssignment(e, aws.StringValue(a.ID))
	}

	return t.AddAWSTags(e, e.buildTags(t.Cloud))
}

/*
func (i *Instance) Destroy(cloud *AWSCloud, output *BashTarget) error {
	existing, err := i.findExisting(cloud)
	if err != nil {
		return err
	}

	if existing != nil {
		glog.V(2).Info("Found instance; will delete: ", i)
		args := []string{"terminate-instances"}
		args = append(args, "--instance-ids", aws.StringValue(existing.InstanceId))

		output.AddEC2Command(args...).AssignTo(i)
	}

	return nil
}
*/
