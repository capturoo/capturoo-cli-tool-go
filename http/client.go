package http

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"capturoo-cli-tool-go/internal"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var timeout = time.Duration(6 * time.Second)

// Client HTTP client
type Client struct {
	endpoint string
	client   *http.Client
	JWT      string
}

// Account for capturoo.
type Account struct {
	Object               string `json:"object"`
	AccountID            string `json:"accountId" mapstructure:"accountId" yaml:"accountId"`
	UID                  string `json:"uid" mapstructure:"uid" yaml:"uid"`
	Role                 string `json:"role" mapstructure:"role" yaml:"role"`
	Email                string `json:"email" mapstructure:"email" yaml:"email"`
	DisplayName          string `json:"displayName" mapstructure:"displayName" yaml:"displayName"`
	DeveloperKey         string `json:"developerKey,omitempty" mapstructure:"developerKey,omitempty" yaml:"developerKey,omitempty"`
	scheduledForDeletion bool
	Created              time.Time `json:"created" mapstructure:"created,omitempty" yaml:"created,omitempty"`
	Modified             time.Time `json:"modified" mapstructure:"modified,omitempty" yaml:"modified,omitempty"`
}

// Bucket for storing leads.
type Bucket struct {
	Object               string `json:"object"`
	BucketID             string `json:"bucketId"`
	AccountID            string `json:"accountId"`
	ResourceName         string `json:"resourceName"`
	BucketName           string `json:"bucketName"`
	PublicAPIKey         string `json:"publicApiKey"`
	scheduledForDeletion bool
	Created              time.Time `json:"created"`
	Modified             time.Time `json:"modified"`
}

// Lead struct.
type Lead struct {
	LeadID   string                 `json:"leadId"`
	System   System                 `json:"system"`
	Data     map[string]interface{} `json:"data"`
	Tracking map[string]interface{} `json:"tracking"`
}

// System structured data about each lead captured.
type System struct {
	ClientVersion string    `json:"clientVersion"`
	Host          string    `json:"host"`
	Origin        string    `json:"Origin"`
	Referrer      string    `json:"referrer"`
	UserAgent     string    `json:"userAgent"`
	RemoteAddr    string    `json:"remoteAddr"`
	Created       time.Time `json:"created"`
}

// Webhook type
type Webhook struct {
	Object    string    `json:"object"`
	WebhookID string    `json:"webhookId"`
	Code      string    `json:"code"`
	Events    []string  `json:"events"`
	URL       string    `json:"url"`
	Enabled   bool      `json:"enabled"`
	Created   time.Time `json:"created"`
	Modified  time.Time `json:"modified"`
}

// FirebaseConfig type
type FirebaseConfig struct {
	APIKey            string `json:"apiKey"`
	AuthDomain        string `json:"authDomain"`
	DatabaseURL       string `json:"databaseURL"`
	ProjectID         string `json:"projectId"`
	StorageBucket     string `json:"storageBucket,omitempty"`
	MessagingSenderID string `json:"messagingSenderId,omitempty"`
	AppID             string `json:"appId,omitempty"`
}

// AutoConf type
type AutoConf struct {
	Object string `json:"object"`
	Data   struct {
		FirebaseConfig *FirebaseConfig `json:"firebaseConfig"`
	} `json:"data"`
}

// ErrBucketCodeExists occurs when attempting to create a new bucket that already exists.
var ErrBucketCodeExists = errors.New("bucket/bucket-code-exists")

// ErrWebhookURLExists error
var ErrWebhookURLExists = errors.New("webhook/webhook-url-exists")

// ErrWebhookCodeExists error
var ErrWebhookCodeExists = errors.New("webhook/webhook-code-exists")

// ErrWebhookResourcesNotFound error
var ErrWebhookResourcesNotFound = errors.New("webhook/webhook-resources-not-found")

// ErrBadRequest error
var ErrBadRequest = errors.New("bad-request")

// NewClient creates an HTTP client
func NewClient(endpoint string) *Client {
	tr := &http.Transport{
		MaxIdleConnsPerHost: 10,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}

	_, err := url.Parse(endpoint)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}

	return &Client{
		endpoint: endpoint,
		client:   client,
	}
}

func (c *Client) request(method, uri string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, uri, body)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.JWT)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "do HTTP %s request", req.Method)
	}
	return res, nil
}

