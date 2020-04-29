package bucket

import (
	"capturoo-cli-tool-go/cmd/capturoo/app"
	"capturoo-cli-tool-go/internal"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// NewCmdBucket returns an instance of the bucket sub command.
func NewCmdBucket() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bucket",
		Aliases: []string{"buckets"},
		Short:   "Manage buckets",
	}
	cmd.AddCommand(NewCmdBucketCreate())
	cmd.AddCommand(NewCmdBucketGet())
	cmd.AddCommand(NewCmdBucketList())
	cmd.AddCommand(NewCmdBucketDelete())
	return cmd
}

// NewCmdBucketCreate returns an instance of the bucket create sub command.
func NewCmdBucketCreate() *cobra.Command {
	var bucketName string
	cmd := &cobra.Command{
		Use:   "create BUCKET_CODE [-n BUCKET_NAME]",
		Short: "Create a new bucket",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("missing BUCKET_CODE argument")
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

			bucketCode := args[0]
			bucket, err := app.Client.CreateBucket(ctx, app.JWTData.CapAID, bucketCode, bucketName)
			if err == internal.ErrBucketCodeExists {
				fmt.Fprintf(os.Stderr, "A bucket with code %q already exists.\n", bucketCode)
				os.Exit(1)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to create bucket: %v", err)
				os.Exit(1)
			}
			tw := new(tabwriter.Writer).Init(os.Stdout, 0, 8, 2, ' ', 0)
			format := "%s\t%s\t\n"

			fmt.Fprintf(tw, format, "Resource name:", bucket.ResourceName)
			fmt.Fprintf(tw, format, "Bucket Name:", bucket.BucketName)
			fmt.Fprintf(tw, format, "Public API Key:", bucket.PublicAPIKey)

			if err := tw.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().StringVarP(&bucketName, "name", "n", "", "human readable bucket name to label your bucket")
	return cmd
}

// NewCmdBucketGet returns an instance of the bucket get sub command.
func NewCmdBucketGet() *cobra.Command {
	return &cobra.Command{
		Use:   "get RESOURCE_NAME",
		Short: "Get bucket details",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("missing RESOURCE_NAME argument")
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
			//   resourceName -> bucketId
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
				fmt.Fprintf(os.Stderr, "Bucket resource name %q not found.\n", resourceName)
				os.Exit(1)
			}

			bucket, err := app.Client.GetBucket(ctx, resourceNameMap[resourceName])
			if err != nil {
				fmt.Fprintf(os.Stderr, "%+v\n", err)
				os.Exit(1)
			}

			tw := new(tabwriter.Writer).Init(os.Stdout, 0, 8, 2, ' ', 0)
			format := "%s\t%s\t\n"
			fmt.Fprintf(tw, format, "Bucket ID:", bucket.BucketID)
			fmt.Fprintf(tw, format, "Resource name:", bucket.ResourceName)
			fmt.Fprintf(tw, format, "Account ID:", bucket.AccountID)
			fmt.Fprintf(tw, format, "Bucket name:", bucket.BucketName)
			fmt.Fprintf(tw, format, "Public API Key:", bucket.PublicAPIKey)
			fmt.Fprintf(tw, format, "Created:", bucket.Created)
			fmt.Fprintf(tw, format, "Modified:", bucket.Modified)
			if err := tw.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}
}

