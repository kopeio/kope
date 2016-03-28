package units

import (
	"fmt"
	"crypto/x509"
	"github.com/kopeio/kope/pkg/fi"
	"bytes"
	"crypto/x509/pkix"
	"github.com/golang/glog"
	"crypto"
	"net"
)

type CertBuilder struct {
	fi.SimpleUnit

	// TODO: This is messy ... random dependencies, random write-backs...
	Kubernetes *K8s
	MasterIP   HasAddress
	MasterName *string
}

func (c*CertBuilder) Key() string {
	return "certbuilder"
}

func buildCertificateAlternateNames(k8s *K8s) ([]string, error) {
	var sans []string
	apiServerIP, err := k8s.GetWellKnownServiceIP(1)
	if err != nil {
		return nil, err
	}
	sans = append(sans, apiServerIP.String())
	for _, s := range k8s.MasterExtraSans {
		sans = append(sans, s)
	}
	sans = append(sans, "kubernetes")
	sans = append(sans, "kubernetes.default")
	sans = append(sans, "kubernetes.default.svc")
	sans = append(sans, "kubernetes.default.svc." + k8s.DNSDomain)

	if k8s.MasterName != "" {
		sans = append(sans, k8s.MasterName)
	}

	if k8s.MasterInternalIP != "" {
		sans = append(sans, k8s.MasterInternalIP)
	}

	return sans, nil
}

func (b *CertBuilder) Run(c *fi.RunContext) error {
	k8s := b.Kubernetes

	certs := c.CAStore()

	if k8s.CACert == nil {
		caCert, err := certs.GetCACert()
		if err != nil {
			return err
		}

		k8s.CACert = certToResource(caCert)
	}

	if k8s.CAKey == nil {
		caKey, err := certs.FindCAKey()
		if err != nil {
			return err
		}

		if caKey != nil {
			k8s.CAKey = keyToResource(caKey)
		} else {
			// We allow this for upgrades
			glog.Warningf("CA key is not set")
		}
	}

	kubecfgSubject := &pkix.Name{
		CommonName: "kubecfg",
	}

	if k8s.KubecfgCert == nil {
		kubecfgCert, err := certs.FindCert(kubecfgSubject)
		if err != nil {
			return err
		}

		if kubecfgCert == nil {
			template := &x509.Certificate{
				Subject: *kubecfgSubject,
				KeyUsage: x509.KeyUsageDigitalSignature,
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth },
				BasicConstraintsValid: true,
				IsCA: false,
			}

			privateKey, err := certs.CreatePrivateKey(kubecfgSubject)
			if err != nil {
				return err
			}
			kubecfgCert, err = certs.IssueCert(privateKey, template)
			if err != nil {
				return err
			}
		}

		k8s.KubecfgCert = certToResource(kubecfgCert)
	}

	if k8s.KubecfgKey == nil {
		key, err := certs.FindPrivateKey(kubecfgSubject)
		if err != nil {
			return err
		}

		if key == nil {
			return fmt.Errorf("kubecfg key not found")
		}
		k8s.KubecfgKey = keyToResource(key)
	}

	kubeletSubject := &pkix.Name{
		CommonName: "kubelet",
	}

	if k8s.KubeletCert == nil {
		kubeletCert, err := certs.FindCert(kubeletSubject)
		if err != nil {
			return err
		}

		if kubeletCert == nil {
			template := &x509.Certificate{
				Subject: *kubeletSubject,
				KeyUsage: x509.KeyUsageDigitalSignature,
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth },
				BasicConstraintsValid: true,
				IsCA: false,
			}

			privateKey, err := certs.CreatePrivateKey(kubeletSubject)
			if err != nil {
				return err
			}
			kubeletCert, err = certs.IssueCert(privateKey, template)
			if err != nil {
				return err
			}
		}

		k8s.KubeletCert = certToResource(kubeletCert)
	}

	if k8s.KubeletKey == nil {
		key, err := certs.FindPrivateKey(kubeletSubject)
		if err != nil {
			return err
		}

		if key == nil {
			return fmt.Errorf("kubelet key not found")
		}
		k8s.KubeletKey = keyToResource(key)
	}

	masterSubject := &pkix.Name{
		CommonName: "kubernetes-master",
	}

	if k8s.MasterCert == nil {
		masterCert, err := certs.FindCert(masterSubject)
		if err != nil {
			return err
		}

		if masterCert == nil {
			template := &x509.Certificate{
				Subject: *masterSubject,
				KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth },
				BasicConstraintsValid: true,
				IsCA: false,
			}

			alternateNames, err := buildCertificateAlternateNames(k8s)
			if err != nil {
				return err
			}
			if b.MasterIP != nil {
				ip, err := b.MasterIP.FindAddress(c.Cloud())
				if err != nil {
					return fmt.Errorf("error querying for MasterIP: %v", err)
				}
				if fi.StringValue(ip) == "" {
					return fmt.Errorf("cannot build SANs for master cert until master Public IP is allocated")
				}
				alternateNames = append(alternateNames, fi.StringValue(ip))
			}
			for _, san := range alternateNames {
				if ip := net.ParseIP(san); ip != nil {
					template.IPAddresses = append(template.IPAddresses, ip)
				} else {
					template.DNSNames = append(template.DNSNames, san)
				}
			}

			glog.V(2).Infof("X509 SANS IPAddresses: %v", template.IPAddresses)
			glog.V(2).Infof("X509 SANS DNSNames: %v", template.DNSNames)

			privateKey, err := certs.CreatePrivateKey(masterSubject)
			if err != nil {
				return err
			}
			masterCert, err = certs.IssueCert(privateKey, template)
			if err != nil {
				return err
			}
		}

		k8s.MasterCert = certToResource(masterCert)
	}

	if k8s.MasterKey == nil {
		key, err := certs.FindPrivateKey(masterSubject)
		if err != nil {
			return err
		}

		if key == nil {
			return fmt.Errorf("kubernetes-master key not found")
		}
		k8s.MasterKey = keyToResource(key)
	}

	return nil
}

func certToResource(cert *fi.Certificate) fi.Resource {
	var data bytes.Buffer
	err := cert.WriteCertificate(&data)
	if err != nil {
		glog.Fatalf("error writing CA certificate: %v", err)
	}
	return fi.NewBytesResource(data.Bytes())
}

func keyToResource(privateKey crypto.PrivateKey) fi.Resource {
	var data bytes.Buffer
	err := fi.WritePrivateKey(privateKey, &data)
	if err != nil {
		glog.Fatalf("error writing private key: %v", err)
	}
	return fi.NewBytesResource(data.Bytes())
}