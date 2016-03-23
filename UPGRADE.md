#  Upgrade a cluster from 1.1 to 1.2

```
mkdir ~/upgrade
export GOPATH=~/upgrade
go get github.com/kopeio/kope
export PATH=$PATH:~/upgrade/bin
~```

# Set the REGION env var to the region where your cluster is
REGION=us-west-2

# List all the clusters in ${REGION}
> kope discover clusters --region ${REGION}
kubernetes	52.27.60.160	us-west-2a

# The first column is your cluster ID (likely `kubernetes`)
# The second column is your master IP, and should match the output of `kubectl cluster-info`
# The third column is your zone

# Set the MASTER_IP & ZONE env vars to match your cluster (as discovered above)
MASTER_IP=52.27.60.160
ZONE=us-west-2a

# You will also need the IP address of one of your nodes
kubectl get nodes -ojson | grep -A1 ExternalIP

# Extract out one of the 'ExternalIP' values
NODE_IP=1.2.3.4

# Set ssh_key to the SSH key you're using with your cluster
ssh_key=~/.ssh/kube_aws_rsa
_
# Extract the configuration and keys from your existing cluster
kope export cluster --master ${MASTER_IP} -i ${ssh_key} --logtostderr --node ${NODE_IP} --dest upgrade11/

# This should have extracted keys & configuration from the running cluster:
find upgrade11/
# should show ....
upgrade11/
upgrade11/pki
upgrade11/pki/ca.crt
upgrade11/pki/issued
upgrade11/pki/issued/cn=kubernetes-master.crt
upgrade11/pki/issued/cn=kubecfg.crt
upgrade11/pki/issued/cn=kubelet.crt
upgrade11/pki/private
upgrade11/pki/private/cn=kubernetes-master.key
upgrade11/pki/private/cn=kubecfg.key
upgrade11/pki/private/cn=kubelet.key
upgrade11/kubernetes.yaml



# kubernetes.yaml has your configuration
> cat upgrade11/kubernetes.yaml
AllocateNodeCIDRs: true
CloudProvider: aws
ClusterID: kubernetes
ClusterIPRange: 10.244.0.0/16
DNSDomain: cluster.local
DNSReplicas: 1
DNSServerIP: 10.0.0.10
DockerStorage: aufs
ElasticsearchLoggingReplicas: 1
EnableClusterDNS: true
EnableClusterLogging: true
EnableClusterMonitoring: influxdb
EnableClusterUI: true
EnableCustomMetrics: false
EnableNodeLogging: true
InternetGatewayID: igw-db10afbf
KubePassword: PTajI3M5mEdoicTo
KubeProxyToken: TraE1N4igFN8gChk65LT1S3VhWea2mwr
KubeUser: admin
KubeletToken: QK5ozLccTx5OshOAcZEVYz1TDOsP9NR1
LoggingDestination: elasticsearch
MasterIPRange: 10.246.0.0/24
NodeCount: 4
RouteTableID: rtb-2c1dc94b
ServiceClusterIPRange: 10.0.0.0/16
SubnetID: subnet-d32547a5
VPCID: vpc-d9080cbd
Zone: us-west-2a

# Download the kubernetes version that you are going to install
release=v1.2.0
mkdir release-${release}
wget https://storage.googleapis.com/kubernetes-release/release/${release}/kubernetes.tar.gz -O release-${release}/kubernetes.tar.gz
tar zxf  release-${release}/kubernetes.tar.gz -C  release-${release}

# See what changes will be made if we apply the cluster changes
kope create cluster -i ${ssh_key} -d upgrade11/ -r release-${release}/kubernetes/ --logtostderr -t dryrun

You should see something like this:

