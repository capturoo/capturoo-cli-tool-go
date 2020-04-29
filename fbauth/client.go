package fbauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var test = "test string another"

var timeout = time.Duration(6 * time.Second)

// Client HTTP client
type RESTClient struct {
	client *http.Client
}

// TokenAndRefreshToken contains a pair of JTW and refresh token for Firebase.
type TokenAndRefreshToken struct {
	IDToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
}

// See https://firebase.google.com/docs/reference/rest/auth/#section-verify-custom-token
// token	      string   A Firebase Auth custom token from which to create an ID and refresh token pair.
// returnSecureToken  boolean  Whether or not to return an ID and refresh token. Should always be true.
type verifyCustomTokenRequest struct {
	Token             string `json:"token"`
	ReturnSecureToken bool   `json:"returnSecureToken"`
}

type verifyCustomTokenResponse struct {
	Kind         string `json:"kind"`
	IDToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    string `json:"expiresIn"`
}

// example
// curl -H 'Content-Type: application/x-www-form-urlencoded' -X POST --data 'grant_type=refresh_token&refresh_token=<refreshToken>' https://securetoken.googleapis.com/v1/token?key=<firebasePublicAPIKey>
// grant_type	  string  The refresh token's grant type, always "refresh_token".
// refresh_token  string  A Firebase Auth refresh token.
type exchangeRefreshTokenRequest struct {
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`
}

// NewClient creates an HTTP client
func NewRESTClient() *RESTClient {
	tr := &http.Transport{
		MaxIdleConnsPerHost: 10,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}
	return &RESTClient{
		client: client,
	}
}

// https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=[API_KEY]

type SignInResponse struct {
	// A Firebase Auth ID token for the authenticated user.
	IDToken string

	// The email for the authenticated user.
	Email string

	// A Firebase Auth refresh token for the authenticated user.
	RefreshToken string

	// The number of seconds in which the ID token expires.
	ExpiresIn string

	// The uid of the authenticated user.
	LocalId string

	// Whether the email is for an existing account.
	Registered bool
}

type badRequestResponse struct {
	Error struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
		Errors  []struct {
			Message string `json:"message"`
			Domain  string `json:"domain"`
			Reason  string `json:"reason"`
		} `json:"errors,omitempty"`
		Status string `json:"status,omitempty"`
	} `json:"error"`
}

// SignInWithEmailAndPassword calls the Firebase REST API to sign in using email and password.
func (c *RESTClient) SignInWithEmailAndPassword(firebaseAPIKey, email, password string) (*SignInResponse, error) {
	// build the URL including Query params
	v := url.Values{}
	v.Set("key", firebaseAPIKey)
	uri := url.URL{
		Scheme:     "https",
		Host:       "identitytoolkit.googleapis.com",
		Path:       "v1/accounts:signInWithPassword",
		ForceQuery: false,
		RawQuery:   v.Encode(),
	}

	// build and excute the request
	type payload struct {
		// The email the user is signing in with.
		Email string `json:"email"`

		// The password for the account.
		Password string `json:"password"`

		// Whether or not to return an ID and refresh token. Should always be true.
		ReturnSecureToken bool `json:"returnSecureToken"`
	}

	reqBody := payload{
		Email:             email,
		Password:          password,
		ReturnSecureToken: true,
	}

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(reqBody)
	if err != nil {
		return nil, fmt.Errorf("json encode failed: %w", err)
	}
	req, err := http.NewRequest("POST", uri.String(), buf)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("creating new POST request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 400 {
		var badReqRes badRequestResponse
		err = json.NewDecoder(res.Body).Decode(&badReqRes)
		if err != nil {
			return nil, fmt.Errorf("decode failed: %w", err)
		}
		return nil, fmt.Errorf("%d %s", badReqRes.Error.Code, badReqRes.Error.Message)
	} else if res.StatusCode > 400 {
		return nil, fmt.Errorf("%s", res.Status)
	}

	var signInRes SignInResponse
	err = json.NewDecoder(res.Body).Decode(&signInRes)
	if err != nil {
		return nil, fmt.Errorf("json decode failed: %w", err)
	}
	return &signInRes, nil
}

// ExchangeCustomTokenForIDAndRefreshToken calls the Firebase REST API to exchange a customer token for Firebase token and refresh token.
func (c *RESTClient) ExchangeCustomTokenForIDAndRefreshToken(firebaseAPIKey, token string) (*TokenAndRefreshToken, error) {
	// build the URL including Query params
	v := url.Values{}
	v.Set("key", firebaseAPIKey)
	uri := url.URL{
		Scheme:     "https",
		Host:       "www.googleapis.com",
		Path:       "identitytoolkit/v3/relyingparty/verifyCustomToken",
		ForceQuery: false,
		RawQuery:   v.Encode(),
	}

	// build and execute the request
	reqBody := verifyCustomTokenRequest{
		Token:             token,
		ReturnSecureToken: true,
	}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(reqBody)
	req, err := http.NewRequest("POST", uri.String(), buf)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("creating new POST request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 400 {
		var badReqRes badRequestResponse
		err = json.NewDecoder(res.Body).Decode(&badReqRes)
		if err != nil {
			return nil, fmt.Errorf("decode failed: %w", err)
		}
		return nil, fmt.Errorf("%d %s", badReqRes.Error.Code, badReqRes.Error.Message)
	} else if res.StatusCode > 400 {
		return nil, fmt.Errorf("%s", res.Status)
	}

	tokenResponse := verifyCustomTokenResponse{}
	err = json.NewDecoder(res.Body).Decode(&tokenResponse)
	if err != nil {
		return nil, fmt.Errorf("json decode failed: %w", err)
	}
	return &TokenAndRefreshToken{
		IDToken:      tokenResponse.IDToken,
		RefreshToken: tokenResponse.RefreshToken,
	}, nil
}

// ExchangeRefreshTokenForIDToken calls Google's REST API.
// Response Payload
// Property Name	Type	Description
// expires_in	string	The number of seconds in which the ID token expires.
// token_type	string	The type of the refresh token, always "Bearer".
// refresh_token	string	The Firebase Auth refresh token provided in the request or a new refresh token.
// id_token	string	A Firebase Auth ID token.
// user_id	string	The uid corresponding to the provided ID token.
// project_id	string	Your Firebase project ID.
func (c *RESTClient) ExchangeRefreshTokenForIDToken(firebaseAPIKey, refreshToken string) (*TokenAndRefreshToken, error) {
	type exchangeRefreshTokenResponse struct {
		ExpiresIn    string `json:"expires_in"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		UserID       string `json:"user_id"`
		ProjectID    string `json:"project_id"`
	}

	v := url.Values{}
	v.Set("key", firebaseAPIKey)
	uri := url.URL{
		Scheme:     "https",
		Host:       "securetoken.googleapis.com",
		Path:       "v1/token",
		ForceQuery: false,
		RawQuery:   v.Encode(),
	}
	reqBody := exchangeRefreshTokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
	}
	payload := url.Values{}
	payload.Set("grant_type", reqBody.GrantType)
	payload.Set("refresh_token", reqBody.RefreshToken)
	req, err := http.NewRequest("POST", uri.String(), strings.NewReader(payload.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create new request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create new POST request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		var e badRequestResponse
		body, _ := ioutil.ReadAll(res.Body)
		if err := json.Unmarshal(body, &e); err != nil {
			return nil, fmt.Errorf("json unmarshal: %w", err)
		}
		return nil, fmt.Errorf("bad request code=%d message=%q status=%q", e.Error.Code, e.Error.Message, e.Error.Status)
	}
	response := exchangeRefreshTokenResponse{}
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("json ecode error: %w", err)
	}
	return &TokenAndRefreshToken{
		IDToken:      response.IDToken,
		RefreshToken: response.RefreshToken,
	}, nil
}
