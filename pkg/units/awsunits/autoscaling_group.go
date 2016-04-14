package awsunits

import (
	"fmt"
	"strconv"

	"encoding/base64"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
	"strings"
	"time"
)

func buildTimestampString() string {
	now := time.Now()
	return now.UTC().Format("20060102T150405Z")
}

// This one is a little weird because we can't update a launch configuration
// So we have to create the launch configuration as part of the group
type AutoscalingGroup struct {
	fi.SimpleUnit

	Name *string

	InstanceCommonConfig
	UserData fi.Resource

	MinSize *int64
	MaxSize *int64
	Subnet  *Subnet
	Tags    map[string]string

	launchConfigurationName *string
}

func (s *AutoscalingGroup) Key() string {
	return *s.Name
}

func (s *AutoscalingGroup) GetID() *string {
	return s.Name
}

func (e *AutoscalingGroup) find(c *fi.RunContext) (*AutoscalingGroup, error) {
	cloud := c.Cloud().(*fi.AWSCloud)

	request := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{e.Name},
	}

	response, err := cloud.Autoscaling.DescribeAutoScalingGroups(request)
	if err != nil {
		return nil, fmt.Errorf("error listing AutoscalingGroups: %v", err)
	}

	if response == nil || len(response.AutoScalingGroups) == 0 {
		return nil, nil
	}

	if len(response.AutoScalingGroups) != 1 {
		glog.Fatalf("found multiple AutoscalingGroups with name: %q", e.Name)
	}

	g := response.AutoScalingGroups[0]
	actual := &AutoscalingGroup{}
	actual.Name = g.AutoScalingGroupName
	actual.MinSize = g.MinSize
	actual.MaxSize = g.MaxSize

	if g.VPCZoneIdentifier != nil {
		subnets := strings.Split(*g.VPCZoneIdentifier, ",")
		if len(subnets) != 1 {
			panic("Multiple subnets not implemented in AutoScalingGroup")
		}
		for _, subnet := range subnets {
			actual.Subnet = &Subnet{ID: aws.String(subnet)}
		}
	}

	if len(g.Tags) != 0 {
		actual.Tags = make(map[string]string)
		for _, tag := range g.Tags {
			actual.Tags[*tag.Key] = *tag.Value
		}
	}

	if g.LaunchConfigurationName == nil {
		return nil, fmt.Errorf("autoscaling Group %q had no LaunchConfiguration", *actual.Name)
	}
	actual.launchConfigurationName = g.LaunchConfigurationName

	found, err := e.findLaunchConfiguration(c, *g.LaunchConfigurationName, actual)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("unable to find autoscaling LaunchConfiguration %q", *g.LaunchConfigurationName)
	}

	return actual, nil
}

func (e *AutoscalingGroup) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	changes := &AutoscalingGroup{}
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

func (s *AutoscalingGroup) checkChanges(a, e, changes *AutoscalingGroup) error {
	if a != nil {
		if e.Name == nil {
			return fi.MissingValueError("Name is required when creating AutoscalingGroup")
		}
	}
	return nil
}

func (e *AutoscalingGroup) buildTags(cloud fi.Cloud) map[string]string {
	tags := make(map[string]string)
	for k, v := range cloud.(*fi.AWSCloud).BuildTags(e.Name) {
		tags[k] = v
	}
	for k, v := range e.Tags {
		tags[k] = v
	}
	return tags
}

