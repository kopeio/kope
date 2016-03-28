package awsunits

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/glog"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/kopeio/kope/pkg/fi"
)

type ElasticIP struct {
	fi.SimpleUnit

	ID            *string
	PublicIP      *string

	// Because ElasticIPs don't supporting tagging (sadly), we instead tag on
	// a different resource
	TagUsingKey   *string
	TagOnResource fi.HasID
}

//var _ units.HasAddress = &ElasticIP{}

func (e *ElasticIP) GetID() *string {
	return e.ID
}

func (e *ElasticIP) Key() string {
	return *e.TagUsingKey
}

func (e *ElasticIP) String() string {
	return fi.JsonString(e)
}

func (e*ElasticIP) FindAddress(c fi.Cloud) (*string, error) {
	actual, err := e.find(c.(*fi.AWSCloud))
	if err != nil {
		return nil, fmt.Errorf("error querying for Master PublicIP: %v", err)
	}
	if actual == nil {
		return nil, nil
	}
	return actual.PublicIP, nil
}

func (e *ElasticIP) find(cloud *fi.AWSCloud) (*ElasticIP, error) {
	publicIP := e.PublicIP
	allocationID := e.ID

	// Find via tag on foreign resource
	if allocationID == nil && publicIP == nil && e.TagUsingKey != nil &&e.TagOnResource != nil &&  e.TagOnResource.GetID() != nil {
		var filters []*ec2.Filter
		filters = append(filters, fi.NewEC2Filter("key", *e.TagUsingKey))
		filters = append(filters, fi.NewEC2Filter("resource-id", *e.TagOnResource.GetID()))

		request := &ec2.DescribeTagsInput{
			Filters: filters,
		}

		response, err := cloud.EC2.DescribeTags(request)
		if err != nil {
			return nil, fmt.Errorf("error listing tags: %v", err)
		}

		if response == nil || len(response.Tags) == 0 {
			return nil, nil
		}

		if len(response.Tags) != 1 {
			return nil, fmt.Errorf("found multiple tags for: %v", e.Key())
		}
		t := response.Tags[0]
		publicIP = t.Value
		glog.V(2).Infof("Found public IP via tag: %v", *publicIP)
	}

	if publicIP != nil || allocationID != nil {
		request := &ec2.DescribeAddressesInput{}
		if allocationID != nil {
			request.AllocationIds = []*string{allocationID}
		} else if publicIP != nil {
			request.Filters = []*ec2.Filter{fi.NewEC2Filter("public-ip", *publicIP) }
		}

		response, err := cloud.EC2.DescribeAddresses(request)
		if err != nil {
			return nil, fmt.Errorf("error listing ElasticIPs: %v", err)
		}

		if response == nil || len(response.Addresses) == 0 {
			return nil, nil
		}

		if len(response.Addresses) != 1 {
			return nil, fmt.Errorf("found multiple ElasticIPs for: %v", e.Key())
		}
		a := response.Addresses[0]
		actual := &ElasticIP{}
		actual.ID = a.AllocationId
		actual.PublicIP = a.PublicIp
		return actual, nil
	}

	return nil, nil
}

func (e *ElasticIP) Run(c *fi.RunContext) error {
	a, err := e.find(c.Cloud().(*fi.AWSCloud))
	if err != nil {
		return err
	}

	if a != nil {
		if e.ID == nil {
			e.ID = a.ID
		}
		if e.PublicIP == nil {
			e.PublicIP = a.PublicIP
		}
	}

	changes := &ElasticIP{}
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

func (s *ElasticIP) checkChanges(a, e, changes *ElasticIP) error {
	return nil
}

func (_*ElasticIP) RenderAWS(t *fi.AWSAPITarget, a, e, changes *ElasticIP) error {
	var publicIP *string
	var tagOnResourceID *string
	if e.TagOnResource != nil {
		tagOnResourceID = e.TagOnResource.GetID()
	}

	if a == nil {
		if tagOnResourceID == nil || e.TagUsingKey == nil {
			return fmt.Errorf("cannot create ElasticIP without TagOnResource being set (would leak)")
		}
		glog.V(2).Infof("Creating ElasticIP for VPC")

		request := &ec2.AllocateAddressInput{}
		request.Domain = aws.String(ec2.DomainTypeVpc)

		response, err := t.Cloud.EC2.AllocateAddress(request)
		if err != nil {
			return fmt.Errorf("error creating ElasticIP: %v", err)
		}

		e.ID = response.AllocationId
		e.PublicIP = response.PublicIp
		publicIP = response.PublicIp
	} else {
		publicIP = a.PublicIP
	}

	if publicIP != nil && e.TagUsingKey != nil && tagOnResourceID != nil {
		tags := map[string]string{
			*e.TagUsingKey: *publicIP,
		}
		err := t.AddAWSTags(*tagOnResourceID, tags)
		if err != nil {
			return fmt.Errorf("error adding tags to resource for ElasticIP: %v", err)
		}
	}
	return nil
}

func (_*ElasticIP) RenderBash(t *fi.BashTarget, a, e, changes *ElasticIP) error {
	t.CreateVar(e)
	if a == nil {
		if e.TagOnResource == nil || e.TagUsingKey == nil {
			return fmt.Errorf("cannot create ElasticIP without TagOnResource being set (would leak)")
		}

		glog.V(2).Infof("Creating ElasticIP for VPC")

		t.AddEC2Command("allocate-address",
			"--domain", "vpc",
			"--query", "AllocationId").AssignTo(e)
	} else {
		t.AddAssignment(e, fi.StringValue(a.ID))
	}

	if e.TagOnResource != nil && e.TagUsingKey != nil {
		tagOnUnit, ok := e.TagOnResource.(fi.Unit)
		if !ok {
			return fmt.Errorf("Expected TagOnResource to be a Unit: %T", e.TagOnResource)
		}

		if a != nil && a.PublicIP != nil {
			tags := map[string]string{
				*e.TagUsingKey:*a.PublicIP,
			}
			err := t.AddAWSTags(tagOnUnit, tags)
			if err != nil {
				return fmt.Errorf("error adding tags to resource for ElasticIP: %v", err)
			}
		} else {
			t.AddEC2Command("describe-addresses", "--allocation-ids", t.ReadVar(e), "--query", "Addresses[].PublicIp").AssignToSuffixedVariable(e, "_PUBLICIP")
			t.AddBashCommand("add-tag", t.ReadVar(tagOnUnit), *e.TagUsingKey, t.ReadVarWithSuffix(e, "_PUBLICIP"))
		}
	}

	return nil
}
