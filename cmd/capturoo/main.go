package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"capturoo-cli-tool-go/cmd/capturoo/account"
	"capturoo-cli-tool-go/cmd/capturoo/app"
	"capturoo-cli-tool-go/cmd/capturoo/bucket"
	"capturoo-cli-tool-go/cmd/capturoo/configmgr"
	"capturoo-cli-tool-go/cmd/capturoo/lead"
	"capturoo-cli-tool-go/cmd/capturoo/token"
	"capturoo-cli-tool-go/cmd/capturoo/webhook"
	"capturoo-cli-tool-go/fbauth"
	"capturoo-cli-tool-go/http"

	"github.com/spf13/cobra"
)

var version string
var endpoint string
var gitCommit string

func main() {
	overrideEndpoint, found := os.LookupEnv("CAPTUROO_CLI_ENDPOINT")
	if found {
		// TODO: sanitise the endpoint URL
		endpoint = overrideEndpoint
	}
	tokenFilename, err := urlToHostName(endpoint)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}

	appv := &app.Ctx{
		Version:       version,
		Endpoint:      endpoint,
		GitCommit:     gitCommit,
		TokenFilename: tokenFilename,
		Client:        http.NewClient(endpoint),
	}

	root := cobra.Command{
		Use:     "capturoo",
		Short:   "capturoo is a CLI tool for managing leads",
		Version: version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			v := ctx.Value(app.ApplicationKey("appk"))
			if v == nil {
				fmt.Fprintf(os.Stderr, "Failed to get application context.\n")
				os.Exit(1)
			}
			app := v.(*app.Ctx)

			tart, err := configmgr.ReadTokenAndRefreshToken(app.TokenFilename)
			if errors.Is(err, configmgr.ErrTokenFileNotFound) {
				fmt.Fprintf(os.Stderr, "No account configured. Run capturoo account login to begin.\n")
				os.Exit(1)
			} else if errors.Is(err, configmgr.ErrTokenExpired) {
				// If the current token has expired, exchange the refresh token
				// for a new one.
				autoconf, err := app.Client.AutoConf(ctx)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to auto configure via the endpoint %v.\n", err)
					os.Exit(1)
				}
				auth := fbauth.NewRESTClient()
				tart, err = auth.ExchangeRefreshTokenForIDToken(autoconf.Data.FirebaseConfig.APIKey, tart.RefreshToken)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Exchange Refresh Token for ID Token failed: %+v\n", err)
					os.Exit(1)
				}
				if err := configmgr.WriteTokenAndRefreshToken(app.TokenFilename, tart); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to write new token: %+v\n", err)
					os.Exit(1)
				}
			} else if err != nil {
				fmt.Fprintf(os.Stderr, "%+v\n", err)
				os.Exit(1)
			}

			app.Client.JWT = tart.IDToken
			app.TART = tart

			app.JWTData, err = configmgr.ParseJWT(tart.IDToken)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to parse JWT: %+v\n", err)
				os.Exit(1)
			}
		},
	}
	root.AddCommand(account.NewCmdAccount())
	root.AddCommand(bucket.NewCmdBucket())
	root.AddCommand(lead.NewCmdLead())
	root.AddCommand(token.NewCmdToken())
	root.AddCommand(NewCmdVersion())
	root.AddCommand(webhook.NewCmdWebhook())

	ctx := context.WithValue(context.Background(), app.ApplicationKey("appk"), appv)
	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}

// NewCmdVersion returns an instance of the version sub command.
func NewCmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Output CLI tool version string",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// prevent root level PersistentPreRun
		},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Capturoo CLI tool %s (build %s)\n", version, gitCommit)
		},
	}
}

// urlToHostName converts a standard URL string to hostname replacing
// the dot character with underscores.
func urlToHostName(u string) (string, error) {
	url, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("failed to parse url %q: %w", u, err)
	}

	port := url.Port()
	var suffix string
	if port != "" {
		suffix = "_" + port
	}
	return strings.ReplaceAll(url.Hostname()+suffix, ".", "_"), nil
}
