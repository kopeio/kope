package awsunits

import (
	"fmt"
	"io"

	"github.com/kopeio/kope/pkg/fi"
	"gopkg.in/yaml.v2"
	"bytes"
	"strconv"
)

type MasterScript struct {
	fi.SimpleUnit

	Config   *K8s

	contents string
}

func (s *MasterScript) Key() string {
	return "master-script"
}

var _ fi.Resource = &MasterScript{}

type NodeScript struct {
	fi.SimpleUnit

	Config   *K8s

	contents string
}

var _ fi.Resource = &NodeScript{}

func (s *NodeScript) Key() string {
	return "node-script"
}

//func (m *NodeScript) Prefix() string {
//	return "node_script"
//}

func buildScript(c *fi.RunContext, k *K8s, isMaster bool) (string, error) {
	var bootstrapScriptURL string

	{
		url, _, err := c.Target.PutResource("bootstrap", k.BootstrapScript, fi.HashAlgorithmSHA1)
		if err != nil {
			return "", err
		}
		bootstrapScriptURL = url
	}

	data, err := k.BuildEnv(c, isMaster)
	if err != nil {
		return "", err
	}
	data["AUTO_UPGRADE"] = strconv.FormatBool(true)
	// TODO: get rid of these exceptions / harmonize with common or GCE
	data["DOCKER_STORAGE"] = k.DockerStorage
	data["API_SERVERS"] = k.MasterInternalIP

	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("error marshaling env to yaml: %v", err)
	}

	// We send this to the ami as a startup script in the user-data field.  Requires a compatible ami
	var s fi.ScriptWriter
	s.WriteString("#! /bin/bash\n")
	s.WriteString("mkdir -p /var/cache/kubernetes-install\n")
	s.WriteString("cd /var/cache/kubernetes-install\n")

	s.WriteHereDoc("kube_env.yaml", string(yamlData))

	s.WriteString("wget -O bootstrap " + bootstrapScriptURL + "\n")
	s.WriteString("chmod +x bootstrap\n")
	s.WriteString("mkdir -p /etc/kubernetes\n")
	s.WriteString("mv kube_env.yaml /etc/kubernetes\n")
	s.WriteString("mv bootstrap /etc/kubernetes/\n")

	s.WriteString("cat > /etc/rc.local << EOF_RC_LOCAL\n")
	s.WriteString("#!/bin/sh -e\n")
	// We want to be sure that we don't pass an argument to bootstrap
	s.WriteString("/etc/kubernetes/bootstrap\n")
	s.WriteString("exit 0\n")
	s.WriteString("EOF_RC_LOCAL\n")
	s.WriteString("/etc/kubernetes/bootstrap\n")

	return s.AsString(), nil
}

func (m*MasterScript) Run(c *fi.RunContext) error {
	isMaster := true
	contents, err := buildScript(c, m.Config, isMaster)
	if err != nil {
		return err
	}
	m.contents = contents
	return nil
}

func (m *MasterScript) Open() (io.ReadSeeker, error) {
	if m.contents == "" {
		panic("executed out of sequence")
	}
	return bytes.NewReader([]byte(m.contents)), nil
}

func (m*NodeScript) Run(c *fi.RunContext) error {
	isMaster := false
	contents, err := buildScript(c, m.Config, isMaster)
	if err != nil {
		return err
	}
	m.contents = contents
	return nil
}

func (m *NodeScript) Open() (io.ReadSeeker, error) {
	if m.contents == "" {
		panic("executed out of sequence")
	}
	return bytes.NewReader([]byte(m.contents)), nil
}

//func (m *MasterScript) Prefix() string {
//	return "master_script"
//}
