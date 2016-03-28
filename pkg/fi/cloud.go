package fi

type ProviderID string

const ProviderAWS ProviderID = "aws"
const ProviderGCE ProviderID = "gce"

type Cloud interface {
	ProviderID() ProviderID
}

/*
func (c *Cloud) init() error {
	// For EC2, check /sys/hypervisor/uuid, then check 169.254.169.254
	//	cat /sys/hypervisor/uuid
	//	ec21884e-23e4-dcf2-d27e-4495fedb2abd

	// curl http://169.254.169.254/
	glog.Warning("Cloud detection hard-coded")

	c.ProviderID = "aws"

	return nil
}
*/
/*
func (c *Cloud) IsAWS() bool {
	return c.ProviderID == "aws"
}

func (c *Cloud) IsGCE() bool {
	return c.ProviderID == "gce"
}

func (c *Cloud) IsVagrant() bool {
	return c.ProviderID == "vagrant"
}
*/
