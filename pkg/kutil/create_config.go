package kutil

import (
	"os"
	"path"
	"os/exec"
	"fmt"
	"github.com/golang/glog"
	"strings"
)

type KubeconfigBuilder struct {
	KubectlPath     string
	KubeconfigPath  string

	KubeMasterIP    string

	Context         string

	KubeBearerToken string
	KubeUser        string
	KubePassword    string

	CACert          string
	KubecfgCert     string
	KubecfgKey      string
}

func (c*KubeconfigBuilder) Init() {
	c.KubectlPath = "kubectl" // default to in-path

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		homedir := os.Getenv("HOME")
		kubeconfig = path.Join(homedir, ".kube", "config")
	}
	c.KubeconfigPath = kubeconfig
}

//# Generate kubeconfig data for the created cluster.
//# Assumed vars:
//#   KUBE_USER
//#   KUBE_PASSWORD
//#   KUBE_MASTER_IP
//#   KUBECONFIG
//#   CONTEXT
//#
//# If the apiserver supports bearer auth, also provide:
//#   KUBE_BEARER_TOKEN
//#
//# The following can be omitted for --insecure-skip-tls-verify
//#   KUBE_CERT
//#   KUBE_KEY
//#   CA_CERT
func (c*KubeconfigBuilder)  CreateKubeconfig() error {
	if _, err := os.Stat(c.KubeconfigPath); os.IsNotExist(err) {
		// mkdir -p $(dirname "${KUBECONFIG}")
		err := os.MkdirAll(path.Dir(c.KubeconfigPath), 0700)
		if err != nil {
			return fmt.Errorf("error creating directories for %q: %v", c.KubeconfigPath, err)
		}
		// touch "${KUBECONFIG}"
		f, err := os.OpenFile(c.KubeconfigPath, os.O_RDWR | os.O_CREATE, 0600)
		if err != nil {
			return fmt.Errorf("error creating config file %q: %v", c.KubeconfigPath, err)
		}
		f.Close()
	}

	var clusterArgs []string

	//"--server=${KUBE_SERVER:-https://${KUBE_MASTER_IP}}"
	clusterArgs = append(clusterArgs, "--server=https://" + c.KubeMasterIP)

	if c.CACert == "" {
		clusterArgs = append(clusterArgs, "--insecure-skip-tls-verify=true")
	} else {
		clusterArgs = append(clusterArgs, "--certificate-authority=" + c.CACert)
		clusterArgs = append(clusterArgs, "--embed-certs=true")
	}

	var userArgs []string

	if c.KubeBearerToken != "" {
		userArgs = append(userArgs, "--token=" + c.KubeBearerToken)
	} else if c.KubeUser != "" && c.KubePassword != "" {
		userArgs = append(userArgs, "--username=" + c.KubeUser)
		userArgs = append(userArgs, "--password=" + c.KubePassword)
	}

	if c.KubecfgCert != "" && c.KubecfgKey != "" {
		userArgs = append(userArgs, "--client-certificate=" + c.KubecfgCert)
		userArgs = append(userArgs, "--client-key=" + c.KubecfgKey)
		userArgs = append(userArgs, "--embed-certs=true")
	}

	setClusterArgs := []string{"config", "set-cluster", c.Context}
	setClusterArgs = append(setClusterArgs, clusterArgs...)
	err := c.kubectl(setClusterArgs...)
	if err != nil {
		return err
	}

	if len(userArgs) != 0 {
		setCredentialsArgs := []string{"config", "set-credentials", c.Context}
		setCredentialsArgs = append(setCredentialsArgs, userArgs...)
		err := c.kubectl(setCredentialsArgs...)
		if err != nil {
			return err
		}
	}

	err = c.kubectl("config", "set-context", c.Context, "--cluster=" + c.Context, "--user=" + c.Context)
	if err != nil {
		return err
	}
	err = c.kubectl("config", "use-context", c.Context, "--cluster=" + c.Context, "--user=" + c.Context)
	if err != nil {
		return err
	}

	// If we have a bearer token, also create a credential entry with basic auth
	// so that it is easy to discover the basic auth password for your cluster
	// to use in a web browser.
	if c.KubeBearerToken != "" && c.KubeUser != "" && c.KubePassword != "" {
		err := c.kubectl("config", "set-credentials", c.Context + "-basic-auth", "--username=" + c.KubeUser, "--password=" + c.KubePassword)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Wrote config for %s to %q\n", c.Context, c.KubeconfigPath)
	return nil
}

func (c*KubeconfigBuilder)  kubectl(args... string) error {
	cmd := exec.Command(c.KubectlPath, args...)
	env := os.Environ()
	env = append(env, fmt.Sprintf("KUBECONFIG=%s", c.KubeconfigPath))
	cmd.Env = env

	glog.V(2).Infof("Running command: %s %s", cmd.Path, strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Info("error running kubectl:")
		glog.Info(string(output))
		return fmt.Errorf("error running kubectl")
	}

	glog.V(2).Info(string(output))
	return nil
}

