package awsunits

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
	"strconv"
)

type SecurityGroupIngress struct {
	fi.SimpleUnit

	SecurityGroup *SecurityGroup
	CIDR          *string
	Protocol      *string
	FromPort      *int64
	ToPort        *int64
	SourceGroup   *SecurityGroup
}

func (s *SecurityGroupIngress) Key() string {
	key := s.SecurityGroup.Key()
	if s.Protocol != nil {
		key += "-" + *s.Protocol
	}
	if s.FromPort != nil {
		key += "-" + strconv.FormatInt(*s.FromPort, 10)
	}
	if s.ToPort != nil {
		key += "-" + strconv.FormatInt(*s.ToPort, 10)
	}
	if s.CIDR != nil {
		key += "-" + *s.CIDR
	}
	if s.SourceGroup != nil {
		key += "-" + s.SourceGroup.Key()
	}
	return key
}

func (e *SecurityGroupIngress) find(c *fi.RunContext) (*SecurityGroupIngress, error) {
	cloud := c.Cloud().(*fi.AWSCloud)

	if e.SecurityGroup == nil || e.SecurityGroup.ID == nil {
		return nil, nil
	}

	request := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			fi.NewEC2Filter("group-id", *e.SecurityGroup.ID),
		},
	}

	response, err := cloud.EC2.DescribeSecurityGroups(request)
	if err != nil {
		return nil, fmt.Errorf("error listing SecurityGroup: %v", err)
	}

	if response == nil || len(response.SecurityGroups) == 0 {
		return nil, nil
	}

	if len(response.SecurityGroups) != 1 {
		glog.Fatalf("found multiple security groups for id=%s", *e.SecurityGroup.ID)
	}
	sg := response.SecurityGroups[0]
	//glog.V(2).Info("found existing security group")

	var foundRule *ec2.IpPermission

	matchProtocol := "-1" // Wildcard
	if e.Protocol != nil {
		matchProtocol = *e.Protocol
	}

	for _, rule := range sg.IpPermissions {
		if aws.Int64Value(rule.FromPort) != aws.Int64Value(e.FromPort) {
			continue
		}
		if aws.Int64Value(rule.ToPort) != aws.Int64Value(e.ToPort) {
			continue
		}
		if aws.StringValue(rule.IpProtocol) != matchProtocol {
			continue
		}
		if e.CIDR != nil {
			// TODO: Only if len 1?
			match := false
			for _, ipRange := range rule.IpRanges {
				if aws.StringValue(ipRange.CidrIp) == *e.CIDR {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		if e.SourceGroup != nil {
			// TODO: Only if len 1?
			match := false
			for _, spec := range rule.UserIdGroupPairs {
				if aws.StringValue(spec.GroupId) == *e.SourceGroup.ID {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		foundRule = rule
		break
	}

	if foundRule != nil {
		actual := &SecurityGroupIngress{}
		actual.SecurityGroup = &SecurityGroup{ID: e.SecurityGroup.ID}
		actual.FromPort = foundRule.FromPort
		actual.ToPort = foundRule.ToPort
		actual.Protocol = foundRule.IpProtocol
		if aws.StringValue(actual.Protocol) == "-1" {
			actual.Protocol = nil
		}
		if e.CIDR != nil {
			actual.CIDR = e.CIDR
		}
		if e.SourceGroup != nil {
			actual.SourceGroup = &SecurityGroup{ID: e.SourceGroup.ID}
		}
		return actual, nil
	}

	return nil, nil
}

func (e *SecurityGroupIngress) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	changes := &SecurityGroupIngress{}
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

func (s *SecurityGroupIngress) checkChanges(a, e, changes *SecurityGroupIngress) error {
	if a == nil {
		if e.SecurityGroup == nil {
			return fi.MissingValueError("Must specify SecurityGroup when creating SecurityGroupIngress")
		}
	}
	return nil
}

func (_ *SecurityGroupIngress) RenderAWS(t *fi.AWSAPITarget, a, e, changes *SecurityGroupIngress) error {
	if a == nil {
		request := &ec2.AuthorizeSecurityGroupIngressInput{}
		request.GroupId = e.SecurityGroup.ID
		request.CidrIp = e.CIDR
		request.IpProtocol = e.Protocol
		request.FromPort = e.FromPort
		request.ToPort = e.ToPort
		if e.SourceGroup != nil {
			request.IpPermissions = []*ec2.IpPermission{
				{
					UserIdGroupPairs: []*ec2.UserIdGroupPair{
						{
							GroupId: e.SourceGroup.ID,
						},
					},
				},
			}
		}
		_, err := t.Cloud.EC2.AuthorizeSecurityGroupIngress(request)
		if err != nil {
			return fmt.Errorf("error creating SecurityGroupIngress: %v", err)
		}
	}

	return nil //return output.AddAWSTags(cloud.Tags(), v, "vpc")
}

func (_ *SecurityGroupIngress) RenderBash(t *fi.BashTarget, a, e, changes *SecurityGroupIngress) error {
	if a == nil {
		glog.V(2).Infof("Creating SecurityGroupIngress")

		args := []string{"authorize-security-group-ingress"}
		args = append(args, "--group-id", t.ReadVar(e.SecurityGroup))

		if e.Protocol != nil {
			args = append(args, "--protocol", *e.Protocol)
		} else {
			args = append(args, "--protocol", "all")
		}
		fromPort := aws.Int64Value(e.FromPort)
		toPort := aws.Int64Value(e.ToPort)
		if fromPort != 0 || toPort != 0 {
			if fromPort == toPort {
				args = append(args, "--port", fmt.Sprintf("%d", fromPort))
			} else {
				args = append(args, "--port", fmt.Sprintf("%d-%d", fromPort, toPort))
			}
		}
		if e.CIDR != nil {
			args = append(args, "--cidr", *e.CIDR)
		}

		if e.SourceGroup != nil {
			args = append(args, "--source-group", t.ReadVar(e.SourceGroup))
		}

		t.AddEC2Command(args...)
	}

	return nil
}

func (s *SecurityGroupIngress) String() string {
	return fmt.Sprintf("SecurityGroupIngress (Port=%d-%d)", s.FromPort, s.ToPort)
}
