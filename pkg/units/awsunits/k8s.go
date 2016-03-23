package awsunits

import (
	"github.com/kopeio/kope/pkg/fi"
	"github.com/golang/glog"
	"strconv"
	"net"
	"fmt"
	"encoding/binary"
	"math/big"
	"encoding/base64"
	"gopkg.in/yaml.v2"
	"encoding/json"
)

const (
	DefaultMasterVolumeSize = 20
)

type K8s struct {
	fi.SimpleUnit

	S3Region                      string
	S3BucketName                  string

	CloudProvider                 string
	CloudProviderConfig           string

	ClusterID                     string

	MasterInstanceType            string
	NodeInstanceType              string

	ImageID                       string

	MasterInternalIP              string
	// TODO: Just move to master volume?
	MasterVolume                  string
	MasterVolumeSize              *int
	MasterVolumeType              string
	MasterCIDR                    string
	MasterRoleDocument fi.Resource
	MasterRolePolicy fi.Resource

	NodeRoleDocument fi.Resource
	NodeRolePolicy fi.Resource

	NodeCount                     int

	InstancePrefix                string
	NodeInstancePrefix            string
	ClusterIPRange                string
	MasterIPRange                 string
	AllocateNodeCIDRs             bool

	ServerBinaryTar               fi.Resource
	SaltTar                       fi.Resource
	BootstrapScript               fi.Resource

	Zone                          string
	KubeUser                      string
	KubePassword                  string

	//SaltMaster                    string
	MasterName                    string

	ServiceClusterIPRange         string
	EnableL7LoadBalancing         string
	EnableClusterMonitoring       string
	EnableClusterLogging          bool
	EnableNodeLogging             bool
	LoggingDestination            string
	ElasticsearchLoggingReplicas  int

	EnableClusterRegistry         *bool
	ClusterRegistryDisk           *string
	ClusterRegistryDiskSize       *int

	EnableClusterDNS              bool
	DNSReplicas                   int
	DNSServerIP                   string
	DNSDomain                     string

	RuntimeConfig                 string

	CACert                        fi.Resource
	CAKey                         fi.Resource
	KubeletCert                   fi.Resource
	KubeletKey                    fi.Resource
	KubeletToken                  string
	KubeProxyToken                string
	BearerToken                   string
	MasterCert                    fi.Resource
	MasterKey                     fi.Resource
	KubecfgCert                   fi.Resource
	KubecfgKey                    fi.Resource

	RegisterMasterKubelet         *bool
	//KubeletApiserver              string

	EnableManifestURL             *bool
	ManifestURL                   *string
	ManifestURLHeader             *string

	NetworkProvider               string

	HairpinMode                   string

	OpencontrailTag               string
	OpencontrailKubernetesTag     string
	OpencontrailPublicSubnet      string

	KubeImageTag                  string
	KubeDockerRegistry            string
	KubeAddonRegistry             string

	Multizone                     *bool

	NonMasqueradeCidr             string

	E2EStorageTestEnvironment     string

	EnableClusterUI               bool

	AdmissionControl              string

	KubeletPort                   *int

	KubeApiserverRequestTimeout   *int

	TerminatedPodGcThreshold      *string

	KubeManifestsTarURL           string
	KubeManifestsTarSha256        string

	TestCluster                   string

	DockerOptions                 string
	DockerStorage                 string

	MasterExtraSans               []string

	// TODO: Make struct?
	KubeletTestArgs               string
	KubeletTestLogLevel           string
	DockerTestArgs                string
	DockerTestLogLevel            string
	ApiserverTestArgs             string
	ApiserverTestLogLevel         string
	ControllerManagerTestArgs     string
	ControllerManagerTestLogLevel string
	SchedulerTestArgs             string
	SchedulerTestLogLevel         string
	KubeProxyTestArgs             string
	KubeProxyTestLogLevel         string

	NodeLabels                    string
	OsDistribution                string

	ExtraDockerOpts               string

	ContainerRuntime              string
	RktVersion                    string
	RktPath                       string
	KubernetesConfigureCbr0       string

	EnableCustomMetrics           bool

	SSHPublicKey                  fi.Resource

	// For upgrades
	SubnetID                      *string
	VPCID                         *string
	InternetGatewayID             *string
	RouteTableID                  *string
	DHCPOptionsID                 *string
	MasterElasticIP               *string
}

