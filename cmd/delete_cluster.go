package cmd

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
	"github.com/kopeio/kope/pkg/kutil"
	"github.com/spf13/cobra"
	"time"
)

type DeleteClusterCmd struct {
	ClusterID string
	Yes       bool
	Zone      string
}

var deleteCluster DeleteClusterCmd

func init() {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Delete cluster",
		Long:  `Deletes a k8s cluster.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := deleteCluster.Run()
			if err != nil {
				glog.Exitf("%v", err)
			}
		},
	}

	deleteCmd.AddCommand(cmd)

	cmd.Flags().BoolVar(&deleteCluster.Yes, "yes", false, "Delete without confirmation")

	cmd.Flags().StringVar(&deleteCluster.ClusterID, "cluster-id", "", "cluster id")
	cmd.Flags().StringVar(&deleteCluster.Zone, "zone", "", "zone")
}

func (c *DeleteClusterCmd) Run() error {
	if c.Zone == "" {
		return fmt.Errorf("--zone is required")
	}
	if c.ClusterID == "" {
		return fmt.Errorf("--cluster-id is required")
	}

	az := c.Zone
	if len(az) <= 2 {
		return fmt.Errorf("Invalid AZ: ", az)
	}
	region := az[:len(az)-1]

	tags := map[string]string{"KubernetesCluster": c.ClusterID}
	cloud := fi.NewAWSCloud(region, tags)

	d := &kutil.DeleteCluster{}

	d.ClusterID = c.ClusterID
	d.Zone = c.Zone
	d.Cloud = cloud

	glog.Infof("TODO: S3 bucket removal")

	resources, err := d.ListResources()
	if err != nil {
		return err
	}

	for _, r := range resources {
		fmt.Printf("%v\n", r)
	}

	if !c.Yes {
		return fmt.Errorf("Must specify --yes to delete")
	}

	for {
		// TODO: Parallel delete
		// TODO: Some form of ordering?
		// TODO: Give up eventually?

		var failed []kutil.DeletableResource
		for _, r := range resources {
			fmt.Printf("Deleting resource %s\n", r)
			err := r.Delete(cloud)
			if err != nil {
				fmt.Printf("error deleting resource %s, will retry: %v\n", r, err)
				failed = append(failed, r)
			}
		}

		resources = failed
		if len(resources) == 0 {
			break
		}
		time.Sleep(10 * time.Second)
	}

	return nil
}
