package controllers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	certsv1 "github.com/sheryarbutt/certificate-manager/api/v1"
	"github.com/sheryarbutt/certificate-manager/pkg/objects"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/sheryarbutt/certificate-manager/pkg/constants"
)

func (r *CertificateReconciler) handleCreate(ctx context.Context, req ctrl.Request, instance *certsv1.Certificate) error {
	log := r.Log.WithValues("certificate", req.NamespacedName)
	log.Info("Creating/Updating Certificate: ", instance.ObjectMeta.Name)

	// Check if the certificate already exists
	secret := objects.Secret(instance.Spec.SecretRef, req.Namespace)
	err := r.Get(ctx, client.ObjectKeyFromObject(secret), secret)
	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Failed to get Secret")
		return err
	}

	// If the secret does not exist, create it
	if errors.IsNotFound(err) {
		log.Info("Creating Secret")
		// Generate a self-signed certificate
		cert, key, err := r.GenerateSelfSignedCertificate(ctx, instance)
		if err != nil {
			log.Error(err, "Failed to generate self-signed certificate")
			return err
		}

		// Create the Secret object
		secret := objects.Secret(instance.Spec.SecretRef, instance.Namespace)
		secret.Data = map[string][]byte{
			"tls.crt": cert,
			"tls.key": key,
		}

		// Create the Secret
		err = r.Create(ctx, secret)
		if err != nil {
			log.Error(err, "Failed to create Secret")
			return err
		}

		// Set owner reference on the Secret
		err = controllerutil.SetControllerReference(instance, secret, r.Scheme)
		if err != nil {
			log.Error(err, "Failed to set owner reference on Secret")
			return err
		}

		if instance.Spec.PurgeOnDelete {
			// Add finalizer to the Certificate
			controllerutil.AddFinalizer(instance, constants.Finalizer)
			if err := r.Update(ctx, instance); err != nil {
				log.Error(err, "Failed to add finalizer to Certificate")
				return err
			}
		}

		if instance.Spec.ReloadOnChange {
			// Add Env to deployments that use this secret
			// This will reload the deployments using this secret
			if err := r.addENVToDeployments(ctx, req, instance, secret); err != nil {
				log.Error(err, "Failed to add env to deployments")
				return err
			}
		}
	}

	return nil
}

// generateCertificate generates a self-signed certificate for the given DNS name
func (r *CertificateReconciler) GenerateSelfSignedCertificate(ctx context.Context, instance *certsv1.Certificate) ([]byte, []byte, error) {
	log := r.Log.WithValues("GenerateSelfSignedCertificate", "generating self-signed certificate")
	log.Info("Generating self-signed certificate..")

	// Generate a new private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Error(err, "Error while generating private key")
		return nil, nil, err
	}

	validity, err := r.getValidity(instance)
	if err != nil {
		log.Error(err, "Error while parsing validity")
		return nil, nil, err
	}

	// Create a template for the certificate
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: instance.Spec.DNSName,
		},
		DNSNames:  []string{instance.Spec.DNSName},
		NotBefore: time.Now(),
		NotAfter:  validity,
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
	}

	// Create the self-signed certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		log.Error(err, "Error while creating self-signed certificate")
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

// getValidity returns the time until the certificate expires
func (r *CertificateReconciler) getValidity(instance *certsv1.Certificate) (time.Time, error) {
	log := r.Log.WithValues("getValidity", "getting validity")
	log.Info("Getting validity..")

	validity := instance.Spec.Validity
	duration, err := time.ParseDuration(validity)
	if err != nil {
		log.Error(err, "Error while parsing validity duration")
		return time.Time{}, err
	}
	return time.Now().Add(duration), nil
}
