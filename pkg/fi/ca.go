package fi

import (
	"crypto/x509/pkix"
	"crypto/x509"
	"crypto"
	crypto_rand "crypto/rand"
	"fmt"
	"time"
	"crypto/rsa"
	"math/big"
	"io"
	"encoding/pem"
	"github.com/golang/glog"
)

type Certificate struct {
	Subject     pkix.Name
	IsCA        bool

	Certificate *x509.Certificate
	PublicKey   crypto.PublicKey
}

type CAStore interface {
	GetCACert() (*Certificate, error)
	FindCAKey() (crypto.PrivateKey, error)
	FindCert(subject *pkix.Name) (*Certificate, error)
	IssueCert(privateKey crypto.PrivateKey, template *x509.Certificate) (*Certificate, error)
	FindPrivateKey(subject *pkix.Name) (crypto.PrivateKey, error)
	CreatePrivateKey(subject *pkix.Name) (crypto.PrivateKey, error)
}

func LoadCertificate(pemData []byte) (*Certificate, error) {
	cert, err := parsePEMCertificate(pemData)
	if err != nil {
		return nil, err
	}

	c := &Certificate{
		Subject: cert.Subject,
		Certificate: cert,
		PublicKey: cert.PublicKey,
		IsCA: cert.IsCA,
	}
	return c, nil
}

func SignNewCertificate(privateKey crypto.PrivateKey, template *x509.Certificate, signer *x509.Certificate, signerPrivateKey crypto.PrivateKey) (*Certificate, error) {
	if template.PublicKey == nil {
		rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
		if ok {
			template.PublicKey = rsaPrivateKey.Public()
		}
	}

	if template.PublicKey == nil {
		return nil, fmt.Errorf("PublicKey not set, and cannot be determined from %T", privateKey)
	}

	now := time.Now()
	if template.NotBefore.IsZero() {
		template.NotBefore = now.Add(time.Hour * -48)
	}

	if template.NotAfter.IsZero() {
		template.NotAfter = now.Add(time.Hour * 10 * 365 * 24)
	}

	if template.SerialNumber == nil {
		serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
		serialNumber, err := crypto_rand.Int(crypto_rand.Reader, serialNumberLimit)
		if err != nil {
			return nil, fmt.Errorf("error generating certificate serial number: %s", err)
		}
		template.SerialNumber = serialNumber
	}
	var parent *x509.Certificate
	if signer != nil {
		parent = signer
	} else {
		parent = template
		signerPrivateKey = privateKey
	}

	if template.KeyUsage == 0 {
		template.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	}

	if template.ExtKeyUsage == nil {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}
	//c.SignatureAlgorithm  = do we want to overrride?

	certificateData, err := x509.CreateCertificate(crypto_rand.Reader, template, parent, template.PublicKey, signerPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("error creating certificate: %v", err)
	}

	c := &Certificate{}
	c.PublicKey = template.PublicKey

	cert, err := x509.ParseCertificate(certificateData)
	if err != nil {
		return nil, fmt.Errorf("error parsing certificate: %v", err)
	}
	c.Certificate = cert

	return c, nil
}

func (c*Certificate) WriteCertificate(w io.Writer) error {
	return pem.Encode(w, &pem.Block{Type: "CERTIFICATE", Bytes: c.Certificate.Raw})
}

func parsePEMCertificate(pemData []byte) (*x509.Certificate, error) {
	for {
		block, rest := pem.Decode(pemData)
		if block == nil {
			return nil, fmt.Errorf("could not parse certificate")
		}

		if block.Type == "CERTIFICATE" {
			glog.V(2).Infof("Parsing pem block: %q", block.Type)
			return x509.ParseCertificate(block.Bytes)
		} else {
			glog.Infof("Ignoring unexpected PEM block: %q", block.Type)
		}

		pemData = rest
	}
}

func WritePrivateKey(privateKey crypto.PrivateKey, w io.Writer) error {
	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	if ok {
		return pem.Encode(w, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rsaPrivateKey)})
	}

	return fmt.Errorf("unknown private key type: %T", privateKey)
}

func parsePEMPrivateKey(pemData []byte) (crypto.PrivateKey, error) {
	for {
		block, rest := pem.Decode(pemData)
		if block == nil {
			return nil, fmt.Errorf("could not parse private key")
		}

		if block.Type == "RSA PRIVATE KEY" {
			glog.V(2).Infof("Parsing pem block: %q", block.Type)
			return x509.ParsePKCS1PrivateKey(block.Bytes)
		} else if block.Type == "PRIVATE KEY" {
			glog.V(2).Infof("Parsing pem block: %q", block.Type)
			k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, err
			}
			return k.(crypto.PrivateKey), nil
		} else {
			glog.Infof("Ignoring unexpected PEM block: %q", block.Type)
		}

		pemData = rest
	}
}