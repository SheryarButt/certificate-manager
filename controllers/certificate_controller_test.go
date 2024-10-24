package controllers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	certsv1 "github.com/sheryarbutt/certificate-manager/api/v1"
	"github.com/sheryarbutt/certificate-manager/pkg/constants"
)

func TestCertificateController(t *testing.T) {
	t.Run("CreateBasicCertificate", TestCreateBasicCertificate)
	t.Run("DeleteDeployedSecret", TestDeleteDeployedSecret)
	t.Run("CertificateWithPurgeOnDelete", TestCertificateWithPurgeOnDelete)
	t.Run("CertificateWithPurgeOnDeleteSetToFalse", TestCertificateWithPurgeOnDeleteSetToFalse)
	t.Run("CertificateWithReloadOnChange", TestCertificateWithReloadOnChange)
	t.Run("CertificateWithReloadOnChangeSetToFalse", TestCertificateWithReloadOnChangeSetToFalse)
	t.Run("CertificateWithRotateOnExpiry", TestCertificateWithRotateOnExpiry)
	t.Run("CertificateWithRotateOnExpirySetToFalse", TestCertificateWithRotateOnExpirySetToFalse)
	t.Run("CertificateWithRotateOnExpiryAndReloadOnChange", TestCertificateWithRotateOnExpiryAndReloadOnChange)
}

// setupTestEnv sets up the test environment for the Certificate controller
func setupTestEnv() *CertificateReconciler {
	// Setup the test environment
	scheme := runtime.NewScheme()
	_ = certsv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := &CertificateReconciler{
		Client: fakeClient,
		Log:    zap.New(zap.UseDevMode(true)),
		Scheme: scheme,
	}

	return r
}

