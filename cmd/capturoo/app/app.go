package app

import (
	"capturoo-cli-tool-go/cmd/capturoo/configmgr"
	"capturoo-cli-tool-go/fbauth"
	"capturoo-cli-tool-go/http"
)

// ApplicationKey context key
type ApplicationKey string

// Ctx global context
type Ctx struct {
	Version       string
	Endpoint      string
	GitCommit     string
	TokenFilename string
	Client        *http.Client
	TART          *fbauth.TokenAndRefreshToken
	JWTData       *configmgr.JWTData
}
