package account

import (
	"bufio"
	"fmt"
	"os"
	"syscall"

	"capturoo-cli-tool-go/cmd/capturoo/app"
	"capturoo-cli-tool-go/fbauth"
	"capturoo-cli-tool-go/http"

	"capturoo-cli-tool-go/cmd/capturoo/configmgr"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

// NewCmdAccount returns a new instance of account sub command
func NewCmdAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "account",
		Short:   "Manage your account",
		Aliases: []string{"accounts"},
	}
	cmd.AddCommand(NewCmdAccountInfo())
	cmd.AddCommand(NewCmdAccountLogin())
	cmd.AddCommand(NewCmdAccountLogout())
	return cmd
}

// NewCmdAccountLogout logout sub command.
func NewCmdAccountLogout() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Account logout",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// prevent root level PersistentPreRun
		},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("NOT YET IMPLEMENTED Logout\n")
		},
	}
}

// NewCmdAccountLogin login sub command.
func NewCmdAccountLogin() *cobra.Command {
	var useEmailLogin bool
	cmd := &cobra.Command{
		Use:   "login [--email]",
		Short: "Account login",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// prevent root level PersistentPreRun
		},
		Run: func(cmd *cobra.Command, args []string) {
			v := cmd.Context().Value(app.ApplicationKey("appk"))
			if v == nil {
				fmt.Fprintf(os.Stderr, "failed to get application context")
				os.Exit(1)
			}
			app := v.(*app.Ctx)

			auth := fbauth.NewRESTClient()
			var tart *fbauth.TokenAndRefreshToken
			var developerKey string
			if useEmailLogin {
				email, password, err := readEmailAndPassword()
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to get email/password: %v\n", err)
					os.Exit(1)
				}

				sir, err := auth.SignInWithEmailAndPassword(app.FirebaseAPIKey, email, string(password))
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to signin with email and password: %v\n", err)
					os.Exit(1)
				}
				tart = &fbauth.TokenAndRefreshToken{
					IDToken:      sir.IDToken,
					RefreshToken: sir.RefreshToken,
				}
			} else {
				var err error
				developerKey, err = readDeveloperKey()
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to get developer key: %v\n", err)
					os.Exit(1)
				}

				// Signin With the developer key.
				client := http.NewClient(app.Endpoint)
				token, _, err := client.SignInWithDevKey(string(developerKey))
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to sign in using developer key: %v\n", err)
					os.Exit(1)
				}

				tart, err = auth.ExchangeCustomTokenForIDAndRefreshToken(app.FirebaseAPIKey, token)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%+v\n", err)
					os.Exit(1)
				}
			}

			err := configmgr.WriteTokenAndRefreshToken(app.TokenFilename, tart)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%+v\n", err)
				os.Exit(1)
			}

			jwtData, err := configmgr.ParseJWT(tart.IDToken)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to parse JWT: %v", err)
				os.Exit(1)
			}

			fmt.Println("Command line tool setup for the following user:")
			fmt.Printf("Name: %s\n", jwtData.Name)
			fmt.Printf("Email: %s\n", jwtData.Email)
			fmt.Printf("Account ID: %s\n", jwtData.CapAID)
			fmt.Printf("Role: %s\n", jwtData.CapRole)
		},
	}
	cmd.Flags().BoolVarP(&useEmailLogin, "email", "e", false, "use email address to login")
	return cmd
}

// NewCmdAccountInfo info sub command.
func NewCmdAccountInfo() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show account information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("listing accounts...")
		},
	}
}

func readEmailAndPassword() (email, password string, err error) {
	fmt.Printf("Email: ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		email = scanner.Text()
	}
	fmt.Printf("Password: ")
	pbyte, err := terminal.ReadPassword(syscall.Stdin)
	if err != nil {
		return "", "", fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println()
	return email, string(pbyte), nil
}

func readDeveloperKey() (devKey string, err error) {
	fmt.Printf("Developer key: ")
	devByte, err := terminal.ReadPassword(syscall.Stdin)
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println()
	return string(devByte), nil
}