// TestCreateBasicCertificate tests the creation of a basic Certificate instance and the corresponding Secret
func TestCreateBasicCertificate(t *testing.T) {
	// Setup the test environment
	r := setupTestEnv()

	// Create a Certificate instance
	instance := getCertificateTemplate("test-certificate", "default", "test-secret", "1h", false, false, false)

	err := r.Create(context.Background(), instance)
	assert.NoError(t, err, "Certificate instance should be created")

	err = triggerReconcile(r, "test-certificate", "default")
	assert.NoError(t, err, "Reconcile should not return an error")

	// Check status of the Certificate instance
	certificate := &certsv1.Certificate{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-certificate", Namespace: "default"}, certificate)
	assert.NoError(t, err, "Certificate instance should exist")
	assert.Equal(t, certificate.Status.Status, constants.StatusDeployed, "Certificate status should be deployed")
	assert.Equal(t, certificate.Status.Message, constants.StatusMessageDeployed, "Certificate message should be deployed")
	assert.Equal(t, certificate.Status.DeployedNamespace, "default", "Certificate should be deployed in the default namespace")

	// Get the secret created by the Certificate instance
	secret := &corev1.Secret{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
	assert.NoError(t, err, "Secret should be created")

	// Check the secret data
	assert.Contains(t, secret.Data, "tls.crt", "Secret should contain tls.crt")
	assert.Contains(t, secret.Data, "tls.key", "Secret should contain tls.key")
}

// TestDeleteDeployedSecret tests the deletion of a deployed Secret
// The Certificate controller should recreate the Secret if it is deleted
func TestDeleteDeployedSecret(t *testing.T) {
	// Setup the test environment
	r := setupTestEnv()

	// Create a Certificate instance
	instance := getCertificateTemplate("test-certificate", "default", "test-secret", "1h", false, false, false)

	err := r.Create(context.Background(), instance)
	assert.NoError(t, err, "Certificate instance should be created")

	err = triggerReconcile(r, "test-certificate", "default")
	assert.NoError(t, err, "Reconcile should not return an error")

	// Get the secret created by the Certificate instance
	secret := &corev1.Secret{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
	assert.NoError(t, err, "Secret should be created")

	// Delete the secret
	err = r.Delete(context.Background(), secret)
	assert.NoError(t, err, "Secret should be deleted")

	err = triggerReconcile(r, "test-certificate", "default")
	assert.NoError(t, err, "Reconcile should not return an error")

	// Check if the secret is recreated
	secret = &corev1.Secret{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
	assert.NoError(t, err, "Secret should be recreated")
}

// TestCertificateWithPurgeOnDelete tests the deletion of a Certificate instance with PurgeOnDelete set to true
// The corresponding Secret should be deleted when the Certificate instance is deleted
func TestCertificateWithPurgeOnDelete(t *testing.T) {
	// Setup the test environment
	r := setupTestEnv()

	// Create a Certificate instance
	instance := getCertificateTemplate("test-certificate", "default", "test-secret", "1h", true, false, false)

	err := r.Create(context.Background(), instance)
	assert.NoError(t, err, "Certificate instance should be created")

	err = triggerReconcile(r, "test-certificate", "default")
	assert.NoError(t, err, "Reconcile should not return an error")

	// Get the secret created by the Certificate instance
	secret := &corev1.Secret{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
	assert.NoError(t, err, "Secret should be created")

	// Delete the Certificate instance
	err = r.Delete(context.Background(), instance)
	assert.NoError(t, err, "Certificate instance should be deleted")

	err = triggerReconcile(r, "test-certificate", "default")
	assert.NoError(t, err, "Reconcile should not return an error")

	// Check if the secret is deleted
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
	assert.Error(t, err, "Secret should be deleted")
}

// TestCertificateWithPurgeOnDeleteSetToFalse tests the deletion of a Certificate instance with PurgeOnDelete set to false
// The corresponding Secret should not be deleted when the Certificate instance is deleted
func TestCertificateWithPurgeOnDeleteSetToFalse(t *testing.T) {
	// Setup the test environment
	r := setupTestEnv()

	// Create a Certificate instance
	instance := getCertificateTemplate("test-certificate", "default", "test-secret", "1h", false, false, false)

	err := r.Create(context.Background(), instance)
	assert.NoError(t, err, "Certificate instance should be created")

	err = triggerReconcile(r, "test-certificate", "default")
	assert.NoError(t, err, "Reconcile should not return an error")

	// Get the secret created by the Certificate instance
	secret := &corev1.Secret{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
	assert.NoError(t, err, "Secret should be created")

	// Delete the Certificate instance
	err = r.Delete(context.Background(), instance)
	assert.NoError(t, err, "Certificate instance should be deleted")

	err = triggerReconcile(r, "test-certificate", "default")
	assert.NoError(t, err, "Reconcile should not return an error")

	// Check if the secret is deleted
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
	assert.NoError(t, err, "Secret should not be deleted")
}

// TestCertificateWithReloadOnChange tests the reloading of a Deployment when the Certificate instance is updated
// The Deployment should be updated with the new certificate ENV when the Certificate instance is updated
func TestCertificateWithReloadOnChange(t *testing.T) {
	// Setup the test environment
	r := setupTestEnv()

	// Create a Certificate instance
	instance := getCertificateTemplate("test-certificate", "default", "test-secret", "1h", false, true, false)

	err := r.Create(context.Background(), instance)
	assert.NoError(t, err, "Certificate instance should be created")

	// Create a Deployment instance
	deployment := getDeploymentTemplate("test-deployment", "default", "test-secret")

	err = r.Create(context.Background(), deployment)
	assert.NoError(t, err, "Deployment should be created")

	err = triggerReconcile(r, "test-certificate", "default")
	assert.NoError(t, err, "Reconcile should not return an error")

	// Check the deployment for inserted ENV
	checkDeployment := &appsv1.Deployment{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-deployment", Namespace: "default"}, checkDeployment)
	assert.NoError(t, err, "Deployment should be updated")

	value, err := checkIfCertificateEnvExists(checkDeployment)
	assert.NoError(t, err, "Certificate ENV should be inserted")

	// Get the secret created by the Certificate instance
	secret := &corev1.Secret{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
	assert.NoError(t, err, "Secret should be created")

	// Check the resourceVersion of the secret matches the one in the deployment
	assert.Equal(t, secret.ResourceVersion, value, "ResourceVersion should match")

	// Update the Certificate instance
	certificate := &certsv1.Certificate{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-certificate", Namespace: "default"}, certificate)
	assert.NoError(t, err, "Certificate instance should exist")

	certificate.Spec.Validity = "2h"
	err = r.Update(context.Background(), certificate)
	assert.NoError(t, err, "Certificate instance should be updated")

	err = triggerReconcile(r, "test-certificate", "default")
	assert.NoError(t, err, "Reconcile should not return an error")

	// Check the deployment for updated ENV
	checkDeployment = &appsv1.Deployment{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-deployment", Namespace: "default"}, checkDeployment)
	assert.NoError(t, err, "Deployment should exist")

	value, err = checkIfCertificateEnvExists(checkDeployment)
	assert.NoError(t, err, "Certificate ENV should be updated")

	// Get the secret created by the Certificate instance
	secret = &corev1.Secret{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
	assert.NoError(t, err, "Secret should exist")

	// Check the resourceVersion of the secret matches the one in the deployment
	assert.Equal(t, secret.ResourceVersion, value, "ResourceVersion should match")
}

// TestCertificateWithReloadOnChangeSetToFalse tests the reloading of a Deployment when the Certificate instance is updated
// The Deployment should not be updated with the new certificate ENV when the Certificate instance is updated
func TestCertificateWithReloadOnChangeSetToFalse(t *testing.T) {
	// Setup the test environment
	r := setupTestEnv()

	// Create a Certificate instance
	instance := getCertificateTemplate("test-certificate", "default", "test-secret", "1h", false, false, false)

	err := r.Create(context.Background(), instance)
	assert.NoError(t, err, "Certificate instance should be created")

	// Create a Deployment instance
	deployment := getDeploymentTemplate("test-deployment", "default", "test-secret")

	err = r.Create(context.Background(), deployment)
	assert.NoError(t, err, "Deployment should be created")

	err = triggerReconcile(r, "test-certificate", "default")
	assert.NoError(t, err, "Reconcile should not return an error")

	// Check the deployment for inserted ENV
	checkDeployment := &appsv1.Deployment{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-deployment", Namespace: "default"}, checkDeployment)
	assert.NoError(t, err, "Deployment should be updated")

	_, err = checkIfCertificateEnvExists(checkDeployment)
	assert.Error(t, err, "Certificate ENV should not be inserted")

	// Save the resourceVersion of the deployment
	oldResourceVersion := checkDeployment.ResourceVersion

	// Update the Certificate instance
	certificate := &certsv1.Certificate{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-certificate", Namespace: "default"}, certificate)
	assert.NoError(t, err, "Certificate instance should exist")

	certificate.Spec.Validity = "2h"
	err = r.Update(context.Background(), certificate)
	assert.NoError(t, err, "Certificate instance should be updated")

	err = triggerReconcile(r, "test-certificate", "default")
	assert.NoError(t, err, "Reconcile should not return an error")

	// Check the deployment for updated ENV
	checkDeployment = &appsv1.Deployment{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-deployment", Namespace: "default"}, checkDeployment)
	assert.NoError(t, err, "Deployment should exist")

	_, err = checkIfCertificateEnvExists(checkDeployment)
	assert.Error(t, err, "Certificate ENV should not exist")

	// Check if the deployment resourceVersion has been updated, indicating that the deployment was not reloaded
	assert.Equal(t, oldResourceVersion, checkDeployment.ResourceVersion, "ResourceVersion should not be updated")
}

// TestCertificateWithRotateOnExpiry tests the rotation of a certificate when it expires
// The Certificate controller should update the Secret with a new certificate when the old one expires
func TestCertificateWithRotateOnExpiry(t *testing.T) {
	// Setup the test environment
	r := setupTestEnv()

	// Create a Certificate instance
	instance := getCertificateTemplate("test-certificate", "default", "test-secret", "5s", false, false, true)

	err := r.Create(context.Background(), instance)
	assert.NoError(t, err, "Certificate instance should be created")

	err = triggerReconcile(r, "test-certificate", "default")
	assert.NoError(t, err, "Reconcile should not return an error")

	// Get the secret created by the Certificate instance
	secret := &corev1.Secret{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
	assert.NoError(t, err, "Secret should be created")

	// Save resource version of the secret
	oldResourceVersion := secret.ResourceVersion

	// Wait for the certificate to expire
	time.Sleep(5 * time.Second)

	err = triggerReconcile(r, "test-certificate", "default")
	assert.NoError(t, err, "Reconcile should not return an error")

	// Get the secret created by the Certificate instance
	secret = &corev1.Secret{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
	assert.NoError(t, err, "Secret should be updated")

	// Check if the secret generation has been updated
	assert.NotEqual(t, oldResourceVersion, secret.ResourceVersion, "ResourceVersion should be updated")
}

// TestCertificateWithRotateOnExpirySetToFalse tests the rotation of a certificate when it expires
// The Certificate controller should not update the Secret with a new certificate when the old one expires and update the status of the Certificate instance
func TestCertificateWithRotateOnExpirySetToFalse(t *testing.T) {
	// Setup the test environment
	r := setupTestEnv()

	// Create a Certificate instance
	instance := getCertificateTemplate("test-certificate", "default", "test-secret", "5s", false, false, false)

	err := r.Create(context.Background(), instance)
	assert.NoError(t, err, "Certificate instance should be created")

	err = triggerReconcile(r, "test-certificate", "default")
	assert.NoError(t, err, "Reconcile should not return an error")

	// Get the secret created by the Certificate instance
	secret := &corev1.Secret{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
	assert.NoError(t, err, "Secret should be created")

	// Save resource version of the secret
	oldResourceVersion := secret.ResourceVersion

	// Wait for the certificate to expire
	time.Sleep(5 * time.Second)

	err = triggerReconcile(r, "test-certificate", "default")
	assert.NoError(t, err, "Reconcile should not return an error")

	// Get the secret created by the Certificate instance
	secret = &corev1.Secret{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
	assert.NoError(t, err, "Secret should exist")

	// Check if the secret generation has been updated
	assert.Equal(t, oldResourceVersion, secret.ResourceVersion, "ResourceVersion should not be updated")

	// Check status of the Certificate instance
	certificate := &certsv1.Certificate{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-certificate", Namespace: "default"}, certificate)
	assert.NoError(t, err, "Certificate instance should exist")
	assert.Equal(t, certificate.Status.Status, constants.StatusExpired, "Certificate status should be Expired")
	assert.Equal(t, certificate.Status.Message, constants.StatusMessageExpired, "Certificate message should be Expired")
}

// TestCertificateWithRotateOnExpiryAndReloadOnChange tests the rotation of a certificate when it expires
// The Certificate controller should update the Secret with a new certificate when the old one expires and update the Deployment with the new certificate ENV
func TestCertificateWithRotateOnExpiryAndReloadOnChange(t *testing.T) {
	// Setup the test environment
	r := setupTestEnv()

	// Create a Certificate instance
	instance := getCertificateTemplate("test-certificate", "default", "test-secret", "5s", false, true, true)

	err := r.Create(context.Background(), instance)
	assert.NoError(t, err, "Certificate instance should be created")

	// Create a Deployment instance
	deployment := getDeploymentTemplate("test-deployment", "default", "test-secret")

	err = r.Create(context.Background(), deployment)
	assert.NoError(t, err, "Deployment should be created")

	err = triggerReconcile(r, "test-certificate", "default")
	assert.NoError(t, err, "Reconcile should not return an error")

	// Check the deployment for inserted ENV
	checkDeployment := &appsv1.Deployment{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-deployment", Namespace: "default"}, checkDeployment)
	assert.NoError(t, err, "Deployment should be updated")

	valueBeforeRotation, err := checkIfCertificateEnvExists(checkDeployment)
	assert.NoError(t, err, "Certificate ENV should be inserted")

	// Get the secret created by the Certificate instance
	secret := &corev1.Secret{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
	assert.NoError(t, err, "Secret should be created")

	// Resouece version of the secret should match the one in the deployment
	assert.Equal(t, secret.ResourceVersion, valueBeforeRotation, "ResourceVersion should match")

	// Save resource version of the secret
	oldResourceVersion := secret.ResourceVersion

	// Wait for the certificate to expire
	time.Sleep(5 * time.Second)

	err = triggerReconcile(r, "test-certificate", "default")
	assert.NoError(t, err, "Reconcile should not return an error")

	// Check the deployment for updated ENV
	checkDeployment = &appsv1.Deployment{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-deployment", Namespace: "default"}, checkDeployment)
	assert.NoError(t, err, "Deployment should exist")

	valueAfterRotation, err := checkIfCertificateEnvExists(checkDeployment)
	assert.NoError(t, err, "Certificate ENV should be updated")

	// ENV should be updated after certificate rotation
	assert.NotEqual(t, valueBeforeRotation, valueAfterRotation, "Certificate ENV should be updated")

	// Get the secret created by the Certificate instance
	secret = &corev1.Secret{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
	assert.NoError(t, err, "Secret should be updated")

	// Check if the secret generation has been updated
	assert.NotEqual(t, oldResourceVersion, secret.ResourceVersion, "ResourceVersion should be updated")

	// Check if the secret resourceVersion matches the one in the deployment
	assert.Equal(t, secret.ResourceVersion, valueAfterRotation, "ResourceVersion should match")
}

// triggerReconcile triggers the Reconcile function of the Certificate controller
func triggerReconcile(r *CertificateReconciler, name, namespace string) error {
	_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: name, Namespace: namespace}})
	return err
}

// checkIfCertificateEnvExists checks if the certificate ENV exists in the deployment
func checkIfCertificateEnvExists(deployment *appsv1.Deployment) (string, error) {
	for _, container := range deployment.Spec.Template.Spec.Containers {
		for _, env := range container.Env {
			if env.Name == constants.CertificateENVName {
				return env.Value, nil
			}
		}
	}
	return "", errors.New("Certificate ENV not found")
}

// getCertificateTemplate returns a basic Certificate instance template
func getCertificateTemplate(name, namespace, secretName, validity string, PurgeOnDelete, ReloadOnChange, RotateOnExpiry bool) *certsv1.Certificate {
	return &certsv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: certsv1.CertificateSpec{
			SecretRef: certsv1.SecretRef{
				Name: secretName,
			},
			Validity:       validity,
			PurgeOnDelete:  PurgeOnDelete,
			ReloadOnChange: ReloadOnChange,
			RotateOnExpiry: RotateOnExpiry,
		},
	}
}

// getDeploymentTemplate returns a basic Deployment instance template
func getDeploymentTemplate(name, namespace, secretName string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "test-container",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "test-volume",
									MountPath: "/etc/secret-volume",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "test-volume",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: secretName,
								},
							},
						},
					},
				},
			},
		},
	}
}
