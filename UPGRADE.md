
To upgrade from 1.1 to 1.2

go install github.com/kopeio/kope/installer/kutil

# Set the REGION env var to the region where your cluster is
REGION=us-west-2

# List all the clusters in ${REGION}
> kutil discover clusters --region ${REGION}
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

# Extract the configuration and keys from your existing cluster
kutil export cluster --master ${MASTER_IP} -i ${ssh_key} --logtostderr --node ${NODE_IP} --dest upgrade11/

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
ssh_key=~/.ssh/kube_aws_rsa
kutil create cluster -i ${ssh_key} -d upgrade11/ -r release-${release}/kubernetes/ --logtostderr --s3-bucket ${BUCKET} -t bash

You should see something like this:

```
#!/bin/bash
set -ex

. ./helpers

export AWS_DEFAULT_REGION="us-west-2"
export AWS_DEFAULT_OUTPUT="text"
PERSISTENTVOLUME_1="vol-f8adc40e"
ELASTICIP_1=`aws ec2 allocate-address --domain vpc --query AllocationId`
ELASTICIP_1_PUBLICIP=`aws ec2 describe-addresses --allocation-ids ${ELASTICIP_1} --query Addresses[].PublicIp`
add-tag ${PERSISTENTVOLUME_1} kubernetes.io/master-ip ${ELASTICIP_1_PUBLICIP}
IAMROLE_1="AROAILHQRUIC4GMMJFPZS"
IAMROLE_2="AROAJQYAOWIFBGC7M4J64"
aws ec2 import-key-pair --key-name kubernetes-kubernetes --public-key-material file://resources/StringResource_1
VPC_1="vpc-a6d017c2"
add-tag ${VPC_1} "Name" "kubernetes-kubernetes"
DHCPOPTIONS_1="dopt-54021936"
add-tag ${DHCPOPTIONS_1} "Name" "kubernetes-kubernetes"
add-tag ${DHCPOPTIONS_1} "KubernetesCluster" "kubernetes"
SUBNET_1="subnet-7fac621b"
add-tag ${SUBNET_1} "Name" "kubernetes-kubernetes"
INTERNETGATEWAY_1="igw-91c2e7f4"
add-tag ${INTERNETGATEWAY_1} "Name" "kubernetes-kubernetes"
add-tag ${INTERNETGATEWAY_1} "KubernetesCluster" "kubernetes"
ROUTETABLE_1="rtb-327ed956"
add-tag ${ROUTETABLE_1} "Name" "kubernetes-kubernetes"
ROUTETABLEASSOCIATION_1="rtbassoc-249ae840"
SECURITYGROUP_1="sg-d4debeb3"
add-tag ${SECURITYGROUP_1} "Name" "kubernetes-master-kubernetes"
SECURITYGROUP_2="sg-dfdebeb8"
add-tag ${SECURITYGROUP_2} "Name" "kubernetes-minion-kubernetes"
INSTANCE_1="i-8d247c4a"
add-tag ${INSTANCE_1} "Role" "master"
wait-for-instance-state ${INSTANCE_1} running
aws ec2 associate-address --allocation-id ${ELASTICIP_1} --instance-id ${INSTANCE_1}

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

Unfortunately the new Elastic IP will require a new certificate.

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
kutil create cluster -i ${ssh_key} -d upgrade11/ -r release-${release}/kubernetes/ --logtostderr --s3-bucket ${BUCKET} -t direct

# Now once again list your clusters; if you weren't using an elastic IP previously, a new one will have been allocated
kutil discover clusters --region ${REGION}

MASTER_IP=52.34.179.39 # or whatever it shows


# Now, if the IP address changed, this means your kubecfg is now pointing to an invalid IP
# The kutil create kubecfg will update with a new configuration:
kutil create kubecfg -i ${ssh_key} --master ${MASTER_IP}


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
kutil create kubecfg -i ${ssh_key} --master ${MASTER_IP}



NEED TO DOCUMENT RESTARTIN OF ASG

