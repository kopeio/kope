package units

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
	"crypto/md5"
	"golang.org/x/crypto/ssh"
	"strings"
	"github.com/kopeio/kope/pkg/units/awsunits"
	"github.com/kopeio/kope/pkg/units/gceunits"
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
	MasterRoleDocument            fi.Resource
	MasterRolePolicy              fi.Resource

	NodeRoleDocument              fi.Resource
	NodeRolePolicy                fi.Resource

	NodeCount                     int

	InstancePrefix                string
	NodeInstancePrefix            string
	ClusterIPRange                string
	MasterIPRange                 string
	AllocateNodeCIDRs             bool

	ServerBinaryTar               fi.Downloadable
	SaltTar                       fi.Downloadable
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

func (k*K8s) BuildEnv(isMaster bool) (map[string]string, error) {
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
		url, hash, err := k.ServerBinaryTar.Resolve(fi.HashAlgorithmSHA1)
		if err != nil {
			return nil, err
		}
		y["SERVER_BINARY_TAR_URL"] = url
		y["SERVER_BINARY_TAR_HASH"] = hash
	}

	{
		url, hash, err := k.SaltTar.Resolve(fi.HashAlgorithmSHA1)
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
	y["CA_CERT"] = asBase64(k.CACert)
	y["KUBELET_CERT"] = asBase64(k.KubeletCert)
	y["KUBELET_KEY"] = asBase64(k.KubeletKey)
	y["NETWORK_PROVIDER"] = k.NetworkProvider
	y["HAIRPIN_MODE"] = k.HairpinMode
	y["OPENCONTRAIL_TAG"] = k.OpencontrailTag
	y["OPENCONTRAIL_KUBERNETES_TAG"] = k.OpencontrailKubernetesTag
	y["OPENCONTRAIL_PUBLIC_SUBNET"] = k.OpencontrailPublicSubnet
	y["E2E_STORAGE_TEST_ENVIRONMENT"] = k.E2EStorageTestEnvironment
	y["KUBE_IMAGE_TAG"] = k.KubeImageTag
	y["KUBE_DOCKER_REGISTRY"] = k.KubeDockerRegistry
	y["KUBE_ADDON_REGISTRY"] = k.KubeAddonRegistry
	if fi.BoolValue(k.Multizone) {
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
		if fi.BoolValue(k.RegisterMasterKubelet) {
			y["KUBELET_APISERVER"] = k.MasterName
		}

		y["KUBERNETES_MASTER"] = strconv.FormatBool(true)
		y["KUBE_USER"] = k.KubeUser
		y["KUBE_PASSWORD"] = k.KubePassword
		y["KUBE_BEARER_TOKEN"] = k.BearerToken
		y["MASTER_CERT"] = asBase64(k.MasterCert)
		y["MASTER_KEY"] = asBase64(k.MasterKey)
		y["KUBECFG_CERT"] = asBase64(k.KubecfgCert)
		y["KUBECFG_KEY"] = asBase64(k.KubecfgKey)

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
	y["CA_KEY"] = asBase64(k.CAKey) // https://github.com/kubernetes/kubernetes/issues/23264

	return y, nil
}

func (k*K8s) Init() {
	k.MasterInternalIP = "172.20.0.9"
	k.NodeCount = 2
	k.DockerStorage = "aufs"
	k.MasterIPRange = "10.246.0.0/24"
	//k.MasterVolumeType = "gp2"
	k.MasterVolumeSize = fi.Int(DefaultMasterVolumeSize)
	//k.Zone = "us-east-1b"
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

	//k.CloudProvider = "aws"
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

	if k.CloudProvider == "aws" {
		k.addAWS(c)
		return
	}

	if k.CloudProvider == "gce" {
		k.addGCE(c)
		return
	}

	glog.Exit("CloudProvider not recognized: %q", k.CloudProvider)
	return
}

func (k *K8s) addAWS(c *fi.BuildContext) {
	clusterID := k.ClusterID
	if clusterID == "" {
		glog.Exit("cluster-id is required")
	}

	if len(k.Zone) <= 2 {
		glog.Exit("Invalid AZ: ", k.Zone)
	}

	if k.ImageID == "" {
		jessie := &awsunits.DistroJessie{}
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

	masterVolumeSize := DefaultMasterVolumeSize
	if k.MasterVolumeSize != nil {
		masterVolumeSize = *k.MasterVolumeSize
	}
	masterVolumeType := "gp2"
	if k.MasterVolumeType != "" {
		masterVolumeType = k.MasterVolumeType
	}
	masterPV := &awsunits.PersistentVolume{
		AvailabilityZone:         fi.String(k.Zone),
		SizeGB:       fi.Int64(int64(masterVolumeSize)),
		VolumeType: fi.String(masterVolumeType),
		Name:    fi.String(clusterID + "-master-pd"),
	}
	c.Add(masterPV)

	masterIP := &awsunits.ElasticIP{
		PublicIP: k.MasterElasticIP,
		TagOnResource: masterPV,
		TagUsingKey: fi.String("kubernetes.io/master-ip"),
	}
	c.Add(masterIP)

	c.Add(&CertBuilder{Kubernetes: k, MasterIP: masterIP})

	iamMasterRole := &awsunits.IAMRole{
		Name:               fi.String("kubernetes-master"),
		RolePolicyDocument: k.MasterRoleDocument,
	}
	c.Add(iamMasterRole)

	iamMasterRolePolicy := &awsunits.IAMRolePolicy{
		Role:           iamMasterRole,
		Name:           fi.String("kubernetes-master"),
		PolicyDocument: k.MasterRolePolicy,
	}
	c.Add(iamMasterRolePolicy)

	iamMasterInstanceProfile := &awsunits.IAMInstanceProfile{
		Name: fi.String("kubernetes-master"),
	}
	c.Add(iamMasterInstanceProfile)

	iamMasterInstanceProfileRole := &awsunits.IAMInstanceProfileRole{
		InstanceProfile: iamMasterInstanceProfile,
		Role: iamMasterRole,
	}
	c.Add(iamMasterInstanceProfileRole)

	iamNodeRole := &awsunits.IAMRole{
		Name:             fi.String("kubernetes-minion"),
		RolePolicyDocument: k.NodeRoleDocument,
	}
	c.Add(iamNodeRole)

	iamNodeRolePolicy := &awsunits.IAMRolePolicy{
		Role:           iamNodeRole,
		Name:          fi.String("kubernetes-minion"),
		PolicyDocument: k.NodeRolePolicy,
	}
	c.Add(iamNodeRolePolicy)

	iamNodeInstanceProfile := &awsunits.IAMInstanceProfile{
		Name:fi.String("kubernetes-minion"),
	}
	c.Add(iamNodeInstanceProfile)

	iamNodeInstanceProfileRole := &awsunits.IAMInstanceProfileRole{
		InstanceProfile: iamNodeInstanceProfile,
		Role: iamNodeRole,
	}
	c.Add(iamNodeInstanceProfileRole)

	sshKey := &awsunits.SSHKey{PublicKey: k.SSHPublicKey}
	if sshKey.Name == nil && sshKey.PublicKey != nil {
		sshPublicKeyAuth, err := fi.ResourceAsString(sshKey.PublicKey)
		if err != nil {
			glog.Exitf("error reading SSH public key: %v", err)
		}

		tokens := strings.Split(sshPublicKeyAuth, " ")
		if len(tokens) < 2 {
			glog.Exitf("error parsing SSH public key: %s", sshPublicKeyAuth)
		}

		sshPublicKeyBytes, err := base64.StdEncoding.DecodeString(tokens[1])
		if len(tokens) < 2 {
			glog.Exitf("error decoding SSH public key: %s", sshPublicKeyAuth)
		}

		// We don't technically need to parse and remarshal it, but it ensures the key is valid
		sshPublicKey, err := ssh.ParsePublicKey(sshPublicKeyBytes)
		if err != nil {
			glog.Exitf("error parsing SSH public key: %v", err)
		}

		h := md5.Sum(sshPublicKey.Marshal())
		sshKeyFingerprint := fmt.Sprintf("%x", h)
		sshKey.Name = fi.String("kubernetes-" + sshKeyFingerprint)

	}
	c.Add(sshKey)

	vpc := &awsunits.VPC{
		ID: k.VPCID,
		CIDR:fi.String("172.20.0.0/16"),
		Name: fi.String("kubernetes-" + clusterID),
		EnableDNSSupport:fi.Bool(true),
		EnableDNSHostnames:fi.Bool(true),
	}
	c.Add(vpc)

	region := c.Cloud().(*fi.AWSCloud).Region
	dhcpDomainName := region + ".compute.internal"
	if region == "us-east-1" {
		dhcpDomainName = "ec2.internal"
	}
	dhcpOptions := &awsunits.DHCPOptions{
		ID: k.DHCPOptionsID,
		Name: fi.String("kubernetes-" + clusterID),
		DomainName: fi.String(dhcpDomainName),
		DomainNameServers: fi.String("AmazonProvidedDNS"),
	}
	c.Add(dhcpOptions)

	c.Add(&awsunits.VPCDHCPOptionsAssociation{VPC: vpc, DHCPOptions: dhcpOptions })

	subnet := &awsunits.Subnet{
		VPC: vpc,
		AvailabilityZone: fi.String(k.Zone),
		CIDR: fi.String("172.20.0.0/24"),
		Name: fi.String("kubernetes-" + clusterID),
		ID: k.SubnetID,
	}
	c.Add(subnet)

	igw := &awsunits.InternetGateway{Name: fi.String("kubernetes-" + clusterID), ID: k.InternetGatewayID}
	c.Add(igw)

	c.Add(&awsunits.InternetGatewayAttachment{VPC: vpc, InternetGateway: igw})

	routeTable := &awsunits.RouteTable{VPC: vpc, Name: fi.String("kubernetes-" + clusterID), ID: k.RouteTableID}
	c.Add(routeTable)

	route := &awsunits.Route{RouteTable: routeTable, CIDR: fi.String("0.0.0.0/0"), InternetGateway: igw}
	c.Add(route)

	c.Add(&awsunits.RouteTableAssociation{RouteTable: routeTable, Subnet: subnet})

	masterSG := &awsunits.SecurityGroup{
		Name:        fi.String("kubernetes-master-" + clusterID),
		Description: fi.String("Security group for master nodes"),
		VPC:         vpc}
	c.Add(masterSG)

	nodeSG := &awsunits.SecurityGroup{
		Name:        fi.String("kubernetes-minion-" + clusterID),
		Description: fi.String("Security group for minion nodes"),
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

	masterUserData := &awsunits.MasterScript{
		Construct: func(c*fi.RunContext) (string, error) {
			isMaster := true
			return buildAWSScript(c, k, isMaster)
		},
	}
	c.Add(masterUserData)

	masterBlockDeviceMappings := []*awsunits.BlockDeviceMapping{}

	// Be sure to map all the ephemeral drives.  We can specify more than we actually have.
	// TODO: Actually mount the correct number (especially if we have more), though this is non-trivial, and
	//  only affects the big storage instance types, which aren't a typical use case right now.
	for i := 0; i < 4; i++ {
		bdm := &awsunits.BlockDeviceMapping{
			DeviceName:  fi.String("/dev/sd" + string('c' + i)),
			VirtualName: fi.String("ephemeral" + strconv.Itoa(i)),
		}
		masterBlockDeviceMappings = append(masterBlockDeviceMappings, bdm)
	}

	nodeBlockDeviceMappings := masterBlockDeviceMappings
	nodeUserData := &awsunits.NodeScript{
		Construct: func(c*fi.RunContext) (string, error) {
			isMaster := false
			return buildAWSScript(c, k, isMaster)
		},
	}
	c.Add(nodeUserData)

	masterInstanceType := k.MasterInstanceType
	if masterInstanceType == "" {
		masterInstanceType = "m3.medium"
	}

	masterInstance := &awsunits.Instance{
		Name: fi.String(clusterID + "-master"),
		Subnet:              subnet,
		PrivateIPAddress:    fi.String(k.MasterInternalIP),
		InstanceCommonConfig: awsunits.InstanceCommonConfig{
			SSHKey:              sshKey,
			SecurityGroups:      []*awsunits.SecurityGroup{masterSG},
			IAMInstanceProfile:  iamMasterInstanceProfile,
			ImageID:             fi.String(k.ImageID),
			InstanceType:        fi.String(k.MasterInstanceType),
			AssociatePublicIP:   fi.Bool(true),
			BlockDeviceMappings: masterBlockDeviceMappings,
		},
		UserData:            masterUserData,
		Tags: map[string]string{"Role": "master"},
	}
	c.Add(masterInstance)

	c.Add(&awsunits.InstanceElasticIPAttachment{Instance:masterInstance, ElasticIP: masterIP})
	c.Add(&awsunits.InstanceVolumeAttachment{Instance:masterInstance, Volume: masterPV, Device: fi.String("/dev/sdb")})

	nodeInstanceType := k.NodeInstanceType
	if nodeInstanceType == "" {
		nodeInstanceType = "m3.medium"
	}

	nodeGroup := &awsunits.AutoscalingGroup{
		Name:                fi.String(clusterID + "-minion-group"),
		MinSize:             fi.Int64(int64(k.NodeCount)),
		MaxSize:             fi.Int64(int64(k.NodeCount)),
		Subnet:              subnet,
		Tags: map[string]string{
			"Role": "node",
		},
		InstanceCommonConfig: awsunits.InstanceCommonConfig{
			SSHKey:              sshKey,
			SecurityGroups:      []*awsunits.SecurityGroup{nodeSG},
			IAMInstanceProfile:  iamNodeInstanceProfile,
			ImageID:             fi.String(k.ImageID),
			InstanceType:        fi.String(k.NodeInstanceType),
			AssociatePublicIP:   fi.Bool(true),
			BlockDeviceMappings: nodeBlockDeviceMappings,
		},
		UserData:            nodeUserData,
	}
	c.Add(nodeGroup)

}

func buildAWSScript(c *fi.RunContext, k *K8s, isMaster bool) (string, error) {
	var bootstrapScriptURL string

	{
		url, _, err := c.Target.FileStore().PutResource("bootstrap", k.BootstrapScript, fi.HashAlgorithmSHA1)
		if err != nil {
			return "", err
		}
		bootstrapScriptURL = url
	}

	data, err := k.BuildEnv(isMaster)
	if err != nil {
		return "", err
	}

	if k.CloudProvider == "aws" {
		data["AUTO_UPGRADE"] = strconv.FormatBool(true)
		// TODO: get rid of these exceptions / harmonize with common or GCE
		data["DOCKER_STORAGE"] = k.DockerStorage
		data["API_SERVERS"] = k.MasterInternalIP
	}

	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("error marshaling env to yaml: %v", err)
	}

	// We send this to the ami as a startup script in the user-data field.  Requires a compatible ami
	var s fi.ScriptWriter
	s.WriteString("#! /bin/bash\n")
	s.WriteString("mkdir -p /var/cache/kubernetes-install\n")
	s.WriteString("cd /var/cache/kubernetes-install\n")

	s.WriteHereDoc("kube_env.yaml", string(yamlData))

	s.WriteString("wget -O bootstrap " + bootstrapScriptURL + "\n")
	s.WriteString("chmod +x bootstrap\n")
	s.WriteString("mkdir -p /etc/kubernetes\n")
	s.WriteString("mv kube_env.yaml /etc/kubernetes\n")
	s.WriteString("mv bootstrap /etc/kubernetes/\n")

	s.WriteString("cat > /etc/rc.local << EOF_RC_LOCAL\n")
	s.WriteString("#!/bin/sh -e\n")
	// We want to be sure that we don't pass an argument to bootstrap
	s.WriteString("/etc/kubernetes/bootstrap\n")
	s.WriteString("exit 0\n")
	s.WriteString("EOF_RC_LOCAL\n")
	s.WriteString("/etc/kubernetes/bootstrap\n")

	return s.AsString(), nil
}

func (k *K8s) addGCE(c *fi.BuildContext) {
	if len(k.Zone) <= 2 {
		glog.Exit("Invalid Zone: ", k.Zone)
	}

	zone := k.Zone

	networkName := "default"

	//if k.ImageID == "" {
	//	jessie := &DistroJessie{}
	//	imageID, err := jessie.GetImageID(c.Context)
	//	if err != nil {
	//		glog.Exitf("error while trying to find AWS image: %v", err)
	//	}
	//	k.ImageID = imageID
	//}
	//region := az[:len(az) - 1]
	//

	clusterID := k.ClusterID

	// Note that some of these defaults are written back to the config, so that they end up in node_env
	if k.InstancePrefix == "" {
		k.InstancePrefix = clusterID
	}

	if k.NodeInstancePrefix == "" {
		k.NodeInstancePrefix = k.InstancePrefix + "-minion"
	}

	if k.MasterName == "" {
		k.MasterName = k.InstancePrefix + "-master"
	}

	masterTag := k.InstancePrefix + "-master"
	nodeTag := k.InstancePrefix + "-minion"

	if k.ClusterIPRange == "" {
		k.ClusterIPRange = "10.244.0.0/16"
	}

	network := &gceunits.Network{
		CIDR: fi.String("10.240.0.0/16"),
		Name: fi.String(networkName),
	}
	c.Add(network)

	masterVolumeSize := DefaultMasterVolumeSize
	if k.MasterVolumeSize != nil {
		masterVolumeSize = *k.MasterVolumeSize
	}
	masterVolumeType := "pd-ssd"
	if k.MasterVolumeType != "" {
		masterVolumeType = k.MasterVolumeType
	}
	masterPV := &gceunits.PersistentDisk{
		Name:    fi.String(k.MasterName + "-pd"),
		Zone:         fi.String(k.Zone),
		SizeGB:       fi.Int64(int64(masterVolumeSize)),
		VolumeType: fi.String(masterVolumeType),
	}
	c.Add(masterPV)

	// Open master HTTPS
	c.Add(&gceunits.FirewallRule{
		Name: fi.String(k.MasterName + "-https"),
		Network: network,
		SourceRanges: []string{"0.0.0.0/0"},
		TargetTags: []string{masterTag},
		Allowed: []string{"tcp:443"},
	})

	masterIP := &gceunits.IPAddress{
		Name: fi.String(k.MasterName + "-ip"),
		Address: k.MasterElasticIP,
	}
	c.Add(masterIP)

	c.Add(&CertBuilder{
		Kubernetes: k,
		MasterIP: masterIP,
	})

	//iamMasterRole := &IAMRole{
	//	Name:               String("kubernetes-master"),
	//	RolePolicyDocument: k.MasterRoleDocument,
	//}
	//c.Add(iamMasterRole)
	//
	//iamMasterRolePolicy := &IAMRolePolicy{
	//	Role:           iamMasterRole,
	//	Name:           String("kubernetes-master"),
	//	PolicyDocument: k.MasterRolePolicy,
	//}
	//c.Add(iamMasterRolePolicy)
	//
	//iamMasterInstanceProfile := &IAMInstanceProfile{
	//	Name: String("kubernetes-master"),
	//}
	//c.Add(iamMasterInstanceProfile)
	//
	//iamMasterInstanceProfileRole := &IAMInstanceProfileRole{
	//	InstanceProfile: iamMasterInstanceProfile,
	//	Role: iamMasterRole,
	//}
	//c.Add(iamMasterInstanceProfileRole)
	//
	//iamNodeRole := &IAMRole{
	//	Name:             String("kubernetes-minion"),
	//	RolePolicyDocument: k.NodeRoleDocument,
	//}
	//c.Add(iamNodeRole)
	//
	//iamNodeRolePolicy := &IAMRolePolicy{
	//	Role:           iamNodeRole,
	//	Name:          String("kubernetes-minion"),
	//	PolicyDocument: k.NodeRolePolicy,
	//}
	//c.Add(iamNodeRolePolicy)
	//
	//iamNodeInstanceProfile := &IAMInstanceProfile{
	//	Name:String("kubernetes-minion"),
	//}
	//c.Add(iamNodeInstanceProfile)
	//
	//iamNodeInstanceProfileRole := &IAMInstanceProfileRole{
	//	InstanceProfile: iamNodeInstanceProfile,
	//	Role: iamNodeRole,
	//}
	//c.Add(iamNodeInstanceProfileRole)
	//
	//sshKey := &SSHKey{PublicKey: k.SSHPublicKey}
	//if sshKey.Name == nil && sshKey.PublicKey != nil {
	//	sshPublicKeyAuth, err := fi.ResourceAsString(sshKey.PublicKey)
	//	if err != nil {
	//		glog.Exitf("error reading SSH public key: %v", err)
	//	}
	//
	//	tokens := strings.Split(sshPublicKeyAuth, " ")
	//	if len(tokens) < 2 {
	//		glog.Exitf("error parsing SSH public key: %s", sshPublicKeyAuth)
	//	}
	//
	//	sshPublicKeyBytes, err := base64.StdEncoding.DecodeString(tokens[1])
	//	if len(tokens) < 2 {
	//		glog.Exitf("error decoding SSH public key: %s", sshPublicKeyAuth)
	//	}
	//
	//	// We don't technically need to parse and remarshal it, but it ensures the key is valid
	//	sshPublicKey, err := ssh.ParsePublicKey(sshPublicKeyBytes)
	//	if err != nil {
	//		glog.Exitf("error parsing SSH public key: %v", err)
	//	}
	//
	//	h := md5.Sum(sshPublicKey.Marshal())
	//	sshKeyFingerprint := fmt.Sprintf("%x", h)
	//	sshKey.Name = String("kubernetes-" + sshKeyFingerprint)
	//
	//}
	//c.Add(sshKey)


	//region := c.Cloud().(*fi.AWSCloud).Region
	//dhcpDomainName := region + ".compute.internal"
	//if region == "us-east-1" {
	//	dhcpDomainName = "ec2.internal"
	//}
	//dhcpOptions := &DHCPOptions{
	//	ID: k.DHCPOptionsID,
	//	Name: String("kubernetes-" + clusterID),
	//	DomainName: String(dhcpDomainName),
	//	DomainNameServers: String("AmazonProvidedDNS"),
	//}
	//c.Add(dhcpOptions)
	//
	//c.Add(&VPCDHCPOptionsAssociation{VPC: vpc, DHCPOptions: dhcpOptions })
	//
	//subnet := &Subnet{VPC: vpc, AvailabilityZone: String(k.Zone), CIDR: String("172.20.0.0/24"), Name: String("kubernetes-" + clusterID), ID: k.SubnetID}
	//c.Add(subnet)
	//
	//igw := &InternetGateway{Name: String("kubernetes-" + clusterID), ID: k.InternetGatewayID}
	//c.Add(igw)
	//
	//c.Add(&InternetGatewayAttachment{VPC: vpc, InternetGateway: igw})
	//
	//routeTable := &RouteTable{VPC: vpc, Name: String("kubernetes-" + clusterID), ID: k.RouteTableID}
	//c.Add(routeTable)
	//
	//route := &Route{RouteTable: routeTable, CIDR: String("0.0.0.0/0"), InternetGateway: igw}
	//c.Add(route)
	//
	//c.Add(&RouteTableAssociation{RouteTable: routeTable, Subnet: subnet})
	//
	//masterSG := &SecurityGroup{
	//	Name:        String("kubernetes-master-" + clusterID),
	//	Description: String("Security group for master nodes"),
	//	VPC:         vpc}
	//c.Add(masterSG)
	//
	//nodeSG := &SecurityGroup{
	//	Name:        String("kubernetes-minion-" + clusterID),
	//	Description: String("Security group for minion nodes"),
	//	VPC:         vpc}
	//c.Add(nodeSG)
	//
	//c.Add(masterSG.AllowFrom(masterSG))
	//c.Add(masterSG.AllowFrom(nodeSG))
	//c.Add(nodeSG.AllowFrom(masterSG))
	//c.Add(nodeSG.AllowFrom(nodeSG))
	//

	// Allow all internal traffic
	c.Add(&gceunits.FirewallRule{
		Name: fi.String(networkName + "-default-internal"),
		Network: network,
		SourceRanges: []string{"10.0.0.0/8"},
		Allowed: []string{"tcp:1-65535", "udp:1-65535", "icmp"},
	})

	// SSH is open to the world
	c.Add(&gceunits.FirewallRule{
		Name: fi.String(networkName + "-default-ssh"),
		Network: network,
		SourceRanges: []string{"0.0.0.0/0"},
		Allowed: []string{"tcp:22"},
	})


	//c.Add(nodeSG.AllowTCP("0.0.0.0/0", 22, 22))
	//c.Add(masterSG.AllowTCP("0.0.0.0/0", 22, 22))
	//
	//// HTTPS to the master is allowed (for API access)
	//c.Add(masterSG.AllowTCP("0.0.0.0/0", 443, 443))
	//
	//masterUserData := &MasterScript{
	//	Config: k,
	//}
	//c.Add(masterUserData)
	//
	//masterBlockDeviceMappings := []*BlockDeviceMapping{}
	//
	//// Be sure to map all the ephemeral drives.  We can specify more than we actually have.
	//// TODO: Actually mount the correct number (especially if we have more), though this is non-trivial, and
	////  only affects the big storage instance types, which aren't a typical use case right now.
	//for i := 0; i < 4; i++ {
	//	bdm := &BlockDeviceMapping{
	//		DeviceName:  String("/dev/sd" + string('c' + i)),
	//		VirtualName: String("ephemeral" + strconv.Itoa(i)),
	//	}
	//	masterBlockDeviceMappings = append(masterBlockDeviceMappings, bdm)
	//}
	//
	//nodeBlockDeviceMappings := masterBlockDeviceMappings
	//nodeUserData := &NodeScript{
	//	Config: k,
	//}
	//c.Add(nodeUserData)

	// TODO: Make configurable
	masterImage := "google-containers/container-vm-v20160321"

	nodeCount := k.NodeCount

	masterInstanceType := k.MasterInstanceType
	if masterInstanceType == "" {
		if nodeCount > 500 {
			masterInstanceType = "n1-standard-32"
		} else if nodeCount > 250 {
			masterInstanceType = "n1-standard-16"
		} else if nodeCount > 100 {
			masterInstanceType = "n1-standard-8"
		} else if nodeCount > 10 {
			masterInstanceType = "n1-standard-4"
		} else if nodeCount > 5 {
			masterInstanceType = "n1-standard-2"
		} else {
			masterInstanceType = "n1-standard-1"
		}
	}
	masterMetadata := map[string]fi.Resource{
		"startup-script": k.BootstrapScript, // cluster/gce/configure-vm.sh
		"kube-env": fi.NewFuncResource(func() ([]byte, error) {
			isMaster := true
			data, err := k.BuildEnv(isMaster)
			if err != nil {
				return nil, err
			}

			yamlData, err := yaml.Marshal(data)
			if err != nil {
				return nil, fmt.Errorf("error marshaling env to yaml: %v", err)
			}
			return yamlData, nil
		}),
		"cluster-name"        : fi.NewStringResource(clusterID),
	}
	masterInstance := &gceunits.Instance{
		Name: &k.MasterName,
		IPAddress: masterIP,
		Zone: &zone,
		MachineType: &masterInstanceType,
		Image: &masterImage,
		Tags: []string{masterTag},
		Network: network,
		Scopes: []string{"storage-ro", "compute-rw", "monitoring", "logging-write"},
		CanIPForward: fi.Bool(true),
		Metadata: masterMetadata,
		Disks: map[string]*gceunits.PersistentDisk{
			"master-pd": masterPV,
		},
	}
	// TODO: Preemptible master
	masterInstance.Preemptible = fi.Bool(false)
	c.Add(masterInstance)



	//c.Add(&InstanceElasticIPAttachment{Instance:masterInstance, ElasticIP: masterIP})
	//c.Add(&InstanceVolumeAttachment{Instance:masterInstance, Volume: masterPV, Device: String("/dev/sdb")})
	//
	//nodeGroup := &AutoscalingGroup{
	//	Name:                String(clusterID + "-minion-group"),
	//	MinSize:             Int64(int64(k.NodeCount)),
	//	MaxSize:             Int64(int64(k.NodeCount)),
	//	Subnet:              subnet,
	//	Tags: map[string]string{
	//		"Role": "node",
	//	},
	//	InstanceCommonConfig: InstanceCommonConfig{
	//		SSHKey:              sshKey,
	//		SecurityGroups:      []*SecurityGroup{nodeSG},
	//		IAMInstanceProfile:  iamNodeInstanceProfile,
	//		ImageID:             String(k.ImageID),
	//		InstanceType:        String(k.NodeInstanceType),
	//		AssociatePublicIP:   Bool(true),
	//		BlockDeviceMappings: nodeBlockDeviceMappings,
	//	},
	//	UserData:            nodeUserData,
	//}
	//c.Add(nodeGroup)

	// Allow traffic from nodes -> nodes
	c.Add(&gceunits.FirewallRule{
		Name: fi.String(nodeTag + "-all"),
		Network: network,
		SourceRanges: []string{k.ClusterIPRange},
		TargetTags: []string{nodeTag},
		Allowed: []string{"tcp", "udp", "icmp", "esp", "ah", "sctp"},
	})

	nodeMachineType := k.NodeInstanceType
	if nodeMachineType == "" {
		nodeMachineType = "n1-standard-2"
	}
	// TODO: Make configurable
	nodeDiskType := "pd-standard"
	nodeDiskSize := int64(100)
	nodeImage := "google-containers/container-vm-v20160321"

	nodeMetadata := map[string]fi.Resource{
		"startup-script": k.BootstrapScript, // cluster/gce/configure-vm.sh
		"kube-env": fi.NewFuncResource(func() ([]byte, error) {
			isMaster := false
			data, err := k.BuildEnv(isMaster)
			if err != nil {
				return nil, err
			}

			yamlData, err := yaml.Marshal(data)
			if err != nil {
				return nil, fmt.Errorf("error marshaling env to yaml: %v", err)
			}
			return yamlData, nil
		}),
		"cluster-name"        : fi.NewStringResource(clusterID),
	}
	nodeTemplate := &gceunits.InstanceTemplate{
		Name: fi.String(k.NodeInstancePrefix + "-template"),
		Network: network,
		MachineType: &nodeMachineType,
		BootDiskType: &nodeDiskType,
		BootDiskSizeGB: &nodeDiskSize,
		BootDiskImage: &nodeImage,
		Tags: []string{nodeTag},
		CanIPForward: fi.Bool(true),
		Metadata: nodeMetadata,
	}

	nodeTemplate.Scopes = []string{"compute-rw", "monitoring", "logging-write", "storage-ro"}

	// TODO: Support preemptible nodes?
	nodeTemplate.Preemptible = fi.Bool(false)
	c.Add(nodeTemplate)

	// TODO: Support mulitple instance groups
	nodeInstances := &gceunits.ManagedInstanceGroup{
		Name: fi.String(k.NodeInstancePrefix + "-group"),
		Zone: &zone,
		BaseInstanceName: fi.String(k.NodeInstancePrefix),
		TargetSize: fi.Int64(int64(nodeCount)),
		InstanceTemplate: nodeTemplate,
	}
	c.Add(nodeInstances)



	// CREATE AUTOSCALER

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

func asBase64(r fi.Resource) string {
	if r == nil {
		return ""
	}
	s, err := fi.ResourceAsBase64String(r)
	if err != nil {
		glog.Exitf("error rendering resource: %v", err)
	}

	return s
}