// SignInWithDevKey exchanges a developer key for a custom token.
// https://www.googleapis.com/identitytoolkit/v3/relyingparty/verifyCustomToken?key=[API_KEY]
func (c *Client) SignInWithDevKey(key string) (token string, account *Account, err error) {
	uri := c.endpoint + "/signin-with-devkey"
	payload := struct {
		DeveloperKey string `json:"developerKey"`
	}{
		DeveloperKey: key,
	}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(payload)

	req, err := http.NewRequest("POST", uri, buf)
	if err != nil {
		return "", nil, errors.Wrap(err, "error creating new POST request")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return "", nil, errors.Wrapf(err, "error executing HTTP POST to %s", uri)
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return "", nil, errors.Wrapf(err, "statuc code %s", res.Status)
	}

	ct := struct {
		CustomToken string   `json:"customToken"`
		Account     *Account `json:"account"`
	}{}
	err = json.NewDecoder(res.Body).Decode(&ct)
	if err != nil {
		return "", nil, errors.Wrap(err, "custom token json decode error")
	}
	return ct.CustomToken, ct.Account, nil
}

// AutoConf retrieves the firebase public config.
func (c *Client) AutoConf(ctx context.Context) (*AutoConf, error) {
	url := c.endpoint + "/autoconf"
	res, err := c.request(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "request failed")
	}
	defer res.Body.Close()

	var autoconf AutoConf
	if err := json.NewDecoder(res.Body).Decode(&autoconf); err != nil {
		return nil, errors.Wrap(err, "json decode")
	}
	return &autoconf, nil
}

// CreateBucket create a new bucket.
func (c *Client) CreateBucket(ctx context.Context, accountID, bucketCode, bucketName string) (*Bucket, error) {
	uri := c.endpoint + "/buckets"
	payload := struct {
		AccountID  string `json:"accountId"`
		BucketCode string `json:"bucketCode"`
		BucketName string `json:"bucketName"`
	}{
		AccountID:  accountID,
		BucketCode: bucketCode,
		BucketName: bucketName,
	}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(payload)

	res, err := c.request(http.MethodPost, uri, buf)
	if err != nil {
		return nil, errors.Wrap(err, "request failed")
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return nil, errorResponse(res)
	}

	var bucket Bucket
	if err := json.NewDecoder(res.Body).Decode(&bucket); err != nil {
		return nil, errors.Wrap(err, "json decode")
	}
	return &bucket, nil
}

// GetBucket returns details of an individual bucket.
func (c *Client) GetBucket(ctx context.Context, bucketID string) (*Bucket, error) {
	url := fmt.Sprintf("%s/buckets/%s", c.endpoint, bucketID)
	res, err := c.request(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "request failed")
	}
	defer res.Body.Close()

	var bucket Bucket
	if err := json.NewDecoder(res.Body).Decode(&bucket); err != nil {
		return nil, errors.Wrap(err, "json decode")
	}
	return &bucket, nil
}

