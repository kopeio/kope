package gceunits

import (
	"fmt"

	"github.com/kopeio/kope/pkg/fi"
	"github.com/kopeio/kope/pkg/fi/gce"
	"google.golang.org/api/compute/v1"
	"strings"
)

type FirewallRule struct {
	fi.SimpleUnit

	Name         *string
	Network      *Network
	SourceTags   []string
	SourceRanges []string
	TargetTags   []string
	Allowed      []string
}

func (s *FirewallRule) Key() string {
	return *s.Name
}

func (s *FirewallRule) GetID() *string {
	return s.Name
}

func (e *FirewallRule) find(c *fi.RunContext) (*FirewallRule, error) {
	cloud := c.Cloud().(*gce.GCECloud)

	r, err := cloud.Compute.Firewalls.Get(cloud.Project, *e.Name).Do()
	if err != nil {
		if gce.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error listing FirewallRules: %v", err)
	}

	actual := &FirewallRule{}
	actual.Name = &r.Name
	actual.Network = &Network{Name: fi.String(lastComponent(r.Network))}
	actual.TargetTags = r.TargetTags
	actual.SourceRanges = r.SourceRanges
	actual.SourceTags = r.SourceTags
	for _, a := range r.Allowed {
		actual.Allowed = append(actual.Allowed, serializeFirewallAllowed(a))
	}

	return actual, nil
}

func (e *FirewallRule) Run(c *fi.RunContext) error {
	a, err := e.find(c)
	if err != nil {
		return err
	}

	changes := &FirewallRule{}
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

func (s *FirewallRule) checkChanges(a, e, changes *FirewallRule) error {
	if a != nil {
	}
	return nil
}

func parseFirewallAllowed(rule string) (*compute.FirewallAllowed, error) {
	o := &compute.FirewallAllowed{}

	tokens := strings.Split(rule, ":")
	if len(tokens) < 1 || len(tokens) > 2 {
		return nil, fmt.Errorf("expected protocol[:portspec] in firewall rule %q", rule)
	}

	o.IPProtocol = tokens[0]
	if len(tokens) == 1 {
		return o, nil
	}

	o.Ports = []string{tokens[1]}
	return o, nil
}

func serializeFirewallAllowed(r *compute.FirewallAllowed) string {
	if len(r.Ports) == 0 {
		return r.IPProtocol
	}

	var tokens []string
	for _, ports := range r.Ports {
		tokens = append(tokens, r.IPProtocol+":"+ports)
	}

	return strings.Join(tokens, ",")
}

func (_ *FirewallRule) RenderGCE(t *gce.GCEAPITarget, a, e, changes *FirewallRule) error {
	var allowed []*compute.FirewallAllowed
	if e.Allowed != nil {
		for _, a := range e.Allowed {
			p, err := parseFirewallAllowed(a)
			if err != nil {
				return err
			}
			allowed = append(allowed, p)
		}
	}
	firewall := &compute.Firewall{
		Name:         *e.Name,
		Network:      e.Network.URL(),
		SourceTags:   e.SourceTags,
		SourceRanges: e.SourceRanges,
		TargetTags:   e.TargetTags,
		Allowed:      allowed,
	}

	if a == nil {
		_, err := t.Cloud.Compute.Firewalls.Insert(t.Cloud.Project, firewall).Do()
		if err != nil {
			return fmt.Errorf("error creating FirewallRule: %v", err)
		}
	} else {
		_, err := t.Cloud.Compute.Firewalls.Update(t.Cloud.Project, *e.Name, firewall).Do()
		if err != nil {
			return fmt.Errorf("error creating FirewallRule: %v", err)
		}
	}

	return nil
}

//func (_*FirewallRule) RenderBash(t *fi.BashTarget, a, e, changes *FirewallRule) error {
//	t.CreateVar(e)
//	if a == nil {
//		//gcloud compute firewall-rules create "${NETWORK}-default-internal" \
//		//--project "${PROJECT}" \
//		//--network "${NETWORK}" \
//		//--source-ranges "10.0.0.0/8" \
//		//--allow "tcp:1-65535,udp:1-65535,icmp" &
//
//
//
//		args := []string{"create-FirewallRule", "--cidr-block", *e.CIDR, "--vpc-id", vpcID, "--query", "FirewallRule.FirewallRuleId"}
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
