package awsunits

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
	"github.com/aws/aws-sdk-go/aws"
)

type InternetGatewayAttachment struct {
	fi.SimpleUnit

	VPC             *VPC
	InternetGateway *InternetGateway
}

func (s *InternetGatewayAttachment) Key() string {
	return s.VPC.Key() + "-" + s.InternetGateway.Key()
}

func (e *InternetGatewayAttachment) find(c *fi.RunContext) (*InternetGatewayAttachment, error) {
	vpcID := e.VPC.ID
	if vpcID == nil {
		return nil, nil
	}
	if e.InternetGateway.ID == nil {
		return nil, nil
	}

	cloud := c.Cloud().(*fi.AWSCloud)

	request := &ec2.DescribeInternetGatewaysInput{
		InternetGatewayIds: []*string{e.InternetGateway.ID},
	}

	response, err := cloud.EC2.DescribeInternetGateways(request)
	if err != nil {
		return nil, fmt.Errorf("error listing DescribeInternetGateways: %v", err)
	}
	if response == nil || len(response.InternetGateways) == 0 {
		return nil, nil
	}

	if len(response.InternetGateways) != 1 {
		for _, ig := range response.InternetGateways {
			glog.Infof("gateway: %v", DebugPrint(ig))
		}
		glog.Fatalf("found multiple InternetGatewayAttachments matching ID")
	}
	igw := response.InternetGateways[0]
	for _, attachment := range igw.Attachments {
		if aws.StringValue(attachment.VpcId) == *vpcID {
			actual := &InternetGatewayAttachment{
				VPC: &VPC{ID:vpcID},
				InternetGateway: &InternetGateway{ID:e.InternetGateway.ID},
			}
			glog.V(2).Infof("found matching InternetGatewayAttachment")
			return actual, nil
		}
	}

	return nil, nil
}

func (e *InternetGatewayAttachment) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	changes := &InternetGatewayAttachment{}
	changed := BuildChanges(a, e, changes)
	if !changed {
		return nil
	}

	err = e.checkChanges(a, e, changes)
	if err != nil {
		return err
	}

	return c.Render(a, e, changes)
}

func (s *InternetGatewayAttachment) checkChanges(a, e, changes *InternetGatewayAttachment) error {
	if a != nil {
		// TODO: I think we can change it; we just detach & attach
		if changes.VPC != nil {
			return InvalidChangeError("Cannot change InternetGatewayAttachment VPC", changes.VPC.ID, e.VPC.ID)
		}
	}
	return nil
}

func (_*InternetGatewayAttachment) RenderAWS(t *fi.AWSAPITarget, a, e, changes *InternetGatewayAttachment) error {
	if a == nil {
		glog.V(2).Infof("Creating InternetGatewayAttachment")

		attachRequest := &ec2.AttachInternetGatewayInput{
			VpcId: e.VPC.ID,
			InternetGatewayId: e.InternetGateway.ID,
		}

		_, err := t.Cloud.EC2.AttachInternetGateway(attachRequest)
		if err != nil {
			return fmt.Errorf("error attaching InternetGatewayAttachment: %v", err)
		}
	}

	return nil // No tags
}

func (_*InternetGatewayAttachment) RenderBash(t *fi.BashTarget, a, e, changes *InternetGatewayAttachment) error {
	//t.CreateVar(e)
	if a == nil {
		vpcID := t.ReadVar(e.VPC)
		igwID := t.ReadVar(e.InternetGateway)

		t.AddEC2Command("attach-internet-gateway", "--internet-gateway-id", igwID, "--vpc-id", vpcID)
	} else {
		//t.AddAssignment(e, StringValue(a.ID))
	}

	return nil // No tags
}