// GetBuckets returns a list of buckets from the server.
func (c *Client) GetBuckets(ctx context.Context, accountID string) ([]*Bucket, error) {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, nil
	}

	// build the URL including Query params
	v := url.Values{}
	v.Set("accountId", accountID)
	uri := url.URL{
		Scheme:     u.Scheme,
		Host:       fmt.Sprintf("%s:%s", u.Hostname(), u.Port()),
		Path:       "/buckets",
		ForceQuery: false,
		RawQuery:   v.Encode(),
	}

	res, err := c.request(http.MethodGet, uri.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "request failed: %w")
	}
	defer res.Body.Close()

	var container struct {
		Object string    `json:"object"`
		Data   []*Bucket `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&container); err != nil {
		return nil, errors.Wrap(err, "json decode")
	}
	return container.Data, nil
}

// DeleteBucket deletes a bucket or schedules it for deletion.
func (c *Client) DeleteBucket(ctx context.Context, bucketID string) error {
	url := fmt.Sprintf("%s/buckets/%s", c.endpoint, bucketID)
	res, err := c.request(http.MethodDelete, url, nil)
	if err != nil {
		return errors.Wrap(err, "request failed")
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return errorResponse(res)
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return nil

	}
	return errors.Wrapf(err, "delete bucket returned unknown status code (%d)", res.StatusCode)
}

// WriteLeads retrieves the leads from the API and immediately writes them
// to w.
func (c *Client) WriteLeads(ctx context.Context, format string, w io.Writer, bucketID string) error {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return nil
	}

	// build the URL including Query params
	v := url.Values{}
	v.Set("bucketId", bucketID)
	uri := url.URL{
		Scheme:     u.Scheme,
		Host:       fmt.Sprintf("%s:%s", u.Hostname(), u.Port()),
		Path:       "/leads",
		ForceQuery: false,
		RawQuery:   v.Encode(),
	}

	res, err := c.request(http.MethodGet, uri.String(), nil)
	if err != nil {
		return errors.Wrap(err, "request failed")
	}
	defer res.Body.Close()

	dec := json.NewDecoder(res.Body)
	// read "{"
	_, err = dec.Token()
	if err != nil {
		return err
	}

	// read "object" key
	_, err = dec.Token()
	if err != nil {
		return err
	}

	// read list value
	_, err = dec.Token()
	if err != nil {
		return err
	}

	// read "data" key
	_, err = dec.Token()
	if err != nil {
		return err
	}

	// read "[" delim
	_, err = dec.Token()
	if err != nil {
		return err
	}

	var enc interface{}
	if format == "json" {
		enc = json.NewEncoder(w)
	} else if format == "yaml" {
		enc = yaml.NewEncoder(w)
	} else if format == "csv" {
		enc = csv.NewWriter(w)
	} else {
		return errors.Wrapf(err, "format not supported (format=%s)", format)
	}

	for dec.More() {
		var lead Lead
		if err := dec.Decode(&lead); err != nil {
			return errors.Wrap(err, "json decode")
		}

		// encode the lead back to the write stream w.
		if format == "json" {
			if err := enc.(*json.Encoder).Encode(&lead); err != nil {
				return err
			}
		} else if format == "yaml" {
			if err := enc.(*yaml.Encoder).Encode(&lead); err != nil {
				return err
			}
		} else if format == "csv" {
			writer := enc.(*csv.Writer)
			fields := make([]string, 0)
			record, err := flattenLead(&lead, fields)
			if err != nil {
				return err
			}
			writer.Write(record)
			if writer.Error() != nil {
				return writer.Error()
			}
		}
	}

	if format == "csv" {
		writer := enc.(*csv.Writer)
		writer.Flush()
		if writer.Error() != nil {
			return writer.Error()
		}
	}

	// read "]" delim
	_, err = dec.Token()
	if err != nil {
		return err
	}

	// read "}" delim
	_, err = dec.Token()
	if err != nil {
		return err
	}

	return nil
}

// CreateWebhook creates a new webhook for the given webhook code, url and event types.
// equivilent to:
// curl -v -d '{"accountId":"89233482", "webhookCode":"my-webby-web-hook", "url":"https://webhook-plugin-test.capturoo.com/", "events": ["lead.created"], "enabled": true}' -H 'Content-Type: application/json' -H "Authorization: Bearer $JWT"  http://localhost:8080/webhooks
func (c *Client) CreateWebhook(ctx context.Context, accountID, code, url string, events []string, enabled bool) (*Webhook, error) {
	type requestBody struct {
		AccountID   string   `json:"accountId"`
		WebhookCode string   `json:"webhookCode"`
		URL         string   `json:"url"`
		Events      []string `json:"events"`
		Enabled     bool     `json:"enabled"`
	}

	uri := c.endpoint + "/webhooks"
	payload := &requestBody{
		AccountID:   accountID,
		WebhookCode: code,
		URL:         url,
		Events:      events,
		Enabled:     enabled,
	}

	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(payload)
	res, err := c.request(http.MethodPost, uri, buf)
	if err != nil {
		return nil, errors.Wrap(err, "post request failed")
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return nil, errorResponse(res)
	}

	var webhook Webhook
	if err := json.NewDecoder(res.Body).Decode(&webhook); err != nil {
		return nil, errors.Wrap(err, "json decode")
	}
	return &webhook, nil
}

// GetWebhooks returns a list of Webhooks for a given account.
func (c *Client) GetWebhooks(ctx context.Context, accountID string) ([]*Webhook, error) {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, nil
	}

	// build the URL including Query params
	v := url.Values{}
	v.Set("accountId", accountID)
	uri := url.URL{
		Scheme:     u.Scheme,
		Host:       fmt.Sprintf("%s:%s", u.Hostname(), u.Port()),
		Path:       "/webhooks",
		ForceQuery: false,
		RawQuery:   v.Encode(),
	}

	res, err := c.request(http.MethodGet, uri.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "request failed: %w")
	}
	defer res.Body.Close()

	var container struct {
		Object string     `json:"object"`
		Data   []*Webhook `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&container); err != nil {
		return nil, errors.Wrap(err, "json decode")
	}
	return container.Data, nil
}