func (_ *AutoscalingGroup) RenderAWS(t *fi.AWSAPITarget, a, e, changes *AutoscalingGroup) error {
	if a == nil {
		launchConfigurationName := *e.Name + "-" + buildTimestampString()
		glog.V(2).Infof("Creating autoscaling LaunchConfiguration with Name:%q", launchConfigurationName)

		err := renderAutoscalingLaunchConfigurationAWS(t, launchConfigurationName, e)
		if err != nil {
			return err
		}

		glog.V(2).Infof("Creating autoscaling Group with Name:%q", *e.Name)

		request := &autoscaling.CreateAutoScalingGroupInput{}
		request.AutoScalingGroupName = e.Name
		request.LaunchConfigurationName = &launchConfigurationName
		request.MinSize = e.MinSize
		request.MaxSize = e.MaxSize
		request.VPCZoneIdentifier = e.Subnet.ID

		tags := []*autoscaling.Tag{}
		for k, v := range e.buildTags(t.Cloud) {
			tags = append(tags, &autoscaling.Tag{
				Key:          aws.String(k),
				Value:        aws.String(v),
				ResourceId:   e.Name,
				ResourceType: aws.String("auto-scaling-group"),
			})
		}
		request.Tags = tags

		_, err = t.Cloud.Autoscaling.CreateAutoScalingGroup(request)
		if err != nil {
			return fmt.Errorf("error creating AutoscalingGroup: %v", err)
		}
	} else {
		if changes.UserData != nil {
			launchConfigurationName := *e.Name + "-" + buildTimestampString()
			glog.V(2).Infof("Creating autoscaling LaunchConfiguration with Name:%q", launchConfigurationName)

			err := renderAutoscalingLaunchConfigurationAWS(t, launchConfigurationName, e)
			if err != nil {
				return err
			}

			request := &autoscaling.UpdateAutoScalingGroupInput{
				AutoScalingGroupName:    e.Name,
				LaunchConfigurationName: &launchConfigurationName,
			}
			_, err = t.Cloud.Autoscaling.UpdateAutoScalingGroup(request)
			if err != nil {
				return fmt.Errorf("error updating AutoscalingGroup: %v", err)
			}
		}
	}

	return nil //return output.AddAWSTags(cloud.Tags(), v, "vpc")
}

func (_ *AutoscalingGroup) RenderBash(t *fi.BashTarget, a, e, changes *AutoscalingGroup) error {
	if a == nil {
		launchConfigurationName := *e.Name + "-" + buildTimestampString()
		glog.V(2).Infof("Creating autoscaling LaunchConfiguration with Name:%q", launchConfigurationName)

		err := renderAutoscalingLaunchConfigurationBash(t, launchConfigurationName, e)
		if err != nil {
			return err
		}

		glog.V(2).Infof("Creating autoscaling Group with Name:%q", *e.Name)

		args := []string{"create-auto-scaling-group"}
		args = append(args, "--auto-scaling-group-name", *e.Name)
		args = append(args, "--launch-configuration-name", launchConfigurationName)
		args = append(args, "--min-size", strconv.FormatInt(*e.MinSize, 10))
		args = append(args, "--max-size", strconv.FormatInt(*e.MaxSize, 10))
		args = append(args, "--vpc-zone-identifier", t.ReadVar(e.Subnet))

		tags := e.buildTags(t.Cloud)
		if len(tags) != 0 {
			args = append(args, "--tags")
			for k, v := range tags {
				args = append(args, fmt.Sprintf("ResourceId=%s,ResourceType=auto-scaling-group,Key=%s,Value=%s", *e.Name, k, v))
			}
		}

		t.AddAutoscalingCommand(args...)
	} else {
		if changes.UserData != nil {
			//ad, _ := fi.ResourceAsString(a.UserData)
			//glog.Infof("ACTUAL %s", ad)
			//ed, _ := fi.ResourceAsString(e.UserData)
			//glog.Infof("EXPECTED %s", ed)
			//
			//for {
			//	if ad[0] != ed[0] {
			//		break
			//	}
			//		ad = ad[1:]
			//		ed = ed[1:]
			//}
			//glog.Infof("ACTUAL DELTA %s", ad)
			//glog.Infof("EXPECTED DELTA %s", ed)

			launchConfigurationName := *e.Name + "-" + buildTimestampString()
			glog.V(2).Infof("Creating autoscaling LaunchConfiguration with Name:%q", launchConfigurationName)

			err := renderAutoscalingLaunchConfigurationBash(t, launchConfigurationName, e)
			if err != nil {
				return err
			}

			args := []string{"update-auto-scaling-group"}
			args = append(args, "--auto-scaling-group-name", *e.Name)
			args = append(args, "--launch-configuration-name", launchConfigurationName)
			t.AddAutoscalingCommand(args...)
		}
	}

	return nil
}

/*
func (g *AutoscalingGroup) Destroy(cloud *AWSCloud, output *BashTarget) error {
	existing, err := g.findExisting(cloud)
	if err != nil {
		return err
	}

	if existing != nil {
		glog.V(2).Info("found autoscaling group; will delete: ", g)
		args := []string{"delete-auto-scaling-group"}
		args = append(args, "--auto-scaling-group-name", g.Name)
		args = append(args, "--force-delete")

		output.AddAutoscalingCommand(args...)
	}

	return nil
}
*/

