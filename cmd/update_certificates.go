package cmd

//
//import (
//	"fmt"
//
//	"github.com/spf13/cobra"
//	"github.com/golang/glog"
//	"github.com/kopeio/kope/pkg/tasks"
//	"github.com/kopeio/kope/pkg/fi"
//	"github.com/kopeio/kope/pkg/fi/filestore"
//	"github.com/kopeio/kope/pkg/fi/ca"
//	"path"
//	"os"
//	"io/ioutil"
//	"golang.org/x/crypto/ssh"
//	"github.com/kopeio/kope/pkg/kutil"
//)
//
//type UpdateCertificatesCmd struct {
//	Master      string
//	StateDir    string
//	SSHIdentity string
//}
//
//var updateCertificates UpdateCertificatesCmd
//
//func init() {
//	cmd := &cobra.Command{
//		Use:   "certificates",
//		Short: "Update certificates",
//		Long: `Pushes certificates to a running k8s cluster.`,
//		Run: func(cmd *cobra.Command, args[]string) {
//			err := updateCertificates.Run()
//			if err != nil {
//				glog.Exitf("%v", err)
//			}
//		},
//	}
//
//	createCmd.AddCommand(cmd)
//
//	cmd.Flags().StringVarP(&updateCertificates.Master, "master", "m", "", "Master IP address or hostname")
//	cmd.Flags().StringVarP(&updateCertificates.SSHIdentity, "i", "i", "", "SSH private key")
//	cmd.Flags().StringVarP(&updateCertificates.StateDir, "dir", "d", "", "Destination directory")
//}
//
//func (c*UpdateCertificatesCmd) Run() error {
//	if c.Master == "" {
//		return fmt.Errorf("--master must be specified")
//	}
//	fmt.Printf("Connecting to master on %s\n", c.Master)
//
//	master := &kutil.NodeSSH{
//		IP: c.Master,
//	}
//	err := master.AddSSHIdentity(c.SSHIdentity)
//	if err != nil {
//		return err
//	}
//
//
//	macs, err := master.GetMetadataList("network/interfaces/macs/")
//	if err != nil {
//		return fmt.Errorf("cannot determine master network interfaces: %v", err)
//	}
//	if len(macs) == 0 {
//		return fmt.Errorf("master did not have any network interfaces")
//	}
//	subnetID, err := master.GetMetadata("network/interfaces/macs/" + macs[0] + "/subnet-id")
//	if err != nil {
//		return fmt.Errorf("cannot determine master Subnet: %v", err)
//	}
//	k8s.SubnetID = &subnetID
//
//	vpcID, err := master.GetMetadata("network/interfaces/macs/" + macs[0] + "/vpc-id")
//	if err != nil {
//		return fmt.Errorf("cannot determine master VPC: %v", err)
//	}
//	k8s.VPCID = &vpcID
//
//
//
//	// We want to upgrade!
//	// k8s.ImageId = ""
//
//	k8s.ClusterIPRange = conf.Settings["CLUSTER_IP_RANGE"]
//	k8s.AllocateNodeCIDRs = parseBool(conf.Settings["ALLOCATE_NODE_CIDRS"])
//	k8s.Zone = conf.Settings["ZONE"]
//	k8s.KubeUser = conf.Settings["KUBE_USER"]
//	k8s.KubePassword = conf.Settings["KUBE_PASSWORD"]
//	k8s.ServiceClusterIPRange = conf.Settings["SERVICE_CLUSTER_IP_RANGE"]
//	k8s.EnableClusterMonitoring = conf.Settings["ENABLE_CLUSTER_MONITORING"]
//	k8s.EnableClusterLogging = parseBool(conf.Settings["ENABLE_CLUSTER_LOGGING"])
//	k8s.EnableNodeLogging = parseBool(conf.Settings["ENABLE_NODE_LOGGING"])
//	k8s.LoggingDestination = conf.Settings["LOGGING_DESTINATION"]
//	k8s.ElasticsearchLoggingReplicas, err = parseInt(conf.Settings["ELASTICSEARCH_LOGGING_REPLICAS"])
//	if err != nil {
//		return fmt.Errorf("cannot parse ELASTICSEARCH_LOGGING_REPLICAS=%q: %v", conf.Settings["ELASTICSEARCH_LOGGING_REPLICAS"], err)
//	}
//	k8s.EnableClusterDNS = parseBool(conf.Settings["ENABLE_CLUSTER_DNS"])
//	k8s.EnableClusterUI = parseBool(conf.Settings["ENABLE_CLUSTER_UI"])
//	k8s.DNSReplicas, err = parseInt(conf.Settings["DNS_REPLICAS"])
//	if err != nil {
//		return fmt.Errorf("cannot parse DNS_REPLICAS=%q: %v", conf.Settings["DNS_REPLICAS"], err)
//	}
//	k8s.DNSServerIP = conf.Settings["DNS_SERVER_IP"]
//	k8s.DNSDomain = conf.Settings["DNS_DOMAIN"]
//	k8s.AdmissionControl = conf.Settings["ADMISSION_CONTROL"]
//	k8s.MasterIPRange = conf.Settings["MASTER_IP_RANGE"]
//	k8s.DNSServerIP = conf.Settings["DNS_SERVER_IP"]
//	k8s.KubeletToken = conf.Settings["KUBELET_TOKEN"]
//	k8s.KubeProxyToken = conf.Settings["KUBE_PROXY_TOKEN"]
//	k8s.DockerStorage = conf.Settings["DOCKER_STORAGE"]
//	//k8s.MasterExtraSans = conf.Settings["MASTER_EXTRA_SANS"] // Not user set
//	k8s.NodeCount, err = parseInt(conf.Settings["NUM_MINIONS"])
//	if err != nil {
//		return fmt.Errorf("cannot parse NUM_MINIONS=%q: %v", conf.Settings["NUM_MINIONS"], err)
//	}
//
//	if conf.Version == "1.1" {
//		// If users went with defaults on some things, clear them out so they get the new defaults
//		if k8s.AdmissionControl == "NamespaceLifecycle,LimitRanger,SecurityContextDeny,ServiceAccount,ResourceQuota" {
//			// More admission controllers in 1.2
//			k8s.AdmissionControl = ""
//		}
//		if k8s.MasterInstanceType == "t2.micro" {
//			// Different defaults in 1.2
//			k8s.MasterInstanceType = ""
//		}
//		if k8s.NodeInstanceType == "t2.micro" {
//			// Encourage users to pick something better...
//			k8s.NodeInstanceType = ""
//		}
//	}
//
//	az := k8s.Zone
//	if len(az) <= 2 {
//		return fmt.Errorf("Invalid AZ: ", az)
//	}
//	region := az[:len(az) - 1]
//	tags := map[string]string{"KubernetesCluster": k8s.ClusterID}
//	cloud := fi.NewAWSCloud(region, tags)
//
//	igw, err := findInternetGateway(cloud, *k8s.VPCID)
//	if err != nil {
//		return err
//	}
//	if igw == nil {
//		return fmt.Errorf("unable to find internet gateway for VPC %q", k8s.VPCID)
//	}
//	k8s.InternetGatewayID = igw.InternetGatewayId
//
//	rt, err := findRouteTable(cloud, *k8s.SubnetID)
//	if err != nil {
//		return err
//	}
//	if rt == nil {
//		return fmt.Errorf("unable to find route table for Subnet %q", k8s.SubnetID)
//	}
//	k8s.RouteTableID = rt.RouteTableId
//
//
//	//b.Context = "aws_" + instancePrefix
//
//	caCertPath := path.Join(c.DestDir, "pki/ca.crt")
//	err = downloadFile(master, "/srv/kubernetes/ca.crt", caCertPath)
//	if err != nil {
//		return err
//	}
//
//	kubecfgCertPath := path.Join(c.DestDir, "pki/issued/cn=kubernetes-master.crt")
//	err = downloadFile(master, "/srv/kubernetes/kubecfg.crt", kubecfgCertPath)
//	if err != nil {
//		return err
//	}
//	kubecfgKeyPath := path.Join(c.DestDir, "pki/private/cn=kubernetes-master.key")
//	err = downloadFile(master, "/srv/kubernetes/kubecfg.key", kubecfgKeyPath)
//	if err != nil {
//		return err
//	}
//
//	kubeletCertPath := path.Join(c.DestDir, "pki/issued/cn=kubelet.crt")
//	err = downloadFile(node, "/var/run/kubernetes/kubelet.crt", kubeletCertPath)
//	if err != nil {
//		return err
//	}
//	kubeletKeyPath := path.Join(c.DestDir, "pki/private/cn=kubelet.key")
//	err = downloadFile(node, "/var/run/kubernetes/kubelet.key", kubeletKeyPath)
//	if err != nil {
//		return err
//	}
//
//	confPath := path.Join(c.DestDir, "kubernetes.yaml")
//	err = writeConf(confPath, k8s)
//	if err != nil {
//		return err
//	}
//
//	return nil
//}
