package gceunits

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
	"github.com/kopeio/kope/pkg/fi/gce"
	"google.golang.org/api/compute/v1"
	"time"
)

type ManagedInstanceGroup struct {
	fi.SimpleUnit

	Name             *string
	Zone             *string
	BaseInstanceName *string
	InstanceTemplate *InstanceTemplate
	TargetSize       *int64
}

func (s *ManagedInstanceGroup) Key() string {
	return *s.Name
}

func (s *ManagedInstanceGroup) GetID() *string {
	return s.Name
}

func (e *ManagedInstanceGroup) find(c *fi.RunContext) (*ManagedInstanceGroup, error) {
	cloud := c.Cloud().(*gce.GCECloud)

	r, err := cloud.Compute.InstanceGroupManagers.Get(cloud.Project, *e.Zone, *e.Name).Do()
	if err != nil {
		if gce.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error listing ManagedInstanceGroups: %v", err)
	}

	actual := &ManagedInstanceGroup{}
	actual.Name = &r.Name
	actual.Zone = fi.String(lastComponent(r.Zone))
	actual.BaseInstanceName = &r.BaseInstanceName
	actual.TargetSize = &r.TargetSize
	actual.InstanceTemplate = &InstanceTemplate{Name: fi.String(lastComponent(r.InstanceTemplate))}

	return actual, nil
}

func (e *ManagedInstanceGroup) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	changes := &ManagedInstanceGroup{}
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

func (s *ManagedInstanceGroup) checkChanges(a, e, changes *ManagedInstanceGroup) error {
	if a != nil {
	}
	return nil
}

func BuildInstanceTemplateURL(project, name string) string {
	return fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/instanceTemplates/%s", project, name)
}

func (_ *ManagedInstanceGroup) RenderGCE(t *gce.GCEAPITarget, a, e, changes *ManagedInstanceGroup) error {
	project := t.Cloud.Project

	i := &compute.InstanceGroupManager{
		Name:             *e.Name,
		Zone:             *e.Zone,
		BaseInstanceName: *e.BaseInstanceName,
		TargetSize:       *e.TargetSize,
		InstanceTemplate: BuildInstanceTemplateURL(project, *e.InstanceTemplate.Name),
	}

	if a == nil {
		for {
			_, err := t.Cloud.Compute.InstanceGroupManagers.Insert(t.Cloud.Project, *e.Zone, i).Do()
			if err != nil {
				if gce.IsNotReady(err) {
					glog.Infof("Found resourceNotReady error - sleeping before retry: %v", err)
					time.Sleep(5 * time.Second)
					continue
				}
				return fmt.Errorf("error creating ManagedInstanceGroup: %v", err)
			} else {
				break
			}
		}
	} else {
		return fmt.Errorf("Cannot apply changes to ManagedInstanceGroup: %v", changes)
	}

	return nil
}

//func (_*ManagedInstanceGroup) RenderBash(t *fi.BashTarget, a, e, changes *ManagedInstanceGroup) error {
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
//		//gcloud compute ManagedInstanceGroups create "${MASTER_NAME}" \
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
//		args := []string{"create-ManagedInstanceGroup", "--cidr-block", *e.CIDR, "--vpc-id", vpcID, "--query", "ManagedInstanceGroup.ManagedInstanceGroupId"}
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
