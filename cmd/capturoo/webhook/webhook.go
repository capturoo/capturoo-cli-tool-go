package webhook

import (
	"capturoo-cli-tool-go/cmd/capturoo/app"
	"capturoo-cli-tool-go/http"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var validEvents = []string{
	"bucket.created",
	"bucket.deleted",
	"lead.created",
}

var codeRegexp = regexp.MustCompile(`^[a-z0-9-]{1,40}$`)

type event struct {
	name      string
	resources []string
}

func (e event) String() string {
	if len(e.resources) == 0 {
		return e.name
	}
	return e.name + ":" + strings.Join(e.resources, "|")
}

// NewCmdWebhook returns an instance of the bucket sub command.
func NewCmdWebhook() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "webhook",
		Aliases: []string{"webhooks"},
		Short:   "Manage webhooks",
	}
	cmd.AddCommand(NewCmdWebhookCreate())
	cmd.AddCommand(NewCmdWebhookList())
	cmd.AddCommand(NewCmdWebhookUpdate())
	cmd.AddCommand(NewCmdWebhookDelete())
	return cmd
}

// NewCmdWebhookCreate returns an instance of the webhook create sub command.
func NewCmdWebhookCreate() *cobra.Command {
	var ids, enabled bool
	var events, url string
	var evtList []event

	cmd := &cobra.Command{
		Use:   "create WEBHOOK_CODE options",
		Short: "Create a new webhook",
		Long:  `Create a new webhook with a unique ENDPOINT for the`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("missing WEBHOOK_CODE argument")
			}

			if events == "" {
				return errors.New("set the target event list using --events")
			}
			var err error
			evtList, err = parseEventArgs(events)
			if err != nil {
				return err
			}

			// url
			if url == "" {
				return errors.New("set the URL endpoint using --url ENDPOINT")
			}
			valid, err := isValidSecureWebhookURL(url)
			if err != nil {
				return err
			}
			if !valid {
				return errors.New("ENDPOINT must use an https secure url")
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

			code := args[0]
			var evs []string
			for _, v := range evtList {
				evs = append(evs, v.String())
			}

			webhook, err := app.Client.CreateWebhook(ctx, app.JWTData.CapAID, code, url, evs, enabled)
			if errors.Is(err, http.ErrWebhookResourcesNotFound) {
				fmt.Fprintf(os.Stderr, "Resources [%s] not found.\n", err)
				os.Exit(1)
			}
			if errors.Is(err, http.ErrWebhookURLExists) {
				fmt.Fprintf(os.Stderr, "Webhook URL %s already exists. Use capturoo update to modify existing webhooks.\n", url)
				os.Exit(1)
			}
			if errors.Is(err, http.ErrWebhookCodeExists) {
				fmt.Fprintf(os.Stderr, "Webhook code %s already exists. Use capturoo update to modify existin webhook.\n", code)
				os.Exit(1)
			}
			if errors.Is(err, http.ErrBadRequest) {
				fmt.Fprintf(os.Stderr, "Bad request: %v", err)
				os.Exit(1)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to create webhook: %v\n", err)
				os.Exit(1)
			}

			if err := displayWebook(webhook, ids); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().BoolVarP(&ids, "id", "", false, "show internal ids in output (used for diagnostics)")
	cmd.Flags().StringVarP(&events, "events", "e", "", "target events EVT1[:bucketCode1|bucketCodeN...],EVT2,...")
	cmd.Flags().StringVarP(&url, "url", "u", "", "ENDPOINT secure url of the webhook handler")
	return cmd
}

// NewCmdWebhookList returns an instance of the webhook list sub command.
func NewCmdWebhookList() *cobra.Command {
	var ids, dates, reverse bool
	var sortByField string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List webhooks",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			v := ctx.Value(app.ApplicationKey("appk"))
			if v == nil {
				fmt.Fprintf(os.Stderr, "failed to get application context")
				os.Exit(1)
			}
			app := v.(*app.Ctx)

			webhooks, err := app.Client.GetWebhooks(ctx, app.JWTData.CapAID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to get webhooks: %v\n", err)
				os.Exit(1)
			}

			// optional sort
			switch sortByField {
			case "created":
				sort.Slice(webhooks, func(i, j int) bool {
					return webhooks[i].Created.Before(webhooks[j].Created)
				})
			}

			// table output
			tw := new(tabwriter.Writer).Init(os.Stdout, 0, 8, 2, ' ', 0)
			format := "%s\t%s\t%s\t%s\t"
			hformat := "%s\t%s\t%s\t%s\t"
			headers := []interface{}{
				"Webhook code",
				"Events",
				"URL",
				"Status",
			}
			if ids {
				format = "%s\t" + format
				hformat = "%s\t" + hformat
				headers = append([]interface{}{"Webhook ID"}, headers...)
			}
			if dates {
				format = format + "%v\t%v\t"
				hformat = hformat + "%v\t%v\t"
				headers = append(headers, "Created", "Modified")
			}
			format = format + "\n"
			hformat = hformat + "\n"

			fmt.Fprintf(tw, hformat, headers...)
			fmt.Fprintf(tw, hformat, headersUnderlined(headers)...)
			for _, w := range webhooks {
				var params []interface{}
				params = []interface{}{
					w.Code,
					displayEvents(w.Events),
					w.URL,
					enabledDisabled(w.Enabled),
				}
				if ids {
					params = append([]interface{}{w.WebhookID}, params...)
				}
				if dates {
					params = append(params, w.Created, w.Modified)
				}
				fmt.Fprintf(tw, format, params...)
			}
			if err := tw.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().BoolVarP(&ids, "id", "", false, "show internal ids in output (used for diagnostics)")
	cmd.Flags().StringVarP(&sortByField, "sortby", "s", "created", "sort results by the field header")
	cmd.Flags().BoolVarP(&reverse, "reverse", "r", false, "reverse the result of the sort")
	cmd.Flags().BoolVarP(&dates, "time", "t", false, "show created time in output")
	return cmd
}

// NewCmdWebhookUpdate returns an instance of the webhook update sub command.
func NewCmdWebhookUpdate() *cobra.Command {
	var ids bool
	var enable, disable bool
	var events, url string
	var evtList *[]event

	cmd := &cobra.Command{
		Use:   "update WEBHOOK_CODE",
		Short: "Update webhook",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("missing WEBHOOK_CODE argument")
			}

			if events == "" && url == "" && !enable && !disable {
				return errors.New("must set at least one of --events, --url or --enable or --disable flags")
			}

			// events (optional)
			if events != "" {
				var err error
				*evtList, err = parseEventArgs(events)
				if err != nil {
					return err
				}
			}

			// url (optional)
			if url != "" {
				valid, err := isValidSecureWebhookURL(url)
				if err != nil {
					return err
				}
				if !valid {
					return errors.New("ENDPOINT must use an https secure url")
				}
			}

			// (optional)
			if enable && disable {
				return errors.New("Use either --enable or --disable but not both")
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

			// Set the update parameter set for those fields that need updating
			var params http.UpdateParamSet
			if evtList != nil {
				var events []string
				for _, event := range *evtList {
					events = append(events, event.name)
				}
				params.Events = &events
			}
			if url != "" {
				params.URL = &url
			}
			if enable {
				var en bool = true
				params.Enabled = &en
			}
			if disable {
				var en bool = false
				params.Enabled = &en
			}

			// ensure the webhook code exists for this user
			webhooks, err := app.Client.GetWebhooks(ctx, app.JWTData.CapAID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			whash := make(map[string]string, 0)
			for _, w := range webhooks {
				whash[w.Code] = w.WebhookID
			}
			code := args[0]
			if _, ok := whash[code]; !ok {
				fmt.Fprintf(os.Stderr, "Webhook %q not found.\n", code)
				os.Exit(1)
			}

			webhook, err := app.Client.UpdateWebhook(ctx, whash[code], &params)
			if errors.Is(err, http.ErrWebhookURLExists) {
				fmt.Fprintf(os.Stderr, "Webhook URL %s already exists.\n", url)
				os.Exit(1)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to update webhook: %v\n", err)
				os.Exit(1)
			}

			if err := displayWebook(webhook, ids); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}

			fmt.Println(app)
			fmt.Println("webhook update...")
		},
	}
	cmd.Flags().BoolVarP(&ids, "id", "", false, "show internal ids in output (used for diagnostics)")
	cmd.Flags().BoolVarP(&enable, "enable", "", false, "enable the webhook if already disabled")
	cmd.Flags().BoolVarP(&disable, "disable", "", false, "disable the webhook if already enabled")
	cmd.Flags().StringVarP(&events, "events", "e", "", "target events EVT1,EVT2,...")
	cmd.Flags().StringVarP(&url, "url", "u", "", "ENDPOINT secure url of the webhook handler")
	return cmd
}

