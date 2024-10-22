package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/sheryarbutt/certificate-manager/pkg/constants"
)

// GetTemplate returns a x509.Certificate template with the given DNS name and validity
func GetTemplate(dnsName string, validity time.Duration) x509.Certificate {
	return x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: dnsName,
		},
		DNSNames:  []string{dnsName},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(validity),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
	}
}

// GenerateSelfSignedCertificate generates a self-signed certificate for the given DNS name
func CreateSelfSignedCertificate(dnsName string, validity time.Duration) ([]byte, []byte, error) {
	// Generate a new private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// Create a template for the certificate
	template := GetTemplate(dnsName, validity)

	// Create the self-signed certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}

	// PEM encode the certificate
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  constants.TypeCertificate,
		Bytes: certBytes,
	})

	// PEM encode the private key
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  constants.TypePrivateKey,
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return certPEM, keyPEM, nil
}

// IsCertificateExpired checks if the given certificate is expired
func IsCertificateExpired(cert []byte) (bool, error) {
	pemBlock, _ := pem.Decode(cert)
	if pemBlock == nil {
		return false, nil
	}

	certificate, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return false, err
	}

	return time.Now().After(certificate.NotAfter), nil
}
