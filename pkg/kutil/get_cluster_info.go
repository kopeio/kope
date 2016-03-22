package kutil

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/aws"
)

const (
	TagKubernetesClusterID = "KubernetesCluster"
	TagRole = "Role"
)

type GetClusterInfo struct {
	ClusterID string
	Cloud     fi.Cloud
}

type ClusterInfo struct {
	ClusterID string
	MasterIP  string
	Zone      string
}

func (c*GetClusterInfo)  GetClusterInfo() (*ClusterInfo, error) {
	cloud := c.Cloud.(*fi.AWSCloud)

	glog.V(2).Infof("Listing all EC2 instances matching cluster tags")
	var filters []*ec2.Filter
	filters = append(filters, fi.NewEC2Filter("tag:" + TagKubernetesClusterID, c.ClusterID))
	filters = append(filters, fi.NewEC2Filter("tag:" + TagRole, "master", "kubernetes-master"))
	request := &ec2.DescribeInstancesInput{
		Filters: filters,
	}
	response, err := cloud.EC2.DescribeInstances(request)
	if err != nil {
		return nil, fmt.Errorf("error listing cluster instances: %v", err)
	}

	var master *ec2.Instance
	for _, r := range response.Reservations {
		for _, i := range r.Instances {
			if i.PublicIpAddress == nil {
				continue
			}
			master = i
		}
	}
	if master == nil {
		return nil, nil
	}

	clusterInfo := &ClusterInfo{
		ClusterID: c.ClusterID,
		MasterIP: aws.StringValue(master.PublicIpAddress),
		Zone: aws.StringValue(master.Placement.AvailabilityZone),
	}
	return clusterInfo, nil
}
