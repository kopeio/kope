package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/golang/glog"
	"io/ioutil"
	"github.com/kopeio/kope/pkg/kutil"
	"os"
	"path"
)

type CreateKubecfgCmd struct {
	Master         string
	SSHIdentity    string
	UseKubeletCert bool
}

var createKubecfg CreateKubecfgCmd

func init() {
	cmd := &cobra.Command{
		Use:   "kubecfg",
		Short: "Create kubecfg file from master",
		Long: `Connects to your master server over SSH, and builds a kubecfg file from the settings.`,
		Run: func(cmd *cobra.Command, args[]string) {
			err := createKubecfg.Run()
			if err != nil {
				glog.Exitf("%v", err)
			}
		},
	}

	createCmd.AddCommand(cmd)

	cmd.Flags().StringVarP(&createKubecfg.Master, "master", "m", "", "Master IP address or hostname")
	cmd.Flags().StringVarP(&createKubecfg.SSHIdentity, "i", "i", "", "SSH private key")
	cmd.Flags().BoolVar(&createKubecfg.UseKubeletCert, "use-kubelet-cert", false, "Build using the kublet cert (useful if the kubecfg cert is not available)")
}

func (c*CreateKubecfgCmd) Run() error {
	if c.Master == "" {
		return fmt.Errorf("--master must be specified")
	}
	fmt.Printf("Connecting to %s\n", c.Master)

	master := &kutil.NodeSSH{
		IP: c.Master,
	}
	if c.SSHIdentity != "" {
		err := master.AddSSHIdentity(c.SSHIdentity)
		if err != nil {
			return err
		}
	}

	conf, err := master.ReadConfiguration()
	if err != nil {
		return err
	}

	instancePrefix := conf.Settings["INSTANCE_PREFIX"]
	if instancePrefix == "" {
		return fmt.Errorf("cannot determine INSTANCE_PREFIX")
	}

	tmpdir, err := ioutil.TempDir("", "k8s")
	if err != nil {
		return fmt.Errorf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	b := &kutil.KubeconfigBuilder{}
	b.Init()
	b.Context = "aws_" + instancePrefix

	caCertPath := path.Join(tmpdir, "ca.crt")
	err = downloadFile(master, "/srv/kubernetes/ca.crt", caCertPath)
	//caCertPath, err := confToFile(tmpdir, conf, "CA_CERT")
	if err != nil {
		return err
	}

	//kubecfgCertConfKey := "KUBECFG_CERT"
	//kubecfgKeyConfKey := "KUBECFG_KEY"
	//if c.UseKubeletCert {
	//	kubecfgCertConfKey = "KUBELET_CERT"
	//	kubecfgKeyConfKey = "KUBELET_KEY"
	//} else {
	//	if conf[kubecfgCertConfKey] == "" {
	//		fmt.Printf("%s was not found in the configuration; you may want to specify --use-kubelet-cert\n", kubecfgCertConfKey)
	//	}
	//}
	//

	kubecfgCertPath := path.Join(tmpdir, "kubecfg.crt")
	err = downloadFile(master, "/srv/kubernetes/kubecfg.crt", kubecfgCertPath)
	//caCertPath, err := confToFile(tmpdir, conf, "CA_CERT")
	if err != nil {
		return err
	}
	kubecfgKeyPath := path.Join(tmpdir, "kubecfg.key")
	err = downloadFile(master, "/srv/kubernetes/kubecfg.key", kubecfgKeyPath)
	//caCertPath, err := confToFile(tmpdir, conf, "CA_CERT")
	if err != nil {
		return err
	}

	//kubeCertPath, err := confToFile(tmpdir, conf, kubecfgCertConfKey)
	//if err != nil {
	//	return err
	//}
	//kubeKeyPath, err := confToFile(tmpdir, conf, kubecfgKeyConfKey)
	//if err != nil {
	//	return err
	//}

	b.CACert = caCertPath
	b.KubecfgCert = kubecfgCertPath
	b.KubecfgKey = kubecfgKeyPath
	b.KubeMasterIP = c.Master

	err = b.CreateKubeconfig()
	if err != nil {
		return err
	}

	return nil
}


func downloadFile(master *kutil.NodeSSH, remotePath string, localPath string) (error) {
	b, err := master.ReadFile(remotePath)
	if err != nil {
		return err
	}

	if len(b) == 0 {
		return fmt.Errorf("remote file  %q was unexpectedly empty", remotePath)
	}

	err = os.MkdirAll(path.Dir(localPath), 0700)
	if err != nil {
		return fmt.Errorf("error creating directories for path %q: %v", path.Dir(localPath), err)
	}

	err = ioutil.WriteFile(localPath, b, 0700)
	if err != nil {
		return fmt.Errorf("error writing to file %q: %v", localPath, err)
	}

	return nil
}