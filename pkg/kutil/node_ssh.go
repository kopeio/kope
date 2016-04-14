package kutil

import (
	"encoding/base64"
	"fmt"
	"github.com/golang/glog"
	"github.com/kopeio/kope/pkg/fi"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"strings"
)

type NodeSSH struct {
	IP        string
	SSHConfig ssh.ClientConfig
	sshClient *ssh.Client
}

func (m *NodeSSH) AddSSHIdentity(p string) error {
	a, err := parsePrivateKeyFile(p)
	if err != nil {
		return err
	}
	m.SSHConfig.Auth = append(m.SSHConfig.Auth, a)
	return nil
}

func (m *NodeSSH) dial() (*ssh.Client, error) {
	users := []string{"admin", "ubuntu"}
	if m.SSHConfig.User != "" {
		users = []string{m.SSHConfig.User}
	}

	var lastError error
	for _, user := range users {
		m.SSHConfig.User = user
		sshClient, err := ssh.Dial("tcp", m.IP+":22", &m.SSHConfig)
		if err == nil {
			return sshClient, err
		}
		lastError = err
	}

	return nil, fmt.Errorf("error connecting to SSH on server %q: %v", m.IP, lastError)
}

func (m *NodeSSH) GetSSHClient() (*ssh.Client, error) {
	if m.sshClient == nil {
		sshClient, err := m.dial()
		if err != nil {
			return nil, err
		}
		m.sshClient = sshClient
	}
	return m.sshClient, nil
}

func (m *NodeSSH) ReadConfiguration() (*MasterConfiguration, error) {
	sshClient, err := m.GetSSHClient()
	if err != nil {
		return nil, err
	}

	sshSession, err := sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("error creating SSH session: %v", err)
	}
	defer sshSession.Close()

	//output, err := sshSession.CombinedOutput("cat /etc/kubernetes/kube_env.yaml")
	//if err != nil {
	//	return fmt.Errorf("error running SSH command: %v", err)
	//}
	outputBytes, err := sshSession.Output("/bin/bash -c 'curl -s http://169.254.169.254/latest/user-data | base64'")
	if err != nil {
		return nil, fmt.Errorf("error running SSH command: %v", err)
	}

	//glog.Infof("User data: %v", string(outputBytes))

	outputBytes, err = base64.StdEncoding.DecodeString(string(outputBytes))
	if err != nil {
		return nil, fmt.Errorf("error decoding base64 user-data")
	}

	if len(outputBytes) > 2 && outputBytes[0] == 31 && outputBytes[1] == 139 {
		// GZIP
		glog.V(2).Infof("gzip data detected; will decompress")

		outputBytes, err = fi.GunzipBytes(outputBytes)
		if err != nil {
			return nil, fmt.Errorf("error decompressing user data: %v", err)
		}
	}
	settings := make(map[string]string)

	output := string(outputBytes)
	version := ""
	if strings.Contains(output, "install-salt master") || strings.Contains(output, "dpkg -s salt-minion") {
		version = "1.1"
	} else {
		version = "1.2"
	}
	if version == "1.1" {
		for _, line := range strings.Split(string(output), "\n") {
			if !strings.HasPrefix(line, "readonly ") {
				continue
			}
			line = line[9:]
			sep := strings.Index(line, "=")
			k := ""
			v := ""
			if sep != -1 {
				k = line[0:sep]
				v = line[sep+1:]
			}

			if k == "" {
				glog.V(4).Infof("Unknown line: %s", line)
			}

			if len(v) >= 2 && v[0] == '\'' && v[len(v)-1] == '\'' {
				v = v[1 : len(v)-1]
			}
			settings[k] = v
		}
	} else {
		for _, line := range strings.Split(string(output), "\n") {
			sep := strings.Index(line, ": ")
			k := ""
			v := ""
			if sep != -1 {
				k = line[0:sep]
				v = line[sep+2:]
			}

			if k == "" {
				glog.V(4).Infof("Unknown line: %s", line)
			}

			if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
				v = v[1 : len(v)-1]
			}
			settings[k] = v
		}
	}

	c := &MasterConfiguration{
		Version:  version,
		Settings: settings,
	}
	return c, nil
}

func (m *NodeSSH) ReadFile(remotePath string) ([]byte, error) {
	b, err := m.exec("sudo cat " + remotePath)
	if err != nil {
		return nil, fmt.Errorf("error reading remote file %q: %v", remotePath, err)
	}
	return b, nil
}

func (m *NodeSSH) exec(cmd string) ([]byte, error) {
	client, err := m.GetSSHClient()
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("error creating SSH session: %v", err)
	}
	defer session.Close()

	b, err := session.Output(cmd)
	if err != nil {
		return nil, fmt.Errorf("error executing command %q: %v", cmd, err)
	}
	return b, nil
}

func (m *NodeSSH) GetMetadata(key string) (string, error) {
	b, err := m.exec("curl -s http://169.254.169.254/latest/meta-data/" + key)
	if err != nil {
		return "", fmt.Errorf("error querying for metadata %q: %v", key, err)
	}
	return string(b), nil
}

func (m *NodeSSH) InstanceType() (string, error) {
	return m.GetMetadata("instance-type")
}

func (m *NodeSSH) GetMetadataList(key string) ([]string, error) {
	d, err := m.GetMetadata(key)
	if err != nil {
		return nil, err
	}
	var macs []string
	for _, line := range strings.Split(d, "\n") {
		mac := line
		mac = strings.Trim(mac, "/")
		mac = strings.TrimSpace(mac)
		if mac == "" {
			continue
		}
		macs = append(macs, mac)
	}

	return macs, nil
}

func parsePrivateKeyFile(p string) (ssh.AuthMethod, error) {
	buffer, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("error reading SSH key file %q: %v", p, err)
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil, fmt.Errorf("error parsing key file %q: %v", p, err)
	}
	return ssh.PublicKeys(key), nil
}

type MasterConfiguration struct {
	Version  string
	Settings map[string]string
}