```
Upload resources:
  bootstrap     1ff7f9b7f9596fe2224d4d4765c98161b89ba33d
  server        289db0f893b47fc00461204dcdddef457def08b6
  salt  eb4aa52f1b81223f36532acee17cefdef7232a24
Created resources:
  *awsunits.ElasticIP   ElasticIP-kubernetes.io/master-ip
  *awsunits.InstanceElasticIPAttachment InstanceElasticIPAttachment-kubernetes-master-kubernetes.io/master-ip
Changed resources:
  *awsunits.IAMRole     IAMRole-kubernetes-master
    RolePolicyDocument <resource> -> <resource>

  *awsunits.IAMRolePolicy       IAMRolePolicy-kubernetes-master
    Role id:kubernetes-master -> id:AROAILHQRUIC4GMMJFPZS
    PolicyDocument <resource> -> <resource>

  *awsunits.IAMRole     IAMRole-kubernetes-minion
    RolePolicyDocument <resource> -> <resource>

  *awsunits.IAMRolePolicy       IAMRolePolicy-kubernetes-minion
    Role id:kubernetes-minion -> id:AROAJQYAOWIFBGC7M4J64
    PolicyDocument <resource> -> <resource>

  *awsunits.SSHKey      SSHKey-kubernetes-kubernetes
    PublicKey <nil> -> <resource>

  *awsunits.VPC VPC-kubernetes-kubernetes
    Name <nil> -> kubernetes-kubernetes

  *awsunits.Subnet      Subnet-kubernetes-kubernetes
    Name <nil> -> kubernetes-kubernetes

  *awsunits.InternetGateway     InternetGateway-kubernetes-kubernetes
    Name <nil> -> kubernetes-kubernetes

  *awsunits.SecurityGroup       SecurityGroup-kubernetes-master-kubernetes
    Description Kubernetes security group applied to master nodes -> Security group for master nodes

  *awsunits.SecurityGroup       SecurityGroup-kubernetes-minion-kubernetes
    Description Kubernetes security group applied to minion nodes -> Security group for minion nodes

  *awsunits.Instance    Instance-kubernetes-master
    InstanceCommonConfig awsunits.InstanceCommonConfig ({<nil> <nil> <nil> [] <nil> [] <nil>}) -> awsunits.InstanceCommonConfig ({0xc82038e040 0xc82038e050 SSHKey (name=%!s(*string=0xc820467c30)) [0xc820226800] 0xc820467fe9 [0xc820467f10 0xc820467f60 0xc820467fc0 0xc82038e010] 0xc820373350})
    UserData <nil> -> <resource>
    Tags map[string]string (map[]) -> map[string]string (map[Role:master])

  *awsunits.AutoscalingGroup    AutoscalingGroup-kubernetes-minion-group
    InstanceCommonConfig awsunits.InstanceCommonConfig ({0xc8202b6028 0xc8204567a8 SSHKey (name=%!s(*string=0xc8204567f8)) [0xc820577700] 0xc82045ab78 [0xc82045b290 0xc82045b2b0 0xc82045b2e0 0xc82045b2f0 0xc82045b300] 0xc82043c5d0}) -> awsunits.InstanceCommonConfig ({0xc82038e090 0xc82038e0a0 SSHKey (name=%!s(*string=0xc820467c30)) [0xc8202268c0] 0xc82038e0b0 [0xc820467f10 0xc820467f60 0xc820467fc0 0xc82038e010] 0xc820373530})
    UserData <resource> -> <resource>
    Tags map[string]string (map[KubernetesCluster:kubernetes Name:kubernetes-minion Role:kubernetes-minion]) -> map[string]string (map[Role:node])

```

You may prefer the bash-view:

```
kope create cluster -i ${ssh_key} -d upgrade11/ -r release-${release}/kubernetes/ --logtostderr -t bash
```

```
#!/bin/bash
set -ex

. ./helpers

export AWS_DEFAULT_OUTPUT="text"
export AWS_DEFAULT_REGION="us-west-2"
PERSISTENTVOLUME_1="vol-14355ee2"
ELASTICIP_1=`aws ec2 allocate-address --domain vpc --query AllocationId`
ELASTICIP_1_PUBLICIP=`aws ec2 describe-addresses --allocation-ids ${ELASTICIP_1} --query Addresses[].PublicIp`
add-tag ${PERSISTENTVOLUME_1} kubernetes.io/master-ip ${ELASTICIP_1_PUBLICIP}
IAMROLE_1="AROAILHQRUIC4GMMJFPZS"
IAMROLE_2="AROAJQYAOWIFBGC7M4J64"
VPC_1="vpc-2373b447"
add-tag ${VPC_1} "Name" "kubernetes-kubernetes"
DHCPOPTIONS_1="dopt-54021936"
SUBNET_1="subnet-0b3ff16f"
add-tag ${SUBNET_1} "Name" "kubernetes-kubernetes"
INTERNETGATEWAY_1="igw-2c98bd49"
add-tag ${INTERNETGATEWAY_1} "Name" "kubernetes-kubernetes"
add-tag ${INTERNETGATEWAY_1} "KubernetesCluster" "kubernetes"
ROUTETABLE_1="rtb-f1b51395"
add-tag ${ROUTETABLE_1} "Name" "kubernetes-kubernetes"
ROUTETABLEASSOCIATION_1="rtbassoc-6df78a09"
SECURITYGROUP_1="sg-14670573"
add-tag ${SECURITYGROUP_1} "Name" "kubernetes-master-kubernetes"
SECURITYGROUP_2="sg-1967057e"
add-tag ${SECURITYGROUP_2} "Name" "kubernetes-minion-kubernetes"
INSTANCE_1="i-82653e45"
add-tag ${INSTANCE_1} "Role" "master"
wait-for-instance-state ${INSTANCE_1} running
aws ec2 associate-address --allocation-id ${ELASTICIP_1} --instance-id ${INSTANCE_1}
aws autoscaling create-launch-configuration --launch-configuration-name kubernetes-minion-group-20160323T051336Z --image-id ami-7840ac18 --instance-type m3.medium --key-name kubernetes-kubernetes --associate-public-ip-address --block-device-mappings "[{\"DeviceName\":\"/dev/sdc\",\"VirtualName\":\"ephemeral0\"},{\"DeviceName\":\"/dev/sdd\",\"VirtualName\":\"ephemeral1\"},{\"DeviceName\":\"/dev/sde\",\"VirtualName\":\"ephemeral2\"},{\"DeviceName\":\"/dev/sdf\",\"VirtualName\":\"ephemeral3\"}]" --security-groups ${SECURITYGROUP_2} --iam-instance-profile kubernetes-minion --user-data file://bash_resources/NodeScript_1
aws autoscaling update-auto-scaling-group --auto-scaling-group-name kubernetes-minion-group --launch-configuration-name kubernetes-minion-group-20160323T051336Z
```

