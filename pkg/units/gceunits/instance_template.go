package gceunits

import (
	"fmt"

	"github.com/kopeio/kope/pkg/fi"
	"google.golang.org/api/compute/v1"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi/gce"
)

type InstanceTemplate struct {
	fi.SimpleUnit

	Name           *string
	Network        *Network
	Tags           []string
	Preemptible    *bool

	BootDiskImage  *string
	BootDiskSizeGB *int64
	BootDiskType   *string

	CanIPForward   *bool
	Subnet         *Subnet

	Scopes         []string

	Metadata       map[string]fi.Resource
	MachineType    *string
}

func (s *InstanceTemplate) Key() string {
	return *s.Name
}

func (s *InstanceTemplate) GetID() *string {
	return s.Name
}

func (e *InstanceTemplate) find(c *fi.RunContext) (*InstanceTemplate, error) {
	cloud := c.Cloud().(*gce.GCECloud)

	r, err := cloud.Compute.InstanceTemplates.Get(cloud.Project, *e.Name).Do()
	if err != nil {
		if gce.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error listing InstanceTemplates: %v", err)
	}

	actual := &InstanceTemplate{}
	actual.Name = &r.Name

	p := r.Properties

	for _, tag := range p.Tags.Items {
		actual.Tags = append(actual.Tags, tag)
	}
	actual.MachineType = fi.String(lastComponent(p.MachineType))
	actual.CanIPForward = &p.CanIpForward

	bootDiskImage, err := ShortenImageURL(p.Disks[0].InitializeParams.SourceImage)
	if err != nil {
		return nil, fmt.Errorf("error parsing source image URL: %v", err)
	}
	actual.BootDiskImage = fi.String(bootDiskImage)
	actual.BootDiskType = &p.Disks[0].InitializeParams.DiskType
	actual.BootDiskSizeGB = &p.Disks[0].InitializeParams.DiskSizeGb

	if p.Scheduling != nil {
		actual.Preemptible = &p.Scheduling.Preemptible
	}
	if len(p.NetworkInterfaces) != 0 {
		ni := p.NetworkInterfaces[0]
		actual.Network = &Network{Name: fi.String(lastComponent(ni.Network))}
	}

	for _, serviceAccount := range p.ServiceAccounts {
		for _, scope := range serviceAccount.Scopes {
			actual.Scopes = append(actual.Scopes, scopeToShortForm(scope))
		}
	}

	//for i, disk := range p.Disks {
	//	if i == 0 {
	//		source := disk.Source
	//
	//		// TODO: Parse source URL instead of assuming same project/zone?
	//		name := lastComponent(source)
	//		d, err := cloud.Compute.Disks.Get(cloud.Project, *e.Zone, name).Do()
	//		if err != nil {
	//			if gce.IsNotFound(err) {
	//				return nil, fmt.Errorf("disk not found %q: %v", source, err)
	//			}
	//			return nil, fmt.Errorf("error querying for disk %q: %v", source, err)
	//		} else {
	//			imageURL, err := gce.ParseGoogleCloudURL(d.SourceImage)
	//			if err != nil {
	//				return nil, fmt.Errorf("unable to parse image URL: %q", d.SourceImage)
	//			}
	//			actual.Image = fi.String(imageURL.Project + "/" + imageURL.Name)
	//		}
	//	}
	//}

	if p.Metadata != nil {
		actual.Metadata = make(map[string]fi.Resource)
		for _, meta := range p.Metadata.Items {
			actual.Metadata[meta.Key] = fi.NewStringResource(*meta.Value)
		}
	}

	return actual, nil
}

func (e *InstanceTemplate) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	changes := &InstanceTemplate{}
	changed := fi.BuildChanges(a, e, changes)
	if !changed {
		return nil
	}

	err = e.checkChanges(a, e, changes)
	if err != nil {
		return err
	}

	return c.Render(a, e, changes)
}

func (s *InstanceTemplate) checkChanges(a, e, changes *InstanceTemplate) error {
	if a != nil {
	}
	return nil
}

