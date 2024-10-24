package constants

const (
	// Certificate generation constants
	TypeCertificate = "CERTIFICATE"
	TypePrivateKey  = "RSA PRIVATE KEY"

	// Finalizer
	Finalizer = "certs.k8c.io/certificate"

	// Certificate status
	StatusReconciling = "Reconciling"
	StatusRotating    = "Rotating"
	StatusDeleting    = "Deleting"
	StatusExpired     = "Expired"
	StatusDeployed    = "Deployed"

	// Certificate status message
	StatusMessageReconciling = "Certificate is being processed"
	StatusMessageRotating    = "Certificate is expired, Regenerating.."
	StatusMessageDeleting    = "Certificate is being deleted"
	StatusMessageExpired     = "Certificate is expired"
	StatusMessageDeployed    = "Certificate deployed successfully"

	// Certificate ENV
	CertificateENVName = "CERTIFICATE_RESOURCE_VERSION"

	// Event types
	EventCreate = "CREATE"
	EventUpdate = "UPDATE"
)