Mostly we are adding missing tags, but we are also creating an elastic ip if there isn't
one key created.  And we're also importing a public key.  (We really should relaunch the
instances because the image has changes, but the tool doesn't yet detect an out of date AMI,
though that is helping us here).  You shouldn't see a lot else - if you do, stop and open
an issue before proceeding!

You will note that there will be some warnings, because we don't have the CA key
(https://github.com/kubernetes/kubernetes/issues/23264)  This means that we can't generate any
new certificates.  You'll also (probably)
see that you aren't using an elastic IP, because we will `allocate-address` a new one
in the above output.

e.g.
```
ELASTICIP_1=`aws ec2 allocate-address --domain vpc --query AllocationId`
ELASTICIP_1_PUBLICIP=`aws ec2 describe-addresses --allocation-ids ${ELASTICIP_1} --query Addresses[].PublicIp`
```

Unfortunately the new Elastic IP will be a new IP address, and this will require a new certificate.

Presuming that's right, we'll need to recreate all our keys, sadly.

Backup the existing keys & config, and then delete the keys:
```
cp -r upgrade11/ upgrade11.backup/
rm -rf upgrade11/pki/
```

Now comes the moment of truth.


# Shut down your master (output as INSTANCE_1 above)
aws ec2 --region ${REGION} terminate-instances --instance-id i-8d247c4a

# Reconfigure your cluster:
kope create cluster -i ${ssh_key} -d upgrade11/ -r release-${release}/kubernetes/ --logtostderr -t direct

# Now once again list your clusters; if you weren't using an elastic IP previously, a new one will have been allocated
kope discover clusters --region ${REGION}

MASTER_IP=52.34.179.39 # or whatever it shows


# Now, if the IP address changed, this means your kubecfg is now pointing to an invalid IP
# The kope create kubecfg will update with a new configuration:
kope create kubecfg -i ${ssh_key} --master ${MASTER_IP}


# If you now try `kubectl get nodes`, you should connect but the certificates are still not fully updated.
> kubectl get nodes
Unable to connect to the server: x509: certificate is valid for 52.27.60.160, 10.0.0.1, not 52.34.179.39

# This is because we sent the correct certificates, but the script kept the old certificates
# (as found on the persistent disk)

PROBABLY BETTER JUST TO AUTOMATE THIS (MAYBE AS PART OF K8S DEPLOY)

ssh -i ${ssh_key} admin@${MASTER_IP} mkdir /tmp/ca
scp -i ${ssh_key} upgrade11/pki/ca.crt  admin@${MASTER_IP}:/tmp/ca/ca.crt
scp -i ${ssh_key} upgrade11/pki/issued/cn\=kubernetes-master.crt  admin@${MASTER_IP}:/tmp/ca/server.cert
scp -i ${ssh_key} upgrade11/pki/private/cn\=kubernetes-master.key admin@${MASTER_IP}:/tmp/ca/server.key
scp -i ${ssh_key} upgrade11/pki/issued/cn\=kubecfg.crt  admin@${MASTER_IP}:/tmp/ca/kubecfg.crt
scp -i ${ssh_key} upgrade11/pki/private/cn\=kubecfg.key admin@${MASTER_IP}:/tmp/ca/kubecfg.key
ssh -i ${ssh_key} admin@${MASTER_IP} sudo cp /tmp/ca/* /mnt/master-pd/srv/kubernetes/
ssh -i ${ssh_key} admin@${MASTER_IP} sudo chown root:root /mnt/master-pd/srv/kubernetes/ca.crt /mnt/master-pd/srv/kubernetes/server.* /mnt/master-pd/srv/kubernetes/kubecfg.*
ssh -i ${ssh_key} admin@${MASTER_IP} sudo chmod 600  /mnt/master-pd/srv/kubernetes/ca.crt /mnt/master-pd/srv/kubernetes/server.* /mnt/master-pd/srv/kubernetes/kubecfg.*
ssh -i ${ssh_key} admin@${MASTER_IP} sudo rm -rf /tmp/ca
ssh -i ${ssh_key} admin@${MASTER_IP} sudo systemctl restart docker

# And then re-update the configuration:
kope create kubecfg -i ${ssh_key} --master ${MASTER_IP}


# To get your nodes to the latest version, you will need to restart them
# You can list them using:
kubectl get nodes -ojson  | jq -r .items[].spec.externalID

# Shut them down using:

kubectl get nodes -ojson  | jq -r .items[].spec.externalID | xargs aws ec2 terminate-instances --instance-ids 

# It is then fun to watch your nodes get removed and then (after a few minutes) come back
watch kubectl get nodes
