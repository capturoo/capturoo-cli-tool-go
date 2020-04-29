package lead

import (
	"capturoo-cli-tool-go/cmd/capturoo/app"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewCmdLead returns an instance of the lead sub command.
func NewCmdLead() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "lead",
		Aliases: []string{"leads"},
		Short:   "Manage leads",
	}
	cmd.AddCommand(NewCmdLeadExport())
	return cmd
}

// NewCmdLeadExport returns an instance of the lead export sub command.
func NewCmdLeadExport() *cobra.Command {
	var format, output string
	cmd := &cobra.Command{
		Use:   "export RESOURCE_NAME",
		Short: "Export leads from a bucket",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("missing RESOURCE_NAME argument")
			}

			if format != "json" && format != "yaml" && format != "csv" {
				return errors.New("format must be either json, yaml or csv")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			v := ctx.Value(app.ApplicationKey("appk"))
			if v == nil {
				fmt.Fprintf(os.Stderr, "failed to get application context")
				os.Exit(1)
			}
			app := v.(*app.Ctx)

			// build lookup tables of both:
			//   bucketCode   -> bucketId
			//   publicAPIKey -> bucketId
			// as the user might want to locate a bucket by id, code or public key.
			buckets, err := app.Client.GetBuckets(ctx, app.JWTData.CapAID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to list buckets: %v\n", err)
				os.Exit(1)
			}
			resourceNameMap := make(map[string]string, 0)
			publicAPIKeyMap := make(map[string]string, 0)
			for _, b := range buckets {
				resourceNameMap[b.ResourceName] = b.BucketID
				publicAPIKeyMap[b.PublicAPIKey] = b.BucketID
			}

			resourceName := args[0]
			if _, ok := resourceNameMap[resourceName]; !ok {
				fmt.Fprintf(os.Stderr, "Bucket with resource name %q not found.\n", resourceName)
				os.Exit(1)
			}

			var f *os.File
			f = os.Stdout
			if output != "" {
				f, err = os.Create(output)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
					os.Exit(1)
				}
				defer func() {
					if err := f.Close(); err != nil {
						fmt.Fprintf(os.Stderr, "%v\n", err)
						os.Exit(1)
					}
				}()
			}

			if err := app.Client.WriteLeads(ctx, format, f, resourceNameMap[resourceName]); err != nil {
				fmt.Fprintf(os.Stderr, "failed to output leads: %v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "json", "export format json, yaml or csv")
	cmd.Flags().StringVarP(&output, "output", "o", "", "output to file")
	return cmd
}
