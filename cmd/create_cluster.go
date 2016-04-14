package cmd

import (
	"fmt"

	"bytes"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
	"github.com/kopeio/kope/pkg/fi/gce"
	"github.com/kopeio/kope/pkg/kutil"
	"github.com/kopeio/kope/pkg/units"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

type CreateClusterCmd struct {
	CloudProvider string
	Project       string
	ClusterID     string
	S3Bucket      string
	S3Region      string
	SSHKey        string
	StateDir      string
	ReleaseDir    string
	Target        string
	Zone          string
}

var createCluster CreateClusterCmd

func init() {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Create cluster",
		Long:  `Creates a new k8s cluster.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := createCluster.Run()
			if err != nil {
				glog.Exitf("%v", err)
			}
		},
	}

	createCmd.AddCommand(cmd)

	cmd.Flags().StringVarP(&createCluster.CloudProvider, "cloud", "c", "", "Cloud to use for cluster")
	cmd.Flags().StringVarP(&createCluster.Project, "project", "p", "", "Project to use with cloud (for GCE)")
	cmd.Flags().StringVarP(&createCluster.StateDir, "dir", "d", "", "Directory to load & store state")
	cmd.Flags().StringVarP(&createCluster.ReleaseDir, "release", "r", "", "Directory to load release from")
	cmd.Flags().StringVar(&createCluster.S3Region, "s3-region", "", "Region in which to create the S3 bucket (if it does not exist)")
	cmd.Flags().StringVar(&createCluster.S3Bucket, "s3-bucket", "", "S3 bucket for upload of artifacts")
	cmd.Flags().StringVarP(&createCluster.SSHKey, "i", "i", "", "SSH Key for cluster")
	cmd.Flags().StringVarP(&createCluster.Target, "target", "t", "direct", "Target type.  Suported: direct, bash")
	cmd.Flags().StringVarP(&createCluster.Zone, "zone", "z", "", "Zone")

	cmd.Flags().StringVar(&createCluster.ClusterID, "cluster-id", "", "cluster id")
}

func (c *CreateClusterCmd) Run() error {
	k := &units.K8s{}
	k.Init()

	k.ClusterID = c.ClusterID

	if c.CloudProvider != "" {
		k.CloudProvider = c.CloudProvider
	}

	if c.SSHKey != "" {
		buffer, err := ioutil.ReadFile(c.SSHKey)
		if err != nil {
			return fmt.Errorf("error reading SSH key file %q: %v", c.SSHKey, err)
		}

		privateKey, err := ssh.ParsePrivateKey(buffer)
		if err != nil {
			return fmt.Errorf("error parsing key file %q: %v", c.SSHKey, err)
		}

		publicKey := privateKey.PublicKey()
		authorized := ssh.MarshalAuthorizedKey(publicKey)

		k.SSHPublicKey = fi.NewStringResource(string(authorized))
	}

	if c.StateDir == "" {
		return fmt.Errorf("state dir is required")
	}

	if c.ReleaseDir == "" {
		return fmt.Errorf("release dir is required")
	}

	{
		confFile := path.Join(c.StateDir, "kubernetes.yaml")
		b, err := ioutil.ReadFile(confFile)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("error loading state file %q: %v", confFile, err)
			}
		}
		glog.Infof("Loading state from %q", confFile)
		err = k.MergeState(b)
		if err != nil {
			return fmt.Errorf("error parsing state file %q: %v", confFile, err)
		}
	}

	if c.Zone != "" {
		k.Zone = c.Zone
	}

	if k.CloudProvider == "" {
		return fmt.Errorf("must specify CloudProvider.  Specify with -c")
	}

	if k.SSHPublicKey == nil {
		// TODO: Implement the generation logic
		return fmt.Errorf("ssh key is required (for now!).  Specify with -i")
	}

	glog.V(4).Infof("Configuration is %s", fi.DebugPrint(k))

	if k.ClusterID == "" {
		return fmt.Errorf("ClusterID is required")
	}

	if k.Zone == "" {
		return fmt.Errorf("Zone is required")
	}

	var cloud fi.Cloud
	var filestore fi.FileStore
	if k.CloudProvider == "aws" {
		az := k.Zone
		if len(az) <= 2 {
			return fmt.Errorf("Invalid AZ: %v", az)
		}

		region := az[:len(az)-1]
		if c.S3Region == "" {
			c.S3Region = region
		}

		tags := map[string]string{"KubernetesCluster": k.ClusterID}

		awsCloud := fi.NewAWSCloud(region, tags)
		cloud = awsCloud

		if c.S3Bucket == "" {
			b, err := kutil.GetDefaultS3Bucket(awsCloud)
			if err != nil {
				return err
			}
			glog.Infof("Using default S3 bucket: %s", b)
			c.S3Bucket = b
		}

		s3Bucket, err := awsCloud.S3.EnsureBucket(c.S3Bucket, c.S3Region)
		if err != nil {
			return fmt.Errorf("error creating s3 bucket: %v", err)
		}
		s3Prefix := "devel/" + k.ClusterID + "/"
		filestore = fi.NewS3FileStore(s3Bucket, s3Prefix)

		serverBinaryTar := fi.NewFileResource(path.Join(c.ReleaseDir, "server/kubernetes-server-linux-amd64.tar.gz"))
		k.ServerBinaryTar = fi.NewDownloadableFromResource(filestore, "kubernetes-server-linux-amd64", serverBinaryTar)
		saltTar := fi.NewFileResource(path.Join(c.ReleaseDir, "server/kubernetes-salt.tar.gz"))
		k.SaltTar = fi.NewDownloadableFromResource(filestore, "salt.tar", saltTar)

		k.MasterRoleDocument = fi.NewFileResource(path.Join(c.ReleaseDir, "cluster/aws/templates/iam/kubernetes-master-role.json"))
		k.MasterRolePolicy = fi.NewFileResource(path.Join(c.ReleaseDir, "cluster/aws/templates/iam/kubernetes-master-policy.json"))

		k.NodeRoleDocument = fi.NewFileResource(path.Join(c.ReleaseDir, "cluster/aws/templates/iam/kubernetes-minion-role.json"))
		k.NodeRolePolicy = fi.NewFileResource(path.Join(c.ReleaseDir, "cluster/aws/templates/iam/kubernetes-minion-policy.json"))

		bootstrapScript, err := buildAWSBootstrapScript(c.ReleaseDir)
		if err != nil {
			return err
		}

		k.BootstrapScript = fi.NewStringResource(bootstrapScript)
	} else if k.CloudProvider == "gce" {
		zone := k.Zone

		tokens := strings.Split(zone, "-")
		if len(tokens) <= 2 {
			return fmt.Errorf("Invalid Zone: %v", zone)
		}

		location := tokens[0]
		region := tokens[0] + "-" + tokens[1]

		project := c.Project
		if project == "" {
			return fmt.Errorf("project is required for GCE")
		}
		gceCloud, err := gce.NewGCECloud(region, project)
		if err != nil {
			return err
		}
		cloud = gceCloud

		if c.S3Bucket == "" {
			b, err := kutil.GetDefaultGCSBucket(gceCloud, location)
			if err != nil {
				return err
			}
			glog.Infof("Using default S3 bucket: %s", b)
			c.S3Bucket = b
		}

		s3Bucket, err := gce.EnsureGCSBucket(gceCloud.Storage, c.Project, c.S3Bucket, location)
		if err != nil {
			return fmt.Errorf("error creating s3 bucket: %v", err)
		}
		gcsPrefix := "devel/" + k.ClusterID + "/"
		filestore = gce.NewGCSFileStore(s3Bucket, gcsPrefix)

		serverBinaryTar := fi.NewFileResource(path.Join(c.ReleaseDir, "server/kubernetes-server-linux-amd64.tar.gz"))
		k.ServerBinaryTar = fi.NewDownloadableFromResource(filestore, "kubernetes-server-linux-amd64", serverBinaryTar)
		saltTar := fi.NewFileResource(path.Join(c.ReleaseDir, "server/kubernetes-salt.tar.gz"))
		k.SaltTar = fi.NewDownloadableFromResource(filestore, "salt.tar", saltTar)

		k.BootstrapScript = fi.NewFileResource(path.Join(c.ReleaseDir, "cluster/gce/configure-vm.sh"))
	}

	castore, err := fi.NewCAStore(path.Join(c.StateDir, "pki"))
	if err != nil {
		return fmt.Errorf("error building CA store: %v", err)
	}

	var target fi.Target
	var bashTarget *fi.BashTarget
	var dryRunTarget *fi.DryRunTarget

	switch c.Target {
	case "direct":
		switch k.CloudProvider {
		case "aws":
			target = fi.NewAWSAPITarget(cloud.(*fi.AWSCloud), filestore)
		case "gce":
			target = gce.NewGCEAPITarget(cloud.(*gce.GCECloud), filestore)
		default:
			return fmt.Errorf("direct configuration not supported with CloudProvider:%q", k.CloudProvider)
		}
	case "bash":
		bashTarget, err = fi.NewBashTarget(cloud.(*fi.AWSCloud), filestore, ".")
		if err != nil {
			return err
		}
		target = bashTarget
	case "dryrun":
		dryRunTarget, err = fi.NewDryRunTarget()
		if err != nil {
			return err
		}
		target = dryRunTarget
	default:
		return fmt.Errorf("unsupported target type %q", c.Target)
	}

	context, err := fi.NewContext(cloud, castore)
	if err != nil {
		return fmt.Errorf("error building config: %v", err)
	}

	bc := context.NewBuildContext()
	bc.Add(k)

	runMode := fi.ModeConfigure
	//if validate {
	//	runMode = fi.ModeValidate
	//}

	rc := context.NewRunContext(target, runMode)
	err = rc.Run()
	if err != nil {
		return fmt.Errorf("error running configuration: %v", err)
	}

	if bashTarget != nil {
		err = bashTarget.PrintShellCommands(os.Stdout)
		if err != nil {
			glog.Fatal("error building shell commands: %v", err)
		}
	} else if dryRunTarget != nil {
		err = dryRunTarget.PrintReport(os.Stdout)
		if err != nil {
			glog.Fatal("error printing dry-run report: %v", err)
		}
	} else {
		fmt.Printf("\n\nDone\n")
	}
	return nil
}

func buildAWSBootstrapScript(releaseDir string) (string, error) {
	p := path.Join(releaseDir, "cluster/gce/configure-vm.sh")
	gceConfigure, err := ioutil.ReadFile(p)
	if err != nil {
		return "", fmt.Errorf("error reading script %q: %v", p, err)
	}

	p = path.Join(releaseDir, "cluster/aws/templates/configure-vm-aws.sh")
	awsConfigure, err := ioutil.ReadFile(p)
	if err != nil {
		return "", fmt.Errorf("error reading script %q: %v", p, err)
	}

	p = path.Join(releaseDir, "cluster/aws/templates/format-disks.sh")
	awsFormatDisks, err := ioutil.ReadFile(p)
	if err != nil {
		return "", fmt.Errorf("error reading script %q: %v", p, err)
	}

	var b bytes.Buffer
	insertedAWS := false
	for _, gceLine := range strings.Split(string(gceConfigure), "\n") {
		if strings.HasPrefix(gceLine, "#+AWS_OVERRIDES_HERE") {
			if insertedAWS {
				return "", fmt.Errorf("Multiple AWS insertion point markers found in GCE script")
			}
			b.Write(awsConfigure)
			b.Write(awsFormatDisks)
			insertedAWS = true
		} else {
			b.WriteString(gceLine)
			b.WriteString("\n")
		}
	}

	if !insertedAWS {
		return "", fmt.Errorf("AWS insertion point marker not found in GCE script")
	}

	return b.String(), nil
}
