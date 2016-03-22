package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
	"github.com/kopeio/kope/pkg/kutil"
)

type DiscoverClustersCmd struct {
	Region    string
	ClusterID string
}

var discoverClusters DiscoverClustersCmd

func init() {
	cmd := &cobra.Command{
		Use:   "clusters",
		Short: "Discover clusters",
		Long: `Discover k8s cluster.`,
		Run: func(cmd *cobra.Command, args[]string) {
			if len(args) != 0 {
				if len(args) == 1 {
					discoverClusters.ClusterID = args[0]
				} else {
					glog.Exitf("unexpected arguments passed")
				}
			}
			err := discoverClusters.Run()
			if err != nil {
				glog.Exitf("%v", err)
			}
		},
	}

	discoverCmd.AddCommand(cmd)

	cmd.Flags().StringVar(&discoverClusters.Region, "region", "", "region")
}

func (c*DiscoverClustersCmd) Run() error {
	if c.Region == "" {
		return fmt.Errorf("--region is required")
	}

	tags := make(map[string]string)
	cloud := fi.NewAWSCloud(c.Region, tags)

	var clusterIDs []string

	if c.ClusterID == "" {
		d := &kutil.DiscoverClusters{}

		d.Region = c.Region
		d.Cloud = cloud

		clusters, err := d.ListClusters()
		if err != nil {
			return err
		}

		for _, c := range clusters {
			clusterIDs = append(clusterIDs, c.ClusterID)
		}
	} else {
		clusterIDs = append(clusterIDs, c.ClusterID)
	}

	for _, c := range clusterIDs {
		gi := &kutil.GetClusterInfo{Cloud: cloud, ClusterID: c}
		info, err := gi.GetClusterInfo()
		if err != nil {
			return err
		}
		if info == nil {
			fmt.Printf("%v\t%v\t%v\n", c, "?", "?")
			continue
		}
		fmt.Printf("%v\t%v\t%v\n", info.ClusterID, info.MasterIP, info.Zone)
	}
	return nil
}