// NewCmdWebhookDelete returns an instance of the webhook delete sub command.cmd
func NewCmdWebhookDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete WEBHOOK_CODE",
		Short: "Delete webhook",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("missing WEBHOOK_CODE argument")
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

			webhooks, err := app.Client.GetWebhooks(ctx, app.JWTData.CapAID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to get webhooks: %v\n", err)
				os.Exit(1)
			}
			hash := make(map[string]string)
			for _, w := range webhooks {
				hash[w.Code] = w.WebhookID
			}

			code := args[0]
			if _, ok := hash[code]; !ok {
				fmt.Fprintf(os.Stderr, "Webhook with code %q not found.\n", code)
				os.Exit(1)
			}

			err = app.Client.DeleteWebhook(ctx, hash[code])
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to delete webhook: %v\n", err)
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

func displayWebook(webhook *http.Webhook, ids bool) error {
	tw := new(tabwriter.Writer).Init(os.Stdout, 0, 8, 2, ' ', 0)
	format := "%s\t%s\t\n"

	// webhook.WebhookID
	if ids {
		fmt.Fprintf(tw, format, "Webhook ID: ", webhook.WebhookID)
	}
	fmt.Fprintf(tw, format, "Webhook code: ", webhook.Code)
	fmt.Fprintf(tw, format, "URL: ", webhook.URL)
	fmt.Fprintf(tw, format, "Events: ", displayEvents(webhook.Events))
	fmt.Fprintf(tw, format, "Enabled:", enabledDisabled(webhook.Enabled))
	fmt.Fprintf(tw, format, "Created: ", webhook.Created)
	fmt.Fprintf(tw, format, "Modified: ", webhook.Modified)
	return tw.Flush()
}

