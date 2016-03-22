package awsunits

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
	"github.com/aws/aws-sdk-go/aws"
)

type DHCPOptions struct {
	fi.SimpleUnit

	ID                *string
	Name              *string
	DomainName        *string
	DomainNameServers *string
}

func (s *DHCPOptions) GetID() *string {
	return s.ID
}

func (s *DHCPOptions) Key() string {
	return *s.Name
}

func (s *DHCPOptions) String() string {
	return JsonString(s)
}

func (e *DHCPOptions) find(c *fi.RunContext) (*DHCPOptions, error) {
	cloud := c.Cloud().(*fi.AWSCloud)

	request := &ec2.DescribeDhcpOptionsInput{}
	if e.ID != nil {
		request.DhcpOptionsIds = []*string{e.ID }
	} else {
		request.Filters = cloud.BuildFilters(e.Name)
	}

	response, err := cloud.EC2.DescribeDhcpOptions(request)
	if err != nil {
		return nil, fmt.Errorf("error listing DHCPOptions: %v", err)
	}

	if response == nil || len(response.DhcpOptions) == 0 {
		return nil, nil
	}

	if len(response.DhcpOptions) != 1 {
		return nil, fmt.Errorf("found multiple DhcpOptions with name: %s", *e.Name)
	}
	glog.V(2).Info("found existing DhcpOptions")
	o := response.DhcpOptions[0]
	actual := &DHCPOptions{}
	actual.ID = o.DhcpOptionsId
	for _, s := range o.DhcpConfigurations {
		k := aws.StringValue(s.Key)
		v := ""
		for _, av := range s.Values {
			if v != "" {
				v = v + ","
			}
			v = v + *av.Value
		}
		switch (k) {
		case "domain-name":
			actual.DomainName = &v
		case "domain-name-servers":
			actual.DomainNameServers = &v
		default:
			glog.Infof("Skipping over DHCPOption with key=%q value=%q", k, v)
		}
	}
	actual.Name = e.Name
	return actual, nil
}

func (e *DHCPOptions) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	if a != nil {
		if e.ID == nil {
			e.ID = a.ID
		}
	}

	changes := &DHCPOptions{}
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

func (s *DHCPOptions) checkChanges(a, e, changes *DHCPOptions) error {
	if a == nil {
		if e.Name == nil {
			return MissingValueError("Name must be specified when creating a DHCPOptions")
		}
	}
	if a != nil {
		if changes.ID != nil {
			return InvalidChangeError("Cannot change DHCPOptions ID", changes.ID, e.ID)
		}
	}
	return nil
}

func (_*DHCPOptions) RenderAWS(t *fi.AWSAPITarget, a, e, changes *DHCPOptions) error {
	if a == nil {
		glog.V(2).Infof("Creating DHCPOptions with Name:%q", *e.Name)

		request := &ec2.CreateDhcpOptionsInput{}
		if e.DomainNameServers != nil {
			o := &ec2.NewDhcpConfiguration{
				Key: aws.String("domain-name-servers"),
				Values: []*string{e.DomainNameServers},
			}
			request.DhcpConfigurations = append(request.DhcpConfigurations, o)
		}
		if e.DomainName != nil {
			o := &ec2.NewDhcpConfiguration{
				Key: aws.String("domain-name"),
				Values: []*string{e.DomainName},
			}
			request.DhcpConfigurations = append(request.DhcpConfigurations, o)
		}

		response, err := t.Cloud.EC2.CreateDhcpOptions(request)
		if err != nil {
			return fmt.Errorf("error creating DHCPOptions: %v", err)
		}

		e.ID = response.DhcpOptions.DhcpOptionsId
	}

	return t.AddAWSTags(*e.ID, t.Cloud.BuildTags(e.Name))
}

func (_*DHCPOptions) RenderBash(t *fi.BashTarget, a, e, changes *DHCPOptions) error {
	t.CreateVar(e)
	if a == nil {
		glog.V(2).Infof("Creating DHCPOptions with Name:%q", *e.Name)

		args := []string{"create-dhcp-options", "--dhcp-configuration"}
		if e.DomainName != nil {
			args = append(args, "Key=%s,Values=%s", "domain-name", *e.DomainName)
		}
		if e.DomainNameServers != nil {
			args = append(args, "Key=%s,Values=%s", "domain-name-servers", *e.DomainNameServers)
		}
		args = append(args, "--query", "DhcpOptions.DhcpOptionsId")
		t.AddEC2Command(args...).AssignTo(e)
	} else {
		t.AddAssignment(e, StringValue(a.ID))
	}

	return t.AddAWSTags(e, t.Cloud.BuildTags(e.Name))
}
