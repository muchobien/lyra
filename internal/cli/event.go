package cli

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

func (app *app) eventCommand() *cobra.Command {
	event := &cobra.Command{Use: "event", Short: "Read the event log"}
	event.AddCommand(app.eventListCommand(), app.eventWaitCommand())
	return event
}

func (app *app) eventListCommand() *cobra.Command {
	var repo string
	var limit int64
	list := &cobra.Command{
		Use:   "list",
		Short: "List recent events with their actors",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := lyraapi.ListEventsParams{Limit: &limit}
			if repo != "" {
				params.Repo = &repo
			}
			return app.show(cmd)(app.client.api.ListEvents(cmd.Context(), &params))
		},
	}
	list.Flags().StringVar(&repo, "repo", "", "only events for this repository")
	list.Flags().Int64Var(&limit, "limit", 50, "maximum number of events")
	return list
}

func (app *app) eventWaitCommand() *cobra.Command {
	var repo, assignee string
	var actions []string
	var timeout time.Duration
	wait := &cobra.Command{
		Use:   "wait",
		Short: "Wait for the next matching event and print it",
		Long: `Wait for the next event that matches the filters and print it as JSON.

Without --since, only events after the command starts match. Pass the id of an
event you have already seen as --since to also match anything that happened
since then. The command reconnects if the connection drops, without missing
events. It exits 3 with {"error":"timeout"} when --timeout passes.

--assignee me waits until an issue or pull request is assigned to you (your
LYRA_AGENT name, or your login without one): a work queue. Check
lyra issue list --assignee @me first for work assigned before you started waiting.`,
		Args: ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := lyraapi.StreamEventsParams{Action: &actions}
			if repo != "" {
				params.Repo = &repo
			}
			if assignee != "" {
				params.Assignee = &assignee
			}
			return app.waitAndEmit(cmd, params, timeout)
		},
	}
	flags := wait.Flags()
	flags.StringVar(&repo, "repo", "", "only events for this repository")
	flags.StringArrayVar(
		&actions,
		"action",
		nil,
		"only these actions, as glob patterns such as push or repo.* (repeatable)",
	)
	flags.StringVar(&assignee, "assignee", "", "only assignments to this login or agent name, or me")
	flags.Int64("since", 0, "match events after this event id (default: from now)")
	flags.DurationVar(&timeout, "timeout", 0, "give up after this long, such as 30m (default: wait forever)")
	return wait
}

func (app *app) waitAndEmit(cmd *cobra.Command, params lyraapi.StreamEventsParams, timeout time.Duration) error {
	flags := cmd.Flags()
	if flags.Changed("since") {
		since, _ := flags.GetInt64("since")
		if since < 0 {
			return UsageError{errors.New("--since must be an event id")}
		}
		value := fmt.Sprint(since)
		params.Since = &value
	}
	output, err := app.client.waitForEvent(cmd.Context(), params, timeout)
	if err != nil {
		return err
	}
	event := jsontext.Value(output)
	if err := event.Indent(jsontext.WithIndent("  ")); err != nil {
		return err
	}
	return app.emit(cmd, append(event, '\n'))
}