// UpdateParamSet type
type UpdateParamSet struct {
	Events  *[]string `json:"events,omitempty"`
	URL     *string   `json:"url,omitempty"`
	Enabled *bool     `json:"enabled,omitempty"`
}

// UpdateWebhook does a partial update to the fields set in the UpdateParamSet. A field
// with a nil pointer is disregarded.
func (c *Client) UpdateWebhook(ctx context.Context, webhookID string, params *UpdateParamSet) (*Webhook, error) {
	type requestBody struct {
		Events  []string `json:"events,omitempty"`
		URL     string   `json:"url,omitempty"`
		Enabled *bool    `json:"enabled,omitempty"`
	}

	var payload requestBody
	if params.Events != nil {
		payload.Events = *params.Events
	}
	if params.URL != nil {
		payload.URL = *params.URL
	}
	if params.Enabled != nil {
		payload.Enabled = params.Enabled
	}

	fmt.Printf("%#v\n", payload)
	uri := c.endpoint + "/webhooks/" + webhookID
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(payload)
	res, err := c.request(http.MethodPatch, uri, buf)
	if err != nil {
		return nil, errors.Wrap(err, "post request failed")
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return nil, errorResponse(res)
	}

	var webhook Webhook
	if err := json.NewDecoder(res.Body).Decode(&webhook); err != nil {
		return nil, errors.Wrap(err, "json decode")
	}
	return &webhook, nil
}

// DeleteWebhook deletes the webhook with the given ID.
func (c *Client) DeleteWebhook(ctx context.Context, webhookID string) error {
	url := c.endpoint + "/webhooks/" + webhookID
	res, err := c.request(http.MethodDelete, url, nil)
	if err != nil {
		return errors.Wrap(err, "delete request failed")
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return errorResponse(res)
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return nil

	}
	return errors.Wrapf(err, "delete webhook returned unknown status code (%d)", res.StatusCode)
}

func errorResponse(res *http.Response) error {
	var badReqRes internal.APIErrorResponse
	err := json.NewDecoder(res.Body).Decode(&badReqRes)
	if badReqRes.Code == internal.ErrCodeBucketCodeExists {
		return ErrBucketCodeExists
	}
	if badReqRes.Code == internal.ErrCodeWebhookURLExists {
		return ErrWebhookURLExists
	}
	if badReqRes.Code == internal.ErrCodeBadRequest {
		return ErrBadRequest
	}
	if badReqRes.Code == internal.ErrCodeWebhookResourcesNotFound {
		return ErrWebhookResourcesNotFound
	}
	if badReqRes.Code == internal.ErrCodeBucketCodeExists {
		return ErrBucketCodeExists
	}
	if badReqRes.Code == internal.ErrCodeWebhookURLExists {
		return ErrWebhookURLExists
	}
	if badReqRes.Code == internal.ErrCodeWebhookCodeExists {
		return ErrWebhookCodeExists
	}
	if badReqRes.Code == internal.ErrCodeWebhookResourcesNotFound {
		return ErrWebhookResourcesNotFound
	}
	if badReqRes.Code == internal.ErrCodeBadRequest {
		return ErrBadRequest
	}
	if err != nil {
		return errors.Wrap(err, "decode failed")
	}
	return errors.Wrapf(err, "status=%s", res.Status)
}

func flattenLead(lead *Lead, fields []string) ([]string, error) {
	record := make([]string, 0)
	record = append(record, lead.LeadID)

	//fmt.Println("LeadID=", lead.LeadID)
	// fmt.Printf("%#v\n", lead.System)
	for k := range lead.Data {
		if !contains(fields, k) {
			fields = append(fields, k)
		}
	}

	for _, field := range fields {
		if val, ok := lead.Data[field]; ok {
			switch v := val.(type) {
			case int:
				record = append(record, strconv.Itoa(v))
			case float64:
				record = append(record, string(strconv.FormatFloat(v, 'f', -1, 64)))
			case bool:
				record = append(record, strconv.FormatBool(v))
			case string:
				record = append(record, v)
			default:
				return nil, fmt.Errorf("unsupported type detected (key=%v value=%v)", val, v)
			}

		} else {
			record = append(record, "")
		}

		//fmt.Printf("key[%s] value[%v]\n", field, lead.Data[field])
	}

	record = append(record, lead.System.ClientVersion, lead.System.Host, lead.System.Referrer, lead.System.UserAgent, lead.System.Created.String())
	return record, nil
}

// find returns the smallest index i at which x == a[i],
// or len(a) if there is no such index.
func find(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return len(a)
}

// contains tells whether a contains x.
func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
