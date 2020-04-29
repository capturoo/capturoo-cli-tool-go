package token

import (
	"capturoo-cli-tool-go/cmd/capturoo/app"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewCmdToken returns an instance of the token sub command.
func NewCmdToken() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "token",
		Aliases: []string{"tokens"},
		Short:   "Token management for calling the API directly",
	}
	cmd.AddCommand(NewCmdTokenShow())
	return cmd
}

// NewCmdTokenShow returns an instance of the token show sub command.
func NewCmdTokenShow() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show usuable JWT for API calls",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			v := ctx.Value(app.ApplicationKey("appk"))
			if v == nil {
				fmt.Fprintf(os.Stderr, "failed to get application context")
				os.Exit(1)
			}
			app := v.(*app.Ctx)

			fmt.Printf("Name: %s\n", app.JWTData.Name)
			fmt.Printf("Email: %s\n", app.JWTData.Email)
			fmt.Printf("Account ID: %s\n", app.JWTData.CapAID)
			fmt.Printf("Role: %s\n", app.JWTData.CapRole)
			fmt.Printf("export JWT='%s'\n", app.TART.IDToken)
		},
	}
}