func (_*InstanceTemplate) RenderGCE(t *gce.GCEAPITarget, a, e, changes *InstanceTemplate) error {
	project := t.Cloud.Project

	// TODO: This is similar to Instance...
	var scheduling *compute.Scheduling

	if fi.BoolValue(e.Preemptible) {
		scheduling = &compute.Scheduling{
			OnHostMaintenance: "TERMINATE",
			Preemptible: true,
		}
	} else {
		scheduling = &compute.Scheduling{
			AutomaticRestart: true,
			// TODO: Migrate or terminate?
			OnHostMaintenance: "MIGRATE",
			Preemptible: false,
		}
	}

	glog.Infof("We should be using NVME for GCE")

	var disks []*compute.AttachedDisk
	disks = append(disks, &compute.AttachedDisk{
		InitializeParams: &compute.AttachedDiskInitializeParams{
			SourceImage: BuildImageURL(project, *e.BootDiskImage),
			DiskSizeGb: *e.BootDiskSizeGB,
			DiskType: *e.BootDiskType,
		},
		Boot: true,
		DeviceName:"persistent-disks-0",
		Index:0,
		AutoDelete:true,
		Mode: "READ_WRITE",
		Type: "PERSISTENT",
	})

	var tags *compute.Tags
	if e.Tags != nil {
		tags = &compute.Tags{
			Items: e.Tags,
		}
	}

	var networkInterfaces []*compute.NetworkInterface
	ni := &compute.NetworkInterface{
		AccessConfigs: []*compute.AccessConfig{&compute.AccessConfig{
			//NatIP: *e.IPAddress.Address,
			Type: "ONE_TO_ONE_NAT",
		}},
		Network: e.Network.URL(),
	}
	if e.Subnet != nil {
		ni.Subnetwork = *e.Subnet.Name
	}
	networkInterfaces = append(networkInterfaces, ni)

	var serviceAccounts []*compute.ServiceAccount
	if e.Scopes != nil {
		var scopes []string
		for _, s := range e.Scopes {
			s = expandScopeAlias(s)

			scopes = append(scopes, s)
		}
		serviceAccounts = append(serviceAccounts, &compute.ServiceAccount{
			Email:"default",
			Scopes: scopes,
		})
	}

	var metadataItems []*compute.MetadataItems
	for key, r := range e.Metadata {
		v, err := fi.ResourceAsString(r)
		if err != nil {
			return fmt.Errorf("error rendering InstanceTemplate metadata %q: %v", key, err)
		}
		metadataItems = append(metadataItems, &compute.MetadataItems{
			Key: key,
			Value: fi.String(v),
		})
	}

	i := &compute.InstanceTemplate{
		Name: *e.Name,
		Properties: &compute.InstanceProperties{
			CanIpForward: *e.CanIPForward,

			Disks: disks,

			MachineType: *e.MachineType,

			Metadata: &compute.Metadata{
				Items: metadataItems,
			},

			NetworkInterfaces: networkInterfaces,

			Scheduling: scheduling,

			ServiceAccounts: serviceAccounts,

			Tags: tags,
		},
	}

	if a == nil {
		_, err := t.Cloud.Compute.InstanceTemplates.Insert(t.Cloud.Project, i).Do()
		if err != nil {
			return fmt.Errorf("error creating InstanceTemplate: %v", err)
		}
	} else {
		// TODO: Make error again
		glog.Errorf("Cannot apply changes to InstanceTemplate: %v", changes)
		//return fmt.Errorf("Cannot apply changes to InstanceTemplate: %v", changes)
	}

	return nil
}

//func (_*InstanceTemplate) RenderBash(t *fi.BashTarget, a, e, changes *InstanceTemplate) error {
//	t.CreateVar(e)
//	if a == nil {
//		local
//		address_opt = ""
//		//	[[ -n ${1:-} ]] && address_opt="--address ${1}"
//		//	local preemptible_master=""
//		//	if [[ "${PREEMPTIBLE_MASTER:-}" == "true" ]]; then
//		//preemptible_master="--preemptible --maintenance-policy TERMINATE"
//		//fi
//		//
//		//write-master-env
//		//gcloud compute InstanceTemplates create "${MASTER_NAME}" \
//		//${address_opt} \
//		//--project "${PROJECT}" \
//		//--zone "${ZONE}" \
//		//--machine-type "${MASTER_SIZE}" \
//		//--image-project="${MASTER_IMAGE_PROJECT}" \
//		//--image "${MASTER_IMAGE}" \
//		//--tags "${MASTER_TAG}" \
//		//--network "${NETWORK}" \
//		//--scopes "storage-ro,compute-rw,monitoring,logging-write" \
//		//--can-ip-forward \
//		//--metadata-from-file \
//		//"startup-script=${KUBE_ROOT}/cluster/gce/configure-vm.sh,kube-env=${KUBE_TEMP}/master-kube-env.yaml,cluster-name=${KUBE_TEMP}/cluster-name.txt" \
//		//--disk "name=${MASTER_NAME}-pd,device-name=master-pd,mode=rw,boot=no,auto-delete=no" \
//		//${preemptible_master}
//
//
//		args := []string{"create-InstanceTemplate", "--cidr-block", *e.CIDR, "--vpc-id", vpcID, "--query", "InstanceTemplate.InstanceTemplateId"}
//		if e.AvailabilityZone != nil {
//			args = append(args, "--availability-zone", *e.AvailabilityZone)
//		}
//
//		t.AddEC2Command(args...).AssignTo(e)
//	} else {
//		t.AddAssignment(e, StringValue(a.ID))
//	}
//
//	return t.AddAWSTags(e, t.Cloud.BuildTags(e.Name))
//}