func (k*K8s) Key() string {
	return k.ClusterID
}

func (k*K8s) BuildEnv(c *fi.RunContext, isMaster bool) (map[string]string, error) {
	// The bootstrap script requires some variables to be set...
	// We use this as a marker for future cleanup
	legacyEmptyVar := ""


	// For now, we add everything as a string...
	// The first problem is that the python parser converts true / false to "True" and "False",
	// which breaks string-based comparisons (as done in bash)
	y := map[string]string{}
	// ENV_TIMESTAMP breaks our deltas!
	//y["ENV_TIMESTAMP"] = time.Now().UTC().Format("2006-01-02T03:04:05+0000")
	y["INSTANCE_PREFIX"] = k.InstancePrefix
	y["NODE_INSTANCE_PREFIX"] = k.NodeInstancePrefix
	y["CLUSTER_IP_RANGE"] = k.ClusterIPRange

	{
		url, hash, err := c.Target.PutResource("server", k.ServerBinaryTar, fi.HashAlgorithmSHA1)
		if err != nil {
			return nil, err
		}
		y["SERVER_BINARY_TAR_URL"] = url
		y["SERVER_BINARY_TAR_HASH"] = hash
	}

	{
		url, hash, err := c.Target.PutResource("salt", k.SaltTar, fi.HashAlgorithmSHA1)
		if err != nil {
			return nil, err
		}
		y["SALT_TAR_URL"] = url
		y["SALT_TAR_HASH"] = hash
	}

	y["SERVICE_CLUSTER_IP_RANGE"] = k.ServiceClusterIPRange

	y["KUBERNETES_MASTER_NAME"] = k.MasterName

	y["ALLOCATE_NODE_CIDRS"] = strconv.FormatBool(k.AllocateNodeCIDRs)

	y["ENABLE_CLUSTER_MONITORING"] = k.EnableClusterMonitoring
	y["ENABLE_L7_LOADBALANCING"] = k.EnableL7LoadBalancing
	y["ENABLE_CLUSTER_LOGGING"] = strconv.FormatBool(k.EnableClusterLogging)
	y["ENABLE_CLUSTER_UI"] = strconv.FormatBool(k.EnableClusterUI)
	y["ENABLE_NODE_LOGGING"] = strconv.FormatBool(k.EnableNodeLogging)
	y["LOGGING_DESTINATION"] = k.LoggingDestination
	y["ELASTICSEARCH_LOGGING_REPLICAS"] = strconv.Itoa(k.ElasticsearchLoggingReplicas)
	y["ENABLE_CLUSTER_DNS"] = strconv.FormatBool(k.EnableClusterDNS)
	if k.EnableClusterRegistry != nil {
		y["ENABLE_CLUSTER_REGISTRY"] = strconv.FormatBool(*k.EnableClusterRegistry)
	} else {
		y["ENABLE_CLUSTER_REGISTRY"] = legacyEmptyVar
	}
	if k.ClusterRegistryDisk != nil {
		y["CLUSTER_REGISTRY_DISK"] = *k.ClusterRegistryDisk
	}
	if k.ClusterRegistryDiskSize != nil {
		y["CLUSTER_REGISTRY_DISK_SIZE"] = strconv.Itoa(*k.ClusterRegistryDiskSize)
	}
	y["DNS_REPLICAS"] = strconv.Itoa(k.DNSReplicas)
	y["DNS_SERVER_IP"] = k.DNSServerIP
	y["DNS_DOMAIN"] = k.DNSDomain

	y["KUBELET_TOKEN"] = k.KubeletToken
	y["KUBE_PROXY_TOKEN"] = k.KubeProxyToken
	y["ADMISSION_CONTROL"] = k.AdmissionControl
	y["MASTER_IP_RANGE"] = k.MasterIPRange
	y["RUNTIME_CONFIG"] = k.RuntimeConfig
	y["CA_CERT"] = ResourceAsBase64String(k.CACert)
	y["KUBELET_CERT"] = ResourceAsBase64String(k.KubeletCert)
	y["KUBELET_KEY"] = ResourceAsBase64String(k.KubeletKey)
	y["NETWORK_PROVIDER"] = k.NetworkProvider
	y["HAIRPIN_MODE"] = k.HairpinMode
	y["OPENCONTRAIL_TAG"] = k.OpencontrailTag
	y["OPENCONTRAIL_KUBERNETES_TAG"] = k.OpencontrailKubernetesTag
	y["OPENCONTRAIL_PUBLIC_SUBNET"] = k.OpencontrailPublicSubnet
	y["E2E_STORAGE_TEST_ENVIRONMENT"] = k.E2EStorageTestEnvironment
	y["KUBE_IMAGE_TAG"] = k.KubeImageTag
	y["KUBE_DOCKER_REGISTRY"] = k.KubeDockerRegistry
	y["KUBE_ADDON_REGISTRY"] = k.KubeAddonRegistry
	if BoolValue(k.Multizone) {
		y["MULTIZONE"] = "1"
	}
	y["NON_MASQUERADE_CIDR"] = k.NonMasqueradeCidr

	if k.KubeletPort != nil {
		y["KUBELET_PORT"] = strconv.Itoa(*k.KubeletPort)
	}

	if k.KubeApiserverRequestTimeout != nil {
		y["KUBE_APISERVER_REQUEST_TIMEOUT"] = strconv.Itoa(*k.KubeApiserverRequestTimeout)
	}

	if k.TerminatedPodGcThreshold != nil {
		y["TERMINATED_POD_GC_THRESHOLD"] = *k.TerminatedPodGcThreshold
	}

	if k.OsDistribution == "trusty" {
		y["KUBE_MANIFESTS_TAR_URL"] = k.KubeManifestsTarURL
		y["KUBE_MANIFESTS_TAR_HASH"] = k.KubeManifestsTarSha256
	}

	if k.TestCluster != "" {
		y["TEST_CLUSTER"] = k.TestCluster
	}

	if k.KubeletTestArgs != "" {
		y["KUBELET_TEST_ARGS"] = k.KubeletTestArgs
	}

	if k.KubeletTestLogLevel != "" {
		y["KUBELET_TEST_LOG_LEVEL"] = k.KubeletTestLogLevel
	}

	if k.DockerTestLogLevel != "" {
		y["DOCKER_TEST_LOG_LEVEL"] = k.DockerTestLogLevel
	}

	if k.EnableCustomMetrics {
		y["ENABLE_CUSTOM_METRICS"] = strconv.FormatBool(k.EnableCustomMetrics)
	}

	if isMaster {
		// If the user requested that the master be part of the cluster, set the
		// environment variable to program the master kubelet to register itself.
		if BoolValue(k.RegisterMasterKubelet) {
			y["KUBELET_APISERVER"] = k.MasterName
		}

		y["KUBERNETES_MASTER"] = strconv.FormatBool(true)
		y["KUBE_USER"] = k.KubeUser
		y["KUBE_PASSWORD"] = k.KubePassword
		y["KUBE_BEARER_TOKEN"] = k.BearerToken
		y["MASTER_CERT"] = ResourceAsBase64String(k.MasterCert)
		y["MASTER_KEY"] = ResourceAsBase64String(k.MasterKey)
		y["KUBECFG_CERT"] = ResourceAsBase64String(k.KubecfgCert)
		y["KUBECFG_KEY"] = ResourceAsBase64String(k.KubecfgKey)

		if k.EnableManifestURL != nil {
			y["ENABLE_MANIFEST_URL"] = strconv.FormatBool(*k.EnableManifestURL)
		} else {
			y["ENABLE_MANIFEST_URL"] = legacyEmptyVar
		}
		if k.ManifestURL != nil {
			y["MANIFEST_URL"] = *k.ManifestURL
		}else {
			y["MANIFEST_URL"] = legacyEmptyVar
		}
		if k.ManifestURLHeader != nil {
			y["MANIFEST_URL_HEADER"] = *k.ManifestURLHeader
		}else {
			y["MANIFEST_URL_HEADER"] = legacyEmptyVar
		}
		y["NUM_NODES"] = strconv.Itoa(k.NodeCount)

		if k.ApiserverTestArgs != "" {
			y["APISERVER_TEST_ARGS"] = k.ApiserverTestArgs
		}

		if k.ApiserverTestLogLevel != "" {
			y["APISERVER_TEST_LOG_LEVEL"] = k.ApiserverTestLogLevel
		}

		if k.ControllerManagerTestArgs != "" {
			y["CONTROLLER_MANAGER_TEST_ARGS"] = k.ControllerManagerTestArgs
		}

		if k.ControllerManagerTestLogLevel != "" {
			y["CONTROLLER_MANAGER_TEST_LOG_LEVEL"] = k.ControllerManagerTestLogLevel
		}

		if k.SchedulerTestArgs != "" {
			y["SCHEDULER_TEST_ARGS"] = k.SchedulerTestArgs
		}

		if k.SchedulerTestLogLevel != "" {
			y["SCHEDULER_TEST_LOG_LEVEL"] = k.SchedulerTestLogLevel
		}

	}

	if !isMaster {
		// Node-only vars

		y["KUBERNETES_MASTER"] = strconv.FormatBool(false)
		y["ZONE"] = k.Zone
		y["EXTRA_DOCKER_OPTS"] = k.ExtraDockerOpts
		if k.ManifestURL != nil {
			y["MANIFEST_URL"] = *k.ManifestURL
		}

		if k.KubeProxyTestArgs != "" {
			y["KUBEPROXY_TEST_ARGS"] = k.KubeProxyTestArgs
		}

		if k.KubeProxyTestLogLevel != "" {
			y["KUBEPROXY_TEST_LOG_LEVEL"] = k.KubeProxyTestLogLevel
		}
	}

	if k.NodeLabels != "" {
		y["NODE_LABELS"] = k.NodeLabels
	}

	if k.OsDistribution == "coreos" {
		// CoreOS-only env vars. TODO(yifan): Make them available on other distros.
		y["KUBE_MANIFESTS_TAR_URL"] = k.KubeManifestsTarURL
		y["KUBE_MANIFESTS_TAR_HASH"] = k.KubeManifestsTarSha256
		y["KUBERNETES_CONTAINER_RUNTIME"] = k.ContainerRuntime
		y["RKT_VERSION"] = k.RktVersion
		y["RKT_PATH"] = k.RktPath
		y["KUBERNETES_CONFIGURE_CBR0"] = k.KubernetesConfigureCbr0

	}


	// This next bit for changes vs kube-up:
	y["CA_KEY"] = ResourceAsBase64String(k.CAKey) // https://github.com/kubernetes/kubernetes/issues/23264

	return y, nil
}

func (k*K8s) Init() {
	k.MasterInstanceType = "m3.medium"
	k.NodeInstanceType = "m3.medium"
	k.MasterInternalIP = "172.20.0.9"
	k.NodeCount = 2
	k.DockerStorage = "aufs"
	k.MasterIPRange = "10.246.0.0/24"
	k.MasterVolumeType = "gp2"
	k.MasterVolumeSize = Int(DefaultMasterVolumeSize)
	k.Zone = "us-east-1b"
	k.EnableClusterUI = true
	k.EnableClusterDNS = true
	k.EnableClusterLogging = true
	k.LoggingDestination = "elasticsearch"
	k.EnableClusterMonitoring = "influxdb" // "none" ?
	k.EnableL7LoadBalancing = "none"
	k.EnableNodeLogging = true
	k.ElasticsearchLoggingReplicas = 1
	k.DNSReplicas = 1
	k.DNSServerIP = "10.0.0.10"
	k.DNSDomain = "cluster.local"
	k.AdmissionControl = "NamespaceLifecycle,LimitRanger,SecurityContextDeny,ServiceAccount,ResourceQuota,PersistentVolumeLabel"
	k.ServiceClusterIPRange = "10.0.0.0/16"
	k.ClusterIPRange = "10.244.0.0/16"

	k.NetworkProvider = "none"

	k.ContainerRuntime = "docker"
	k.KubernetesConfigureCbr0 = "true"

	// Required to work with autoscaling minions
	k.AllocateNodeCIDRs = true

	k.CloudProvider = "aws"
}

