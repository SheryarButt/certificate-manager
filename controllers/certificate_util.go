package controllers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	certsv1 "github.com/sheryarbutt/certificate-manager/api/v1"
	"github.com/sheryarbutt/certificate-manager/pkg/constants"
)

// Event is a global variable to store the event type
var Event string

// getDeploymentsWithMountedSecret returns a list of Deployments that have the secret mounted
func (r *CertificateReconciler) getDeploymentsWithMountedSecret(ctx context.Context, req ctrl.Request, instance *certsv1.Certificate) ([]appsv1.Deployment, error) {
	log := r.Log.WithValues("getDeploymentsWithMountedSecret", instance.ObjectMeta.Name)
	log.Info("Getting deployments with mounted secret")

	deployments := &appsv1.DeploymentList{}
	err := r.List(ctx, deployments, client.InNamespace(req.Namespace))
	if err != nil {
		return nil, err
	}

	var deploymentsWithMountedSecret []appsv1.Deployment
	for _, deployment := range deployments.Items {
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			if volume.Secret != nil && volume.Secret.SecretName == instance.Spec.SecretRef.Name {
				deploymentsWithMountedSecret = append(deploymentsWithMountedSecret, deployment)
				break
			}
		}
	}

	return deploymentsWithMountedSecret, nil
}

// addEnvToDeployments adds an ENV to all deployments that use the secret and updates the deployment
func (r *CertificateReconciler) addEnvToDeployments(ctx context.Context, req ctrl.Request, instance *certsv1.Certificate, secret *corev1.Secret) error {
	log := r.Log.WithValues("addENVToDeployments", instance.ObjectMeta.Name)
	log.Info("ReloadOnChange is enabled, adding ENV to deployments")

	deployments, err := r.getDeploymentsWithMountedSecret(ctx, req, instance)
	if err != nil {
		return err
	}

	for _, deployment := range deployments {
		// Update the deployment
		deploymentCopy := deployment.DeepCopy() // Copy the deployment to avoid modifying the original
		for i, container := range deploymentCopy.Spec.Template.Spec.Containers {
			// 	// Adding ResourceVersion helps identify if the secret has been updated and the deployment needs to be reloaded
			var found bool
			for j, env := range container.Env {
				if env.Name == constants.CertificateENVName {
					// update the value of the env
					deploymentCopy.Spec.Template.Spec.Containers[i].Env[j].Value = secret.ResourceVersion
					found = true
					break
				}
			}
			if !found {
				deploymentCopy.Spec.Template.Spec.Containers[i].Env = append(container.Env, corev1.EnvVar{
					Name:  constants.CertificateENVName,
					Value: secret.ResourceVersion,
				})
			}
		}

		// Patch the deployment
		if err := r.Patch(ctx, deploymentCopy, client.MergeFrom(&deployment)); err != nil {
			return err
		}

	}
	return nil
}

// ParseDuration parses the validity duration from the Certificate spec
func parseDuration(validity string) (time.Duration, error) {
	// Check if the input ends with 'd' (for days)
	if strings.HasSuffix(validity, "d") {
		// Remove the 'd' suffix
		daysStr := strings.TrimSuffix(validity, "d")
		// Convert the string number of days to an integer
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return 0, fmt.Errorf("invalid duration format: %v", err)
		}
		// Convert days to hours and then to a time.Duration
		return time.Duration(days) * 24 * time.Hour, nil
	}
	// Otherwise, fall back to the standard time.ParseDuration function
	// This will parse durations like "1h", "1m", "1s", etc.
	return time.ParseDuration(validity)
}

// SetStatus sets the status of the Certificate instance
func (r *CertificateReconciler) SetStatus(ctx context.Context, instance *certsv1.Certificate, status string, message string, deployedNamespace string, expiryDate time.Duration) error {
	log := r.Log.WithValues("SetStatus", instance.ObjectMeta.Name)
	log.Info("Setting status to " + status)

	patchBase := client.MergeFrom(instance.DeepCopy())
	instance.Status.Status = status
	instance.Status.Message = message
	instance.Status.DeployedNamespace = deployedNamespace
	instance.Status.ExpiryDate = metav1.NewTime(time.Now().Add(expiryDate))
	if err := r.Status().Patch(ctx, instance, patchBase); err != nil {
		log.Error(err, "Failed to patch Certificate status")
		return err
	}

	return nil
}

// CreateOrUpdateSecret creates or updates the Secret object
func (r *CertificateReconciler) CreateOrUpdateSecret(ctx context.Context, secret *corev1.Secret) error {

	// Check if the secret already exists
	err := r.Get(ctx, client.ObjectKeyFromObject(secret), secret)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	// If the secret does not exist, create it
	if errors.IsNotFound(err) {
		return r.Create(ctx, secret)
	}

	// Otherwise, update the secret
	return r.Update(ctx, secret)
}

// MapSecretsToCertificates maps the secret names to the Certificate names
func MapSecretsToCertificates(object client.Object, client client.Client, log logr.Logger) []reconcile.Request {
	secret := object.(*corev1.Secret)

	// Get all Certificates
	certificates := &certsv1.CertificateList{}
	err := client.List(context.Background(), certificates)
	if err != nil {
		log.Error(err, "Failed to list Certificates")
		return nil
	}

	// Check if the secret is referenced by any Certificates
	for _, certificate := range certificates.Items {
		if certificate.Spec.SecretRef.Name == secret.Name {
			log.Info("Found Certificate referencing secret", "Certificate", certificate.Name)
			requests := []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      certificate.Name,
						Namespace: certificate.Namespace,
					},
				},
			}
			return requests
		}
	}

	return nil
}