func displayEvents(events []string) string {
	for i, e := range events {
		events[i] = fmt.Sprintf("'%s'", e)
	}
	return "[" + strings.Join(events, ", ") + "]"
}

func enabledDisabled(t bool) string {
	if t {
		return "Enabled"
	}
	return "Disabled"
}

// func parseCSVString(s string) ([]string, error) {
// 	events := strings.Split(s, ",")
// 	unknowns := unknownEvents(events)
// 	if len(unknowns) > 0 {
// 		return nil, fmt.Errorf("unknown event types %s", strings.Join(unknowns, ","))
// 	}
// 	return events, nil
// }

// TODO: use regexp to validate format
func isValidResourceName(s string) bool {
	return true
}

func adjacentHyphens(s string) bool {
	var hyphen bool
	for _, c := range s {
		if c == '-' {
			if hyphen {
				return true
			}
			hyphen = true
			continue
		}
		if hyphen {
			hyphen = false
		}
	}
	return false
}

func isValidCode(c string) (ok bool, message string) {
	// a bucket code should be all lower case a to z,
	// including the hyphen character. Digits may be added
	// to the string provided they are placed at the end
	// of the string. Two ajacent hyphens are not permitted.
	// The string must not start or end with a hyphen.
	// The string length must not exceed 40 characters so that
	// it may be easily distinguished from a Firestore id.
	//
	// Valid bucket codes:
	//   "my-swish-bucket", "123-bucket", "balloons-99"
	//
	// Invalid
	//   "my--bucket", "-apples" and "CamelCaseBucket"
	if len(c) == 0 {
		return false, "must be at least 1 character in length"
	}

	if len(c) > 40 {
		return false, "exceeds 40 characters"
	}

	if strings.HasPrefix(c, "-") {
		return false, "starts with hyphen character"
	}

	if strings.HasSuffix(c, "-") {
		return false, "ends with hyphen character"
	}

	if adjacentHyphens(c) {
		return false, "has two or more adjacent hyphens"
	}

	if !codeRegexp.MatchString(c) {
		return false, "must contain lower case characters a-z including the hyphen only"
	}
	return true, ""
}

func parseEventArgs(s string) ([]event, error) {
	events := make([]event, 0)
	// iterate each event e.g.
	// bucket.created,bucket.deleted,leads.created:bucket-one|leads.creatd:bucket-two,
	parts := strings.Split(s, ",")
	for _, v := range parts {
		if isContextDriven(v) {
			parts := strings.Split(v, ":")
			if len(parts) != 2 {
				return nil, fmt.Errorf("context driven event must contain a single colon : character")
			}
			name := parts[0]
			resources := strings.Split(parts[1], "|")
			for _, r := range resources {
				valid, message := isValidCode(r)
				if !valid {
					return nil, fmt.Errorf("resource name %q is invalid : %s", r, message)
				}
			}
			events = append(events, event{
				name:      name,
				resources: resources,
			})
		} else {
			if v == "lead.created" {
				return nil, fmt.Errorf("lead.created is a context driven event and must contain a resource name in the form of evt:resource")
			}
			ev := event{
				name: v,
			}
			events = append(events, ev)
		}
	}
	return events, nil
}

// func unknownEvents(events []string) []string {
// 	var unknowns []string
// 	for _, v := range events {
// 		if isContextDriven(v) {
// 			parseEvent(v)

// 		}
// 		if !contains(validEvents, v) {
// 			unknowns = append(unknowns, v)
// 		}
// 	}
// 	return unknowns
// }

func isContextDriven(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return true
		}
	}
	return false
}

func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func isValidSecureWebhookURL(endpoint string) (bool, error) {
	u, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return false, fmt.Errorf("failed to parse url: %w", err)
	}
	if u.Scheme != "https" {
		return false, nil
	}
	return true, nil
}