// NewCmdBucketList returns an instance of the bucket list sub command.
func NewCmdBucketList() *cobra.Command {
	var showAccountColumn bool
	var ids, dates, reverse bool
	var sortByField string

	cmd := &cobra.Command{
		Use:   "list [--time|-t] [--id] [-ta] [-s code|name|created] [--reverse]",
		Short: "List buckets",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			v := ctx.Value(app.ApplicationKey("appk"))
			if v == nil {
				fmt.Fprintf(os.Stderr, "failed to get application context")
				os.Exit(1)
			}
			app := v.(*app.Ctx)

			buckets, err := app.Client.GetBuckets(ctx, app.JWTData.CapAID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to list buckets: %v\n", err)
				os.Exit(1)
			}

			// optional sort
			switch sortByField {
			case "created":
				if reverse {
					sort.Slice(buckets, func(i, j int) bool {
						return buckets[i].Created.After(buckets[j].Created)
					})
				} else {
					sort.Slice(buckets, func(i, j int) bool {
						return buckets[i].Created.Before(buckets[j].Created)
					})
				}
			case "code":
				if reverse {
					sort.Slice(buckets, func(i, j int) bool {
						return buckets[i].ResourceName > buckets[j].ResourceName
					})
				} else {
					sort.Slice(buckets, func(i, j int) bool {
						return buckets[i].ResourceName < buckets[j].ResourceName
					})
				}
			case "name":
				if reverse {
					sort.Slice(buckets, func(i, j int) bool {
						return buckets[i].BucketName > buckets[j].BucketName
					})
				} else {
					sort.Slice(buckets, func(i, j int) bool {
						return buckets[i].BucketName < buckets[j].BucketName
					})

				}
			}

			// table output
			tw := new(tabwriter.Writer).Init(os.Stdout, 0, 8, 2, ' ', 0)
			format := "%s\t%s\t%s\t"
			headers := []interface{}{
				"Resource name",
				"Bucket name",
				"Public API Key",
			}
			if showAccountColumn {
				format = fmt.Sprintf("%s%s", "%s\t", format)
				headers = append([]interface{}{"Account ID"}, headers...)
			}
			if ids {
				format = fmt.Sprintf("%s%s", "%s\t", format)
				headers = append([]interface{}{"Bucket ID"}, headers...)
			}
			if dates {
				format = format + "%v\t%v\t"
				headers = append(headers, "Created", "Modified")
			}
			format = fmt.Sprintf("%s\n", format)

			fmt.Fprintf(tw, format, headers...)
			fmt.Fprintf(tw, format, headersUnderlined(headers)...)
			for _, b := range buckets {
				var params []interface{}
				params = []interface{}{
					b.ResourceName,
					b.BucketName,
					b.PublicAPIKey,
				}
				if showAccountColumn {
					params = append([]interface{}{b.AccountID}, params...)
				}
				if ids {
					params = append([]interface{}{b.BucketID}, params...)
				}
				if dates {
					params = append(params, b.Created, b.Modified)
				}
				fmt.Fprintf(tw, format, params...)
			}
			if err := tw.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			var plural string
			if len(buckets) > 1 {
				plural = "s"
			}
			fmt.Printf("\n%d bucket%s in your account\n", len(buckets), plural)
		},
	}
	cmd.Flags().BoolVarP(&showAccountColumn, "accounts", "a", false, "show account IDs alongside buckets")
	cmd.Flags().BoolVarP(&ids, "id", "x", false, "Show internal ids in output (used for diagnostics)")
	cmd.Flags().BoolVarP(&reverse, "reverse", "r", false, "Reverse the result of the sort")
	cmd.Flags().StringVarP(&sortByField, "sortby", "s", "created", "Sort results by the field header (default is created). Can be code, name or created.")
	cmd.Flags().BoolVarP(&dates, "time", "t", false, "Include the created and modified timestamps in the output.")
	return cmd
}

// NewCmdBucketDelete returns an instance of the bucket delete sub command.
func NewCmdBucketDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete RESOURCE_NAME",
		Short: "Delete a bucket",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("missing RESOURCE_NAME argument")
			}
			if len(args) > 1 {
				return errors.New("delete accepts a single argument")
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
			//   resourceName -> bucketId
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
				fmt.Fprintf(os.Stderr, "Bucket with resource name %q not found. (Resource name format is accountId:bucketCode)\n", resourceName)
				os.Exit(1)
			}

			if err := app.Client.DeleteBucket(ctx, resourceNameMap[resourceName]); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}
	return cmd
}

func headersUnderlined(headers []interface{}) []interface{} {
	results := make([]interface{}, 0)
	for _, h := range headers {
		results = append(results, strings.Repeat("-", len(h.(string))))
	}
	return results
}
