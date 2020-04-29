package internal

import "errors"

// ErrAccountNotFound error
var ErrAccountNotFound = errors.New("account/account-not-found")

// ErrAccountEmailAlreadyExists occurs when attempting to signup with an email that is in use.
var ErrAccountEmailAlreadyExists = errors.New("account/email-exists")

// ErrBucketCodeExists occurs when attempting to create a new bucket that already exists.
var ErrBucketCodeExists = errors.New("bucket/bucket-code-exists")

// ErrBucketNotFound error
var ErrBucketNotFound = errors.New("bucket/bucket-not-found")

// ErrLeadNotFound error
var ErrLeadNotFound = errors.New("lead/lead-not-found")

// ErrDeveloperKeyNotFound error
var ErrDeveloperKeyNotFound = errors.New("signin/developer-key-not-found")

// ErrBucketScheduledForDeletion sentinel value to indicate future bucket deletion.
var ErrBucketScheduledForDeletion = errors.New("bucket/bucket-scheduled-for-deletion")

// ErrWebhookURLExists sentinel error
var ErrWebhookURLExists = errors.New("webhook/webhook-url-exists")

// ErrWebhookCodeExists error
var ErrWebhookCodeExists = errors.New("webhook/webhook-code-exists")

// ErrWebhookNotFound error
var ErrWebhookNotFound = errors.New("webhook/webhook-not-found")

// ErrWebhookPermissionDenied error
var ErrWebhookPermissionDenied = errors.New("webhook/webhook-permission-denied")

const (
	// ErrCodeBadRequest is sent as the error code for 400 Bad Request.
	ErrCodeBadRequest string = "bad-request"

	// ErrCodeAuthenticationFailed occurs when the Authentication has failed.
	ErrCodeAuthenticationFailed string = "auth/authentication-failed"
)

// APIErrorResponse standard response format for RESTful API calls.
type APIErrorResponse struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

const (
	// ErrCodeAccountSignUp error code string
	ErrCodeAccountSignUp string = "accounts/signup-failed"

	// ErrCodeAccountNotFound error code string
	ErrCodeAccountNotFound string = "accounts/account-not-found"

	// ErrCodeEmailAlreadyExists errors code string
	ErrCodeEmailAlreadyExists string = "accounts/email-already-exists"

	// ErrCodeBucketCodeExists error code string.
	ErrCodeBucketCodeExists string = "buckets/bucket-code-exists"

	// ErrCodeBucketNotFound error code string.
	ErrCodeBucketNotFound string = "buckets/bucket-not-found"

	// ErrCodeLeadNotFound error code string.
	ErrCodeLeadNotFound string = "leads/lead-not-found"

	// ErrCodeWebhookURLExists error code string.
	ErrCodeWebhookURLExists string = "webhook/webhook-url-exists"

	// ErrCodeWebhookCodeExists error code string.
	ErrCodeWebhookCodeExists string = "webhook/webhook-code/exists"

	// ErrCodeWebhookNotFound error code string.
	ErrCodeWebhookNotFound string = "webhook/webhook-not-found"

	// ErrCodeWebhookUnknownEventTypes error code string.
	ErrCodeWebhookUnknownEventTypes string = "webhook/webhook-unknown-event-types"

	// ErrCodeWebhookResourcesNotFound error code string.
	ErrCodeWebhookResourcesNotFound string = "webhook/webhook-resources-not-found"
)
