package fi

import (
	"crypto"
	crypto_rand "crypto/rand"
	"crypto/x509/pkix"
	"crypto/x509"
	"path"
	"os"
	"io/ioutil"
	"crypto/rsa"
	"fmt"
	"bytes"
	"github.com/golang/glog"
)

type FilesystemCAStore struct {
	basedir       string
	caCertificate *Certificate
	caPrivateKey  crypto.PrivateKey
}

var _ CAStore = &FilesystemCAStore{}

func NewCAStore(basedir string) (CAStore, error) {
	c := &FilesystemCAStore{
		basedir: basedir,
	}
	err := os.MkdirAll(path.Join(basedir, "private"), 0700)
	if err != nil {
		return nil, fmt.Errorf("error creating directory: %v", err)
	}
	err = os.MkdirAll(path.Join(basedir, "issued"), 0700)
	if err != nil {
		return nil, fmt.Errorf("error creating directory: %v", err)
	}
	caCertificate, err := c.loadCertificate(path.Join(basedir, "ca.crt"))
	if err != nil {
		return nil, err
	}
	if caCertificate != nil {
		privateKeyPath := path.Join(basedir, "private", "ca.key")
		caPrivateKey, err := c.loadPrivateKey(privateKeyPath)
		if err != nil {
			return nil, err
		}
		if caPrivateKey == nil {
			glog.Warningf("CA private key was not found %q", privateKeyPath)
			//return nil, fmt.Errorf("error loading CA private key - key not found")
		}
		c.caCertificate = caCertificate
		c.caPrivateKey = caPrivateKey
	} else {
		err := c.generateCACertificate()
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c*FilesystemCAStore) generateCACertificate() error {
	subject := &pkix.Name{
		CommonName: "kubernetes",
	}
	template := &x509.Certificate{
		Subject: *subject,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage: []x509.ExtKeyUsage{},
		BasicConstraintsValid: true,
		IsCA: true,
	}

	caPrivateKey, err := rsa.GenerateKey(crypto_rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("error generating RSA private key: %v", err)
	}

	caCertificate, err := SignNewCertificate(caPrivateKey, template, nil, nil)
	if err != nil {
		return err
	}

	keyPath := path.Join(c.basedir, "private", "ca.key")
	err = c.storePrivateKey(caPrivateKey, keyPath)
	if err != nil {
		return err
	}

	certPath := path.Join(c.basedir, "ca.crt")
	err = c.storeCertificate(caCertificate, certPath)
	if err != nil {
		return err
	}

	// Make double-sure it round-trips
	caCertificate, err = c.loadCertificate(certPath)
	if err != nil {
		return err
	}

	c.caPrivateKey = caPrivateKey
	c.caCertificate = caCertificate
	return nil
}

func (c*FilesystemCAStore) getSubjectKey(subject *pkix.Name) string {
	seq := subject.ToRDNSequence()
	var s bytes.Buffer
	for _, rdnSet := range seq {
		for _, rdn := range rdnSet {
			if s.Len() != 0 {
				s.WriteString(",")
			}
			key := ""
			t := rdn.Type
			if len(t) == 4 && t[0] == 2 && t[1] == 5 && t[2] == 4 {
				switch t[3] {
				case 3:
					key = "cn"
				case 5:
					key = "serial"
				case 6:
					key = "c"
				case 7:
					key = "l"
				case 10:
					key = "o"
				case 11:
					key = "ou"
				}
			}
			if key == "" {
				key = t.String()
			}
			s.WriteString(fmt.Sprintf("%v=%v", key, rdn.Value))
		}
	}
	return s.String()
}

func (c*FilesystemCAStore) buildCertificatePath(subject *pkix.Name) string {
	key := c.getSubjectKey(subject)
	return path.Join(c.basedir, "issued", key + ".crt")
}

func (c*FilesystemCAStore) buildPrivateKeyPath(subject *pkix.Name) string {
	key := c.getSubjectKey(subject)
	return path.Join(c.basedir, "private", key + ".key")
}

func (c *FilesystemCAStore) GetCACert() (*Certificate, error) {
	return c.caCertificate, nil
}

func (c *FilesystemCAStore) FindCAKey() (crypto.PrivateKey, error) {
	return c.caPrivateKey, nil
}

func (c *FilesystemCAStore) loadCertificate(p string) (*Certificate, error) {
	data, err := ioutil.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
	}
	cert, err := LoadCertificate(data)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

func (c *FilesystemCAStore) FindCert(subject *pkix.Name) (*Certificate, error) {
	p := c.buildCertificatePath(subject)
	return c.loadCertificate(p)
}

func (c *FilesystemCAStore) IssueCert(privateKey crypto.PrivateKey, template *x509.Certificate) (*Certificate, error) {
	p := c.buildCertificatePath(&template.Subject)

	if c.caPrivateKey == nil {
		return nil, fmt.Errorf("ca.key was not found; cannot issue certificates")
	}
	cert, err := SignNewCertificate(privateKey, template, c.caCertificate.Certificate, c.caPrivateKey)
	if err != nil {
		return nil, err
	}

	err = c.storeCertificate(cert, p)
	if err != nil {
		return nil, err
	}

	// Make double-sure it round-trips
	return c.loadCertificate(p)
}

func (c *FilesystemCAStore) loadPrivateKey(p string) (crypto.PrivateKey, error) {
	data, err := ioutil.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
	}
	k, err := parsePEMPrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("error parsing private key from %q: %v", p, err)
	}
	return k, nil
}

func (c *FilesystemCAStore) FindPrivateKey(subject *pkix.Name) (crypto.PrivateKey, error) {
	p := c.buildPrivateKeyPath(subject)
	return c.loadPrivateKey(p)
}

func (c *FilesystemCAStore) CreatePrivateKey(subject *pkix.Name) (crypto.PrivateKey, error) {
	p := c.buildPrivateKeyPath(subject)

	privateKey, err := rsa.GenerateKey(crypto_rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("error generating RSA private key: %v", err)
	}

	err = c.storePrivateKey(privateKey, p)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func (c*FilesystemCAStore) storePrivateKey(privateKey crypto.PrivateKey, p string) error {
	var data bytes.Buffer
	err := WritePrivateKey(privateKey, &data)
	if err != nil {
		return err
	}

	return c.writeFile(data.Bytes(), p)
}

func (c*FilesystemCAStore) storeCertificate(cert *Certificate, p string) error {
	var data bytes.Buffer
	err := cert.WriteCertificate(&data)
	if err != nil {
		return err
	}

	return c.writeFile(data.Bytes(), p)
}

func (c*FilesystemCAStore) writeFile(data []byte, p string) error {
	// TODO: concurrency?
	err := ioutil.WriteFile(p, data, 0600)
	if err != nil {
		// TODO: Delete file on disk?  Write a temp file and move it atomically?
		return fmt.Errorf("error writing certificate/key data to path %q: %v", p, err)
	}
	return nil
}
