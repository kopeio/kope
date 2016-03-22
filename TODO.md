* Mount disk on master
* Add route on master
* Bring kube-config down locally

* Write config in new unified format
* Delete-cluster functionality
* Smarter comparisons
* A second backend as a proof-of-concept (CloudFormation?)
* Optimize s3 object-acl



To verify:

DHCP option set

Fix SSH key

P0

Use default bucket

Export config from a 1.2 cluster

Export config from a 1.1 cluster

Follow volume tag to get IP address to delete

All k8s variables should be pointers

Always copy keys & certs

Restart ASG instances

Update master when AMI / UserData changes


P1

Delete the other types: vpc, route-table, internet-gateway

Include launch configuration in ASG tags?


=======================================================

P2

Export config from a 1.0 cluster?


Need to add the elastic IP to the list of SANs... tricky interdependencies...

Should we have a pre-gen stage?  Where we set up the CA?
Another where we upload our resources?

Should we tag our resources on S3 e.g. with SHA1?

We should probably be able to change the CA key etc

We should regenerate the cert when we have a different set of SANs
