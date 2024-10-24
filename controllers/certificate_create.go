package controllers

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	certsv1 "github.com/sheryarbutt/certificate-manager/api/v1"
	"github.com/sheryarbutt/certificate-manager/pkg/constants"
	"github.com/sheryarbutt/certificate-manager/pkg/objects"
	"github.com/sheryarbutt/certificate-manager/pkg/utils"
	"github.com/sheryarbutt/certificate-manager/pkg/utils/cert"
)

func (r *CertificateReconciler) handleCreate(ctx context.Context, req ctrl.Request, instance *certsv1.Certificate) (time.Duration, error) {
	log := r.Log.WithValues("certificate", req.NamespacedName)
	log.Info("Creating/Updating Certificate")

	// Check if the certificate already exists
	log.Info("Checking if the Secret exists")
	secret := objects.Secret(instance.Spec.SecretRef.Name, req.Namespace)
	err := r.Get(ctx, client.ObjectKeyFromObject(secret), secret)
	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Failed to get Secret")
		return 0, err
	}

	// If the secret does not exist, create it
	if errors.IsNotFound(err) || Event == constants.EventUpdate {
		log.Info("Secret does not exist, creating..")
		// Generate a self-signed certificate
		cert, key, err := r.GenerateSelfSignedCertificate(ctx, instance)
		if err != nil {
			log.Error(err, "Failed to generate self-signed certificate")
			return 0, err
		}

		// Create the Secret object
		secret := objects.Secret(instance.Spec.SecretRef.Name, instance.Namespace)
		secret.Type = corev1.SecretTypeTLS
		secret.Data = map[string][]byte{
			"tls.crt": cert,
			"tls.key": key,
		}

		// Create the Secret
		log.Info("Creating Secret")
		err = r.CreateOrUpdateSecret(ctx, secret)
		if err != nil {
			log.Error(err, "Failed to create Secret")
			return 0, err
		}

		// Set owner reference on the Secret
		log.Info("Setting owner reference on Secret")
		err = controllerutil.SetOwnerReference(instance, secret, r.Scheme)
		if err != nil {
			log.Error(err, "Failed to set owner reference on Secret")
			return 0, err
		}

		if instance.Spec.PurgeOnDelete {
			// Add finalizer to the Certificate
			log.Info("PurgeOnDelete is enabled, adding finalizer to Certificate")
			controllerutil.AddFinalizer(instance, constants.Finalizer)
			if err := r.Update(ctx, instance); err != nil {
				log.Error(err, "Failed to add finalizer to Certificate")
				return 0, err
			}
		}

		if instance.Spec.ReloadOnChange {
			// Add Env to deployments that use this secret
			// This will reload the deployments that are using this secret
			if err := r.addEnvToDeployments(ctx, req, instance, secret); err != nil {
				log.Error(err, "Failed to add env to deployments")
				return 0, err
			}
		}
	} else {
		// If secret already exists, check if the certificate is expired or not
		log.Info("Secret exists, checking if certificate is expired..")
		tlsCert, ok := secret.Data["tls.crt"]
		if !ok {
			log.Info("Secret does not contain tls.crt key")
			return 0, nil
		}

		if expired, err := cert.IsCertificateExpired(tlsCert); err != nil {
			log.Error(err, "Failed to check if certificate is expired")
			return 0, err
		} else if expired {
			if instance.Spec.RotateOnExpiry {
				log.Info("Certificate is expired, Regenerating..")

				// Set the status to "Rotating"
				err := r.SetStatus(ctx, instance, constants.StatusRotating, constants.StatusMessageRotating, req.Namespace, 0)
				if err != nil {
					log.Error(err, "Failed to set status")
					return 0, err
				}

				// Generate a new self-signed certificate
				cert, key, err := r.GenerateSelfSignedCertificate(ctx, instance)
				if err != nil {
					log.Error(err, "Failed to generate self-signed certificate")
					return 0, err
				}

				// Update the Secret with the new certificate
				secret.Data["tls.crt"] = cert
				secret.Data["tls.key"] = key

				// Update the Secret
				err = r.CreateOrUpdateSecret(ctx, secret)
				if err != nil {
					log.Error(err, "Failed to update Secret")
					return 0, err
				}

				if instance.Spec.ReloadOnChange {
					// Add Env to deployments that use this secret
					// This will reload the deployments using this secret
					if err := r.addEnvToDeployments(ctx, req, instance, secret); err != nil {
						log.Error(err, "Failed to add env to deployments")
						return 0, err
					}
				}
			} else {
				log.Info("Certificate is expired but RotateOnExpiry is disabled")
				// Set the status to "Expired"
				err := r.SetStatus(ctx, instance, constants.StatusExpired, constants.StatusMessageExpired, req.Namespace, 0)
				if err != nil {
					log.Error(err, "Failed to set status")
					return 0, err
				}
			}
		}
	}
	return utils.ParseDuration(instance.Spec.Validity)
}

// generateCertificate generates a self-signed certificate for the given DNS name
func (r *CertificateReconciler) GenerateSelfSignedCertificate(ctx context.Context, instance *certsv1.Certificate) ([]byte, []byte, error) {
	log := r.Log.WithValues("GenerateSelfSignedCertificate", "generating self-signed certificate")
	log.Info("Generating self-signed certificate..")

	validity, err := utils.ParseDuration(instance.Spec.Validity)
	if err != nil {
		log.Error(err, "Error while parsing validity")
		return nil, nil, err
	}

	return cert.CreateSelfSignedCertificate(instance.Spec.DNSName, validity)
}
