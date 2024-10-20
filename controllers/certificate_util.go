package controllers

import (
	"context"

	certsv1 "github.com/sheryarbutt/certificate-manager/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
			if volume.Secret != nil && volume.Secret.SecretName == instance.Spec.SecretRef {
				deploymentsWithMountedSecret = append(deploymentsWithMountedSecret, deployment)
				break
			}
		}
	}

	return deploymentsWithMountedSecret, nil
}

// addENVToDeployments adds an ENV to all deployments that use the secret and updates the deployment
func (r *CertificateReconciler) addENVToDeployments(ctx context.Context, req ctrl.Request, instance *certsv1.Certificate, secret *corev1.Secret) error {
	log := r.Log.WithValues("addENVToDeployments", instance.ObjectMeta.Name)
	log.Info("Adding ENV to deployments")

	deployments, err := r.getDeploymentsWithMountedSecret(ctx, req, instance)
	if err != nil {
		return err
	}

	for _, deployment := range deployments {
		// Update the deployment
		deploymentCopy := deployment.DeepCopy() // Copy the deployment to avoid modifying the original
		for i, container := range deploymentCopy.Spec.Template.Spec.Containers {
			deploymentCopy.Spec.Template.Spec.Containers[i].Env = append(container.Env, corev1.EnvVar{
				// Adding ResourceVersion helps identify if the secret has been updated and the deployment needs to be reloaded
				Name:  "CERTIFICATE_RESOURCE_VERSION",
				Value: secret.ResourceVersion,
			})
		}

		// Patch the deployment
		if err := r.Patch(ctx, deploymentCopy, client.MergeFrom(&deployment)); err != nil {
			return err
		}

	}
	return nil
}
