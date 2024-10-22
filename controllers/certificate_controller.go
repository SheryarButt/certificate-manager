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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	certsv1 "github.com/sheryarbutt/certificate-manager/api/v1"
	"github.com/sheryarbutt/certificate-manager/pkg/constants"
	"github.com/sheryarbutt/certificate-manager/pkg/utils/k8s"
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
	log.Info("Request received to reconcile Certificate")

	// Fetch the Certificate instance
	log.Info("Fetching Certificate instance")
	instance := &certsv1.Certificate{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Certificate resource not found. Ignoring since object must be deleted")
			return k8s.DoNotRequeue()
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Certificate")
		return k8s.RequeueWithError(err)
	}

	// Set status condition to reconciling
	err := r.SetStatus(ctx, instance, "Reconciling", "Certificate is being reconciled", instance.Namespace, 0)
	if err != nil {
		log.Error(err, "Failed to set status to reconciling")
		return k8s.RequeueWithError(err)
	}

	// check if resource is marked for deletion
	log.Info("Checking if resource is marked for deletion")
	if instance.DeletionTimestamp != nil {
		log.Info("Deletion timestamp found for instance " + req.Name)
		if instance.Spec.PurgeOnDelete {
			// update status to deleting
			err := r.SetStatus(ctx, instance, "Deleting", "Certificate is being deleted", instance.Namespace, 0)
			if err != nil {
				log.Error(err, "Failed to set status to deleting")
				return k8s.RequeueWithError(err)
			}

			// Handle the delete logic
			err = r.handleDelete(ctx, req, instance)
			if err != nil {
				log.Error(err, "Failed to handle delete logic")
				return k8s.RequeueWithError(err)
			}

			// remove finalizer
			log.Info("Removing finalizer from Certificate")
			controllerutil.RemoveFinalizer(instance, constants.Finalizer)
			if err := r.Update(ctx, instance); err != nil {
				log.Error(err, "Failed to remove finalizer from Certificate")
				return k8s.RequeueWithError(err)
			}
		}
		return k8s.DoNotRequeue()
	}

	// Handle the create/update logic
	duration, err := r.handleCreate(ctx, req, instance)
	if err != nil {
		log.Error(err, "Failed to handle create/update logic")
		return k8s.RequeueWithError(err)
	}

	// Set the status to deployed
	err = r.SetStatus(ctx, instance, "Deployed", "Certificate deployed successfully", instance.Namespace, duration)
	if err != nil {
		log.Error(err, "Failed to set status to deployed")
		return k8s.RequeueWithError(err)
	}

	if instance.Spec.RotateOnExpiry {
		// Requeue after the specified duration to renew the certificate
		return k8s.RequeueAfter(duration)
	}

	return k8s.DoNotRequeue()
}

// SetupWithManager sets up the controller with the Manager.
func (r *CertificateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	statusUpdatePredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to the status subresource
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&certsv1.Certificate{}).
		WithEventFilter(statusUpdatePredicate).
		Owns(&corev1.Secret{}).
		Complete(r)
}
