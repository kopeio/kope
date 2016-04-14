package gce

import (
	"fmt"
	"github.com/kopeio/kope/pkg/fi"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/storage/v1"
)

type GCECloud struct {
	Compute *compute.Service
	Storage *storage.Service

	Region  string
	Project string

	//tags    map[string]string
}

var _ fi.Cloud = &GCECloud{}

func (c *GCECloud) ProviderID() fi.ProviderID {
	return fi.ProviderGCE
}

func NewGCECloud(region string, project string) (*GCECloud, error) {
	c := &GCECloud{Region: region, Project: project}

	ctx := context.Background()

	client, err := google.DefaultClient(ctx, compute.ComputeScope)
	if err != nil {
		return nil, fmt.Errorf("error building google API client: %v", err)
	}
	computeService, err := compute.New(client)
	if err != nil {
		return nil, fmt.Errorf("error building compute API client: %v", err)
	}
	c.Compute = computeService

	storageService, err := storage.New(client)
	if err != nil {
		return nil, fmt.Errorf("error building storage API client: %v", err)
	}
	c.Storage = storageService

	//c.Compute = compute.New(nil)
	//config := aws.NewConfig().WithRegion(region)
	//c.EC2 = ec2.New(session.New(), config)
	//c.S3 = NewS3Helper(config)
	//c.IAM = iam.New(session.New(), config)
	//c.ELB = elb.New(session.New(), config)
	//c.Autoscaling = autoscaling.New(session.New(), config)

	//c.tags = tags
	return c, nil
}

//func (c*GCECloud) GetS3(region string) *s3.S3 {
//	return c.S3.GetS3(region)
//}
//
//func NewEC2Filter(name string, values ...string) *ec2.Filter {
//	awsValues := []*string{}
//	for _, value := range values {
//		awsValues = append(awsValues, aws.String(value))
//	}
//	filter := &ec2.Filter{
//		Name:   aws.String(name),
//		Values: awsValues,
//	}
//	return filter
//}
//
//func (c *GCECloud) Tags() map[string]string {
//	// Defensive copy
//	tags := make(map[string]string)
//	for k, v := range c.tags {
//		tags[k] = v
//	}
//	return tags
//}
//
//func (c *GCECloud) GetTags(resourceId string) (map[string]string, error) {
//	tags := map[string]string{}
//
//	request := &ec2.DescribeTagsInput{
//		Filters: []*ec2.Filter{
//			NewEC2Filter("resource-id", resourceId),
//		},
//	}
//
//	response, err := c.EC2.DescribeTags(request)
//	if err != nil {
//		return nil, fmt.Errorf("error listing tags on %v: %v", resourceId, err)
//	}
//
//	for _, tag := range response.Tags {
//		if tag == nil {
//			glog.Warning("unexpected nil tag")
//			continue
//		}
//		tags[aws.StringValue(tag.Key)] = aws.StringValue(tag.Value)
//	}
//
//	return tags, nil
//}
//
//func (c *GCECloud) CreateTags(resourceId string, tags map[string]string) (error) {
//	if len(tags) == 0 {
//		return nil
//	}
//
//	ec2Tags := []*ec2.Tag{}
//	for k, v := range tags {
//		ec2Tags = append(ec2Tags, &ec2.Tag{Key: aws.String(k), Value: aws.String(v)})
//	}
//	request := &ec2.CreateTagsInput{
//		Tags: ec2Tags,
//		Resources: []*string{&resourceId},
//	}
//
//	_, err := c.EC2.CreateTags(request)
//	if err != nil {
//		return fmt.Errorf("error creating tags on %v: %v", resourceId, err)
//	}
//
//	return nil
//}
//
//func (c *GCECloud) BuildTags(name *string) map[string]string {
//	tags := make(map[string]string)
//	if name != nil {
//		tags["Name"] = *name
//	} else {
//		glog.Warningf("Name not set when filtering by name")
//	}
//	for k, v := range c.tags {
//		tags[k] = v
//	}
//	return tags
//}
//
//func (c *GCECloud) BuildFilters(name *string) []*ec2.Filter {
//	filters := []*ec2.Filter{}
//
//	merged := make(map[string]string)
//	if name != nil {
//		merged["Name"] = *name
//	} else {
//		glog.Warningf("Name not set when filtering by name")
//	}
//	for k, v := range c.tags {
//		merged[k] = v
//	}
//
//	for k, v := range merged {
//		filter := NewEC2Filter("tag:" + k, v)
//		filters = append(filters, filter)
//	}
//	return filters
//}
//
//func (c *GCECloud) EnvVars() map[string]string {
//	env := map[string]string{}
//	env["AWS_DEFAULT_REGION"] = c.Region
//	env["AWS_DEFAULT_OUTPUT"] = "text"
//	return env
//}
//
//func (t *GCECloud) DescribeInstance(instanceID string) (*ec2.Instance, error) {
//	glog.V(2).Infof("Calling DescribeInstances for instance %q", instanceID)
//	request := &ec2.DescribeInstancesInput{
//		InstanceIds: []*string{&instanceID},
//	}
//
//	response, err := t.EC2.DescribeInstances(request)
//	if err != nil {
//		return nil, fmt.Errorf("error listing Instances: %v", err)
//	}
//	if response == nil || len(response.Reservations) == 0 {
//		return nil, nil
//	}
//	if len(response.Reservations) != 1 {
//		glog.Fatalf("found multiple Reservations for instance id")
//	}
//
//	reservation := response.Reservations[0]
//	if len(reservation.Instances) == 0 {
//		return nil, nil
//	}
//
//	if len(reservation.Instances) != 1 {
//		return nil, fmt.Errorf("found multiple Instances for instance id")
//	}
//
//	instance := reservation.Instances[0]
//	return instance, nil
//}
//
//func (t *GCECloud) DescribeVPC(vpcID string) (*ec2.Vpc, error) {
//	glog.V(2).Infof("Calilng DescribeVPC for VPC %q", vpcID)
//	request := &ec2.DescribeVpcsInput{
//		VpcIds: []*string{&vpcID},
//	}
//
//	response, err := t.EC2.DescribeVpcs(request)
//	if err != nil {
//		return nil, fmt.Errorf("error listing VPCs: %v", err)
//	}
//	if response == nil || len(response.Vpcs) == 0 {
//		return nil, nil
//	}
//	if len(response.Vpcs) != 1 {
//		return nil, fmt.Errorf("found multiple VPCs for instance id")
//	}
//
//	vpc := response.Vpcs[0]
//	return vpc, nil
//}
//
//
//