func (k*K8s) MergeState(state []byte) error {
	glog.V(4).Infof("Loading yaml: %s", string(state))

	var yamlObj map[string]interface{}
	err := yaml.Unmarshal(state, &yamlObj)
	if err != nil {
		return fmt.Errorf("error loading state (yaml read phase): %v", err)
	}

	jsonBytes, err := json.Marshal(yamlObj)
	if err != nil {
		return fmt.Errorf("error loading state (json write phase): %v", err)
	}

	err = json.Unmarshal(jsonBytes, k)
	if err != nil {
		return fmt.Errorf("error loading state (json read phase): %v", err)
	}

	return nil
}

func (k *K8s) Add(c *fi.BuildContext) {
	clusterID := k.ClusterID
	if clusterID == "" {
		glog.Exit("cluster-id is required")
	}

	if len(k.Zone) <= 2 {
		glog.Exit("Invalid AZ: ", k.Zone)
	}

	if k.ImageID == "" {
		jessie := &DistroJessie{}
		imageID, err := jessie.GetImageID(c.Context)
		if err != nil {
			glog.Exitf("error while trying to find AWS image: %v", err)
		}
		k.ImageID = imageID
	}
	//region := az[:len(az) - 1]

	// Simplifications
	instancePrefix := k.ClusterID
	if k.InstancePrefix == "" {
		k.InstancePrefix = instancePrefix
	}

	if k.NodeInstancePrefix == "" {
		k.NodeInstancePrefix = instancePrefix + "-minion"
	}
	if k.MasterName == "" {
		k.MasterName = instancePrefix + "-master"
	}

	if k.KubeUser == "" {
		k.KubeUser = "admin"
	}
	if k.KubePassword == "" {
		k.KubePassword = RandomToken(16)
	}

	if k.KubeletToken == "" {
		k.KubeletToken = RandomToken(32)
	}

	if k.KubeProxyToken == "" {
		k.KubeProxyToken = RandomToken(32)
	}

	//s3BucketName := k.S3BucketName
	//if k.S3BucketName == "" {
	//	// TODO: Implement the generation logic
	//	glog.Exit("s3-bucket is required (for now!)")
	//}

	//s3Region := k.S3Region
	//if s3Region == "" {
	//	s3Region = region
	//}

	//s3Bucket := &S3Bucket{
	//	Name:         String(s3BucketName),
	//	Region: String(s3Region),
	//}
	//c.Add(s3Bucket)
	//
	//s3KubernetesFile := &S3File{
	//	Bucket: s3Bucket,
	//	Key:    String("devel/kubernetes-server-linux-amd64.tar.gz"),
	//	Source: findKubernetesTarGz(),
	//	Public: Bool(true),
	//}
	//c.Add(s3KubernetesFile)
	//
	//s3SaltFile := &S3File{
	//	Bucket: s3Bucket,
	//	Key:    String("devel/kubernetes-salt.tar.gz"),
	//	Source: findSaltTarGz(),
	//	Public: Bool(true),
	//}
	//c.Add(s3SaltFile)
	//
	//s3BootstrapScriptFile := &S3File{
	//	Bucket: s3Bucket,
	//	Key:    String("devel/bootstrap"),
	//	Source: findBootstrap(),
	//	Public: Bool(true),
	//}
	//c.Add(s3BootstrapScriptFile)
	//
	//glog.Info("Processing S3 resources")
	//
	//k.ServerBinaryTarURL = s3KubernetesFile.PublicURL()
	//k.ServerBinaryTarHash = s3KubernetesFile.Hash()
	//k.SaltTarURL = s3SaltFile.PublicURL()
	//k.SaltTarHash = s3SaltFile.Hash()
	//k.BootstrapScriptURL = s3BootstrapScriptFile.PublicURL()

	masterVolumeSize := DefaultMasterVolumeSize
	if k.MasterVolumeSize != nil {
		masterVolumeSize = *k.MasterVolumeSize
	}
	masterPV := &PersistentVolume{
		AvailabilityZone:         String(k.Zone),
		Size:       Int64(int64(masterVolumeSize)),
		VolumeType: String(k.MasterVolumeType),
		Name:    String(clusterID + "-master-pd"),
	}
	c.Add(masterPV)

	masterIP := &ElasticIP{
		PublicIP: k.MasterElasticIP,
		TagOnResource: masterPV,
		TagUsingKey: String("kubernetes.io/master-ip"),
	}
	c.Add(masterIP)

	c.Add(&CertBuilder{Kubernetes: k, MasterIP: masterIP})

	//glog.Info("Processing master volume resource")
	//masterPVResources := []fi.Unit{
	//	masterPV,
	//}
	//renderItems(context, masterPVResources...)
	//
	//k.MasterVolume = target.ReadVar(masterPV)

	iamMasterRole := &IAMRole{
		Name:               String("kubernetes-master"),
		RolePolicyDocument: k.MasterRoleDocument,
	}
	c.Add(iamMasterRole)

	iamMasterRolePolicy := &IAMRolePolicy{
		Role:           iamMasterRole,
		Name:           String("kubernetes-master"),
		PolicyDocument: k.MasterRolePolicy,
	}
	c.Add(iamMasterRolePolicy)

	iamMasterInstanceProfile := &IAMInstanceProfile{
		Name: String("kubernetes-master"),
	}
	c.Add(iamMasterInstanceProfile)

	iamMasterInstanceProfileRole := &IAMInstanceProfileRole{
		InstanceProfile: iamMasterInstanceProfile,
		Role: iamMasterRole,
	}
	c.Add(iamMasterInstanceProfileRole)

	iamNodeRole := &IAMRole{
		Name:             String("kubernetes-minion"),
		RolePolicyDocument: k.NodeRoleDocument,
	}
	c.Add(iamNodeRole)

	iamNodeRolePolicy := &IAMRolePolicy{
		Role:           iamNodeRole,
		Name:          String("kubernetes-minion"),
		PolicyDocument: k.NodeRolePolicy,
	}
	c.Add(iamNodeRolePolicy)

	iamNodeInstanceProfile := &IAMInstanceProfile{
		Name:String("kubernetes-minion"),
	}
	c.Add(iamNodeInstanceProfile)

	iamNodeInstanceProfileRole := &IAMInstanceProfileRole{
		InstanceProfile: iamNodeInstanceProfile,
		Role: iamNodeRole,
	}
	c.Add(iamNodeInstanceProfileRole)

	sshKey := &SSHKey{Name: String("kubernetes-" + clusterID), PublicKey: k.SSHPublicKey}
	c.Add(sshKey)

	vpc := &VPC{
		ID: k.VPCID,
		CIDR:String("172.20.0.0/16"),
		Name: String("kubernetes-" + clusterID),
		EnableDNSSupport:Bool(true),
		EnableDNSHostnames:Bool(true),
	}
	c.Add(vpc)

	region := c.Cloud().(*fi.AWSCloud).Region
	dhcpDomainName := region + ".compute.internal"
	if region == "us-east-1" {
		dhcpDomainName = "ec2.internal"
	}
	dhcpOptions := &DHCPOptions{
		ID: k.DHCPOptionsID,
		Name: String("kubernetes-" + clusterID),
		DomainName: String(dhcpDomainName),
		DomainNameServers: String("AmazonProvidedDNS"),
	}
	c.Add(dhcpOptions)

	c.Add(&VPCDHCPOptionsAssociation{VPC: vpc, DHCPOptions: dhcpOptions })

	subnet := &Subnet{VPC: vpc, AvailabilityZone: String(k.Zone), CIDR: String("172.20.0.0/24"), Name: String("kubernetes-" + clusterID), ID: k.SubnetID}
	c.Add(subnet)

	igw := &InternetGateway{Name: String("kubernetes-" + clusterID), ID: k.InternetGatewayID}
	c.Add(igw)

	c.Add(&InternetGatewayAttachment{VPC: vpc, InternetGateway: igw})

	routeTable := &RouteTable{VPC: vpc, Name: String("kubernetes-" + clusterID), ID: k.RouteTableID}
	c.Add(routeTable)

	route := &Route{RouteTable: routeTable, CIDR: String("0.0.0.0/0"), InternetGateway: igw}
	c.Add(route)

	c.Add(&RouteTableAssociation{RouteTable: routeTable, Subnet: subnet})

	masterSG := &SecurityGroup{
		Name:        String("kubernetes-master-" + clusterID),
		Description: String("Security group for master nodes"),
		VPC:         vpc}
	c.Add(masterSG)

	nodeSG := &SecurityGroup{
		Name:        String("kubernetes-minion-" + clusterID),
		Description: String("Security group for minion nodes"),
		VPC:         vpc}
	c.Add(nodeSG)

	c.Add(masterSG.AllowFrom(masterSG))
	c.Add(masterSG.AllowFrom(nodeSG))
	c.Add(nodeSG.AllowFrom(masterSG))
	c.Add(nodeSG.AllowFrom(nodeSG))

	// SSH is open to the world
	c.Add(nodeSG.AllowTCP("0.0.0.0/0", 22, 22))
	c.Add(masterSG.AllowTCP("0.0.0.0/0", 22, 22))

	// HTTPS to the master is allowed (for API access)
	c.Add(masterSG.AllowTCP("0.0.0.0/0", 443, 443))

	masterUserData := &MasterScript{
		Config: k,
	}
	c.Add(masterUserData)

	masterBlockDeviceMappings := []*BlockDeviceMapping{}

	// Be sure to map all the ephemeral drives.  We can specify more than we actually have.
	// TODO: Actually mount the correct number (especially if we have more), though this is non-trivial, and
	//  only affects the big storage instance types, which aren't a typical use case right now.
	for i := 0; i < 4; i++ {
		bdm := &BlockDeviceMapping{
			DeviceName:  String("/dev/sd" + string('c' + i)),
			VirtualName: String("ephemeral" + strconv.Itoa(i)),
		}
		masterBlockDeviceMappings = append(masterBlockDeviceMappings, bdm)
	}

	nodeBlockDeviceMappings := masterBlockDeviceMappings
	nodeUserData := &NodeScript{
		Config: k,
	}
	c.Add(nodeUserData)

	masterInstance := &Instance{
		Name: String(clusterID + "-master"),
		Subnet:              subnet,
		PrivateIPAddress:    String(k.MasterInternalIP),
		InstanceCommonConfig: InstanceCommonConfig{
			SSHKey:              sshKey,
			SecurityGroups:      []*SecurityGroup{masterSG},
			IAMInstanceProfile:  iamMasterInstanceProfile,
			ImageID:             String(k.ImageID),
			InstanceType:        String(k.MasterInstanceType),
			AssociatePublicIP:   Bool(true),
			BlockDeviceMappings: masterBlockDeviceMappings,
		},
		UserData:            masterUserData,
		Tags: map[string]string{"Role": "master"},
	}
	c.Add(masterInstance)

	c.Add(&InstanceElasticIPAttachment{Instance:masterInstance, ElasticIP: masterIP})
	c.Add(&InstanceVolumeAttachment{Instance:masterInstance, Volume: masterPV, Device: String("/dev/sdb")})

	nodeGroup := &AutoscalingGroup{
		Name:                String(clusterID + "-minion-group"),
		MinSize:             Int64(int64(k.NodeCount)),
		MaxSize:             Int64(int64(k.NodeCount)),
		Subnet:              subnet,
		Tags: map[string]string{
			"Role": "node",
		},
		InstanceCommonConfig: InstanceCommonConfig{
			SSHKey:              sshKey,
			SecurityGroups:      []*SecurityGroup{nodeSG},
			IAMInstanceProfile:  iamNodeInstanceProfile,
			ImageID:             String(k.ImageID),
			InstanceType:        String(k.NodeInstanceType),
			AssociatePublicIP:   Bool(true),
			BlockDeviceMappings: nodeBlockDeviceMappings,
		},
		UserData:            nodeUserData,
	}
	c.Add(nodeGroup)

}

