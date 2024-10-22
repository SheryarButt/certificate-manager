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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CertificateSpec defines the desired state of Certificate
type CertificateSpec struct {
	// DNSName is the DNS name for which the certificate should be issued
	DNSName string `json:"dnsName,omitempty"`

	// Validity the time until the certificate expires
	Validity string `json:"validity,omitempty"`

	// SecretRef is the reference to the secret where the certificate should be stored
	SecretRef SecretRef `json:"secretRef,omitempty"`

	// ReloadOnChange specifies if the deployment should be reloaded when the secret changes
	// +optional
	// +kubebuilder:default=false
	ReloadOnChange bool `json:"reloadOnChange,omitempty"`

	// PurgeOnDelete specifies if the secret should be deleted when the certificate is deleted
	// +optional
	// +kubebuilder:default=false
	PurgeOnDelete bool `json:"purgeOnDelete,omitempty"`

	// RotateOnExpiry specifies if the certificate should be rotated when it expires
	// +optional
	// +kubebuilder:default=false
	RotateOnExpiry bool `json:"rotateOnExpiry,omitempty"`
}

// SecretRef is a reference to a secret
type SecretRef struct {
	// Name is the name of the secret
	Name string `json:"name,omitempty"`
}

// CertificateStatus defines the observed state of Certificate
type CertificateStatus struct {
	// Status is the current status of the certificate
	Status string `json:"status,omitempty"`

	// Message is a human readable message indicating details about the certificate
	Message string `json:"message,omitempty"`

	// DeployedNamespace is the namespace where the certificate is deployed
	DeployedNamespace string `json:"deployedNamespace,omitempty"`

	// ExpiryDate is the date when the certificate expires
	ExpiryDate metav1.Time `json:"expiryDate,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:path=certificates,scope=Namespaced

// Certificate is the Schema for the certificates API
type Certificate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CertificateSpec   `json:"spec,omitempty"`
	Status CertificateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CertificateList contains a list of Certificate
type CertificateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Certificate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Certificate{}, &CertificateList{})
}
