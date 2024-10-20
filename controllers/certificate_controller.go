/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	
	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	certsv1 "github.com/sheryarbutt/certificate-manager/api/v1"
	"github.com/sheryarbutt/certificate-manager/pkg/constants"
	"github.com/sheryarbutt/certificate-manager/pkg/utils"
)

// CertificateReconciler reconciles a Certificate object
type CertificateReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=certs.k8c.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=certs.k8c.io,resources=certificates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=certs.k8c.io,resources=certificates/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
func (r *CertificateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Initialize the log with the request namespace
	log := r.Log.WithValues("certificate", req.NamespacedName)

	// Fetch the Certificate instance
	instance := &certsv1.Certificate{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Certificate resource not found. Ignoring since object must be deleted")
			return utils.DoNotRequeue()
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Certificate")
		return utils.RequeueWithError(err)
	}

	// set status condition to reconciling
	patchBase := client.MergeFrom(instance.DeepCopy())
	instance.Status.Status = "Reconciling"
	if err := r.Status().Patch(ctx, instance, patchBase); err != nil {
		log.Error(err, "Failed to patch Certificate status")
		return utils.RequeueWithError(err)
	}

	// check if resource is marked for deletion
	if instance.DeletionTimestamp != nil {
		log.Info("Deletion timestamp found for instance " + req.Name)
		if instance.Spec.PurgeOnDelete {
			// update status to deleting
			patchBase = client.MergeFrom(instance.DeepCopy())
			instance.Status.Status = "Deleting"
			if err := r.Status().Patch(ctx, instance, patchBase); err != nil {
				log.Error(err, "Failed to patch Certificate status")
				return utils.RequeueWithError(err)
			}

			// Handle the delete logic
			err := r.handleDelete(ctx, req, instance)
			if err != nil {
				log.Error(err, "Failed to handle delete logic")
				return utils.RequeueWithError(err)
			}

			// remove finalizer
			controllerutil.RemoveFinalizer(instance, constants.Finalizer)
			if err := r.Update(ctx, instance); err != nil {
				log.Error(err, "Failed to remove finalizer from Certificate")
				return utils.RequeueWithError(err)
			}
		}
		return utils.DoNotRequeue()
	}

	// Handle the create/update logic
	err := r.handleCreate(ctx, req, instance)
	if err != nil {
		log.Error(err, "Failed to handle create/update logic")
		return utils.RequeueWithError(err)
	}

	// Set the status to deployed
	patchBase = client.MergeFrom(instance.DeepCopy())
	instance.Status.Status = "Deployed"
	instance.Status.Message = "Certificate deployed successfully"
	instance.Status.DeployedNamespace = instance.Namespace
	if err := r.Status().Patch(ctx, instance, patchBase); err != nil {
		log.Error(err, "Failed to patch Certificate status")
		return utils.RequeueWithError(err)
	}

	return utils.DoNotRequeue()
}

// SetupWithManager sets up the controller with the Manager.
func (r *CertificateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&certsv1.Certificate{}).
		Complete(r)
}
