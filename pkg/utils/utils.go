package utils

import (
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

// DoNotRequeue won't requeue a CR for reconciliation
func DoNotRequeue() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// RequeueWithError will requeue the CR for reconciliation with an error
func RequeueWithError(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}

// RequeueAfter will requeue the CR to be reconciled after a time duration
func RequeueAfter(requeueTime time.Duration) (ctrl.Result, error) {
	return ctrl.Result{Requeue: true, RequeueAfter: requeueTime}, nil
}