func (e *AutoscalingGroup) findLaunchConfiguration(c *fi.RunContext, name string, dest *AutoscalingGroup) (bool, error) {
	cloud := c.Cloud().(*fi.AWSCloud)

	request := &autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: []*string{&name},
	}

	response, err := cloud.Autoscaling.DescribeLaunchConfigurations(request)
	if err != nil {
		return false, fmt.Errorf("error listing AutoscalingLaunchConfigurations: %v", err)
	}

	if response == nil || len(response.LaunchConfigurations) == 0 {
		return false, nil
	}

	if len(response.LaunchConfigurations) != 1 {
		return false, fmt.Errorf("found multiple AutoscalingLaunchConfigurations with name: %q", *e.Name)
	}

	glog.V(2).Info("found existing AutoscalingLaunchConfiguration")
	i := response.LaunchConfigurations[0]
	dest.Name = i.LaunchConfigurationName
	dest.ImageID = i.ImageId
	dest.InstanceType = i.InstanceType
	dest.SSHKey = &SSHKey{Name: i.KeyName}

	securityGroups := []*SecurityGroup{}
	for _, sgID := range i.SecurityGroups {
		securityGroups = append(securityGroups, &SecurityGroup{ID: sgID})
	}
	dest.SecurityGroups = securityGroups
	dest.AssociatePublicIP = i.AssociatePublicIpAddress

	dest.BlockDeviceMappings = []*BlockDeviceMapping{}
	for _, b := range i.BlockDeviceMappings {
		dest.BlockDeviceMappings = append(dest.BlockDeviceMappings, BlockDeviceMappingFromAutoscaling(b))
	}
	userData, err := base64.StdEncoding.DecodeString(*i.UserData)
	if err != nil {
		return false, fmt.Errorf("error decoding UserData: %v", err)
	}
	dest.UserData = fi.NewStringResource(string(userData))
	dest.IAMInstanceProfile = &IAMInstanceProfile{ID: i.IamInstanceProfile}
	dest.AssociatePublicIP = i.AssociatePublicIpAddress

	return true, nil
}

func renderAutoscalingLaunchConfigurationAWS(t *fi.AWSAPITarget, name string, e *AutoscalingGroup) error {
	glog.V(2).Infof("Creating AutoscalingLaunchConfiguration with Name:%q", name)

	request := &autoscaling.CreateLaunchConfigurationInput{}
	request.LaunchConfigurationName = &name
	request.ImageId = e.ImageID
	request.InstanceType = e.InstanceType
	if e.SSHKey != nil {
		request.KeyName = e.SSHKey.Name
	}
	securityGroupIDs := []*string{}
	for _, sg := range e.SecurityGroups {
		securityGroupIDs = append(securityGroupIDs, sg.ID)
	}
	request.SecurityGroups = securityGroupIDs
	request.AssociatePublicIpAddress = e.AssociatePublicIP
	if e.BlockDeviceMappings != nil {
		request.BlockDeviceMappings = []*autoscaling.BlockDeviceMapping{}
		for _, b := range e.BlockDeviceMappings {
			request.BlockDeviceMappings = append(request.BlockDeviceMappings, b.ToAutoscaling())
		}
	}

	if e.UserData != nil {
		d, err := fi.ResourceAsBytes(e.UserData)
		if err != nil {
			return fmt.Errorf("error rendering AutoScalingLaunchConfiguration UserData: %v", err)
		}
		request.UserData = aws.String(base64.StdEncoding.EncodeToString(d))
	}
	if e.IAMInstanceProfile != nil {
		request.IamInstanceProfile = e.IAMInstanceProfile.Name
	}

	_, err := t.Cloud.Autoscaling.CreateLaunchConfiguration(request)
	if err != nil {
		return fmt.Errorf("error creating AutoscalingLaunchConfiguration: %v", err)
	}

	return nil //return output.AddAWSTags(cloud.Tags(), v, "vpc")
}

func renderAutoscalingLaunchConfigurationBash(t *fi.BashTarget, name string, e *AutoscalingGroup) error {
	t.CreateVar(e)
	glog.V(2).Infof("Creating AutoscalingLaunchConfiguration with Name:%q", *e.Name)

	args := []string{"create-launch-configuration"}
	args = append(args, "--launch-configuration-name", name)
	args = append(args, e.buildAutoscalingCreateArgs(t)...)

	if e.UserData != nil {
		tempFile, err := t.AddLocalResource(e.UserData)
		if err != nil {
			glog.Fatalf("error adding resource: %v", err)
		}
		args = append(args, "--user-data", "file://"+tempFile)
	}

	t.AddAutoscalingCommand(args...)

	return nil
}
