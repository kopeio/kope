package gce

import "github.com/kopeio/kope/pkg/fi"

type GCEAPITarget struct {
	Cloud     *GCECloud
	filestore fi.FileStore
}

var _ fi.Target = &GCEAPITarget{}

func NewGCEAPITarget(cloud*GCECloud, filestore fi.FileStore) *GCEAPITarget {
	return &GCEAPITarget{
		Cloud: cloud,
		filestore: filestore,
	}
}

func (t *GCEAPITarget) FileStore() fi.FileStore {
	return t.filestore
}


//func (t *GCEAPITarget) AddAWSTags(id string, expected map[string]string) error {
//	actual, err := t.Cloud.GetTags(id)
//	if err != nil {
//		return fmt.Errorf("unexpected error fetching tags for resource: %v", err)
//	}
//
//	missing := map[string]string{}
//	for k, v := range expected {
//		actualValue, found := actual[k]
//		if found && actualValue == v {
//			continue
//		}
//		missing[k] = v
//	}
//
//	if len(missing) != 0 {
//		request := &ec2.CreateTagsInput{}
//		request.Resources = []*string{&id}
//		for k, v := range missing {
//			request.Tags = append(request.Tags, &ec2.Tag{
//				Key:   aws.String(k),
//				Value: aws.String(v),
//			})
//		}
//
//		_, err := t.Cloud.EC2.CreateTags(request)
//		if err != nil {
//			return fmt.Errorf("error adding tags to resource %q: %v", id, err)
//		}
//	}
//
//	return nil
//}

//func (t *GCEAPITarget) WaitForInstanceRunning(instanceID string) (error) {
//	attempt := 0
//	for {
//		instance, err := t.Cloud.DescribeInstance(instanceID)
//		if err != nil {
//			return fmt.Errorf("error while waiting for instance to be running: %v", err)
//		}
//
//		if instance == nil {
//			// TODO: Wait if we _just_ created the instance?
//			return fmt.Errorf("instance not found while waiting for instance to be running")
//		}
//
//		state := "?"
//		if instance.State != nil {
//			state = aws.StringValue(instance.State.Name)
//		}
//		glog.V(4).Infof("state of instance %q is %q", instanceID, state)
//		if state == "running" {
//			return nil
//		}
//
//		time.Sleep(10 * time.Second)
//		attempt++
//		if attempt > 30 {
//			return fmt.Errorf("timeout waiting for instance %q to be running, state was %q", instanceID, state)
//		}
//	}
//}

