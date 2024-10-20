package controllers

import (
	"context"

	certsv1 "github.com/sheryarbutt/certificate-manager/api/v1"
	"github.com/sheryarbutt/certificate-manager/pkg/objects"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *CertificateReconciler) handleDelete(ctx context.Context, req ctrl.Request, instance *certsv1.Certificate) error {
	log := r.Log.WithValues("certificate", req.NamespacedName)
	log.Info("Deleting Certificate: ", instance.ObjectMeta.Name)

	// Check if the certificate exists
	secret := objects.Secret(instance.Spec.SecretRef, instance.Namespace)
	err := r.Get(ctx, client.ObjectKeyFromObject(secret), secret)
	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Failed to get Secret")
		return err
	}

	// If the secret does not exist, return
	if errors.IsNotFound(err) {
		log.Info("Secret does not exist")
		return nil
	}

	// Delete the Secret
	err = r.Delete(ctx, secret)
	if err != nil {
		log.Error(err, "Failed to delete Secret")
		return err
	}

	return nil
}
