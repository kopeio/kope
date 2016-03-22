package kutil

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type DiscoverClusters struct {
	Region string
	Cloud  fi.Cloud
}

type DiscoveredCluster struct {
	ClusterID string
}

func (c*DiscoverClusters)  ListClusters() (map[string]*DiscoveredCluster, error) {
	cloud := c.Cloud.(*fi.AWSCloud)

	clusters := make(map[string]*DiscoveredCluster)

	{

		glog.V(2).Infof("Listing all EC2 tags matching cluster tags")
		var filters []*ec2.Filter
		filters = append(filters, fi.NewEC2Filter("key", "KubernetesCluster"))
		request := &ec2.DescribeTagsInput{
			Filters: filters,
		}
		response, err := cloud.EC2.DescribeTags(request)
		if err != nil {
			return nil, fmt.Errorf("error listing cluster tags: %v", err)
		}

		for _, t := range response.Tags {
			clusterID := *t.Value
			if clusters[clusterID] == nil {
				clusters[clusterID] = &DiscoveredCluster{ClusterID: clusterID}
			}
		}
	}

	return clusters, nil
}