func (k *K8s) GetWellKnownServiceIP(id int) (net.IP, error) {
	_, cidr, err := net.ParseCIDR(k.ServiceClusterIPRange)
	if err != nil {
		return nil, fmt.Errorf("error parsing ServiceClusterIPRange: %v", err)
	}

	ip4 := cidr.IP.To4()
	if ip4 != nil {
		n := binary.BigEndian.Uint32(ip4)
		n += uint32(id)
		serviceIP := make(net.IP, len(ip4))
		binary.BigEndian.PutUint32(serviceIP, n)
		return serviceIP, nil
	}

	ip6 := cidr.IP.To16()
	if ip6 != nil {
		baseIPInt := big.NewInt(0)
		baseIPInt.SetBytes(ip6)
		serviceIPInt := big.NewInt(0)
		serviceIPInt.Add(big.NewInt(int64(id)), baseIPInt)
		serviceIP := make(net.IP, len(ip6))
		serviceIPBytes := serviceIPInt.Bytes()
		for i := range serviceIPBytes {
			serviceIP[len(serviceIP) - len(serviceIPBytes) + i] = serviceIPBytes[i]
		}
		return serviceIP, nil
	}

	return nil, fmt.Errorf("Unexpected IP address type for ServiceClusterIPRange: %s", k.ServiceClusterIPRange)

}

func ResourceAsBase64String(r fi.Resource) string {
	if r == nil {
		return ""
	}

	data, err := fi.ResourceAsBytes(r)
	if err != nil {
		glog.Fatalf("error reading resource: %v", err)
	}

	return base64.StdEncoding.EncodeToString(data)
}