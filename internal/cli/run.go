package cli

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"slices"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

var runFinished = []string{"success", "failure", "cancelled", "skipped"}

func (app *app) runCommand() *cobra.Command {
	run := &cobra.Command{Use: "run", Short: "Follow and control Actions runs"}
	run.AddCommand(
		app.runListCommand(),
		app.runViewCommand(),
		app.runLogsCommand(),
		app.runWaitCommand(),
		app.runCancelCommand(),
		app.runRerunCommand(),
	)
	return run
}

func (app *app) fetchRun(cmd *cobra.Command, repo string, number int64) (lyraapi.Run, []byte, error) {
	owner, name := splitRepo(repo)
	return decode[lyraapi.Run](app.client.api.GetRun(cmd.Context(), owner, name, number))
}

func (app *app) runListCommand() *cobra.Command {
	var limit int64
	list := &cobra.Command{
		Use:   "list <repo>",
		Short: "List runs, newest first, with their jobs",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			params := lyraapi.ListRunsParams{Limit: &limit}
			return app.show(cmd)(app.client.api.ListRuns(cmd.Context(), owner, repo, &params))
		},
	}
	list.Flags().Int64Var(&limit, "limit", 30, "how many runs, up to 100")
	return list
}

func (app *app) runViewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "view <repo> <number>",
		Short: "Show a run with its jobs and steps",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := parseNumber(args[1])
			if err != nil {
				return err
			}
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.GetRun(cmd.Context(), owner, repo, number))
		},
	}
}

func (app *app) runLogsCommand() *cobra.Command {
	var jobFilter string
	var follow bool
	logs := &cobra.Command{
		Use:   "logs <repo> <number>",
		Short: "Print a run's job logs as plain text",
		Long: `Print a run's job logs as plain text. With several jobs, each starts with a "== <key>"
line. --job picks one job by key or id. --follow keeps printing until the jobs finish.`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.printRunLogs(cmd, args, jobFilter, follow)
		},
	}
	logs.Flags().StringVar(&jobFilter, "job", "", "only this job, by key or id")
	logs.Flags().BoolVar(&follow, "follow", false, "keep printing until the jobs finish")
	return logs
}

func (app *app) printRunLogs(cmd *cobra.Command, args []string, jobFilter string, follow bool) error {
	number, err := parseNumber(args[1])
	if err != nil {
		return err
	}
	info, _, err := app.fetchRun(cmd, args[0], number)
	if err != nil {
		return err
	}
	all := []lyraapi.Job{}
	if info.Jobs != nil {
		all = *info.Jobs
	}
	jobs := all
	if jobFilter != "" {
		jobs = nil
		for _, job := range all {
			if job.Key == jobFilter || fmt.Sprint(job.Id) == jobFilter {
				jobs = append(jobs, job)
			}
		}
		if len(jobs) == 0 {
			return fmt.Errorf("run #%d has no job %s", info.Number, jobFilter)
		}
	}
	for _, job := range jobs {
		if len(all) > 1 && jobFilter == "" {
			fmt.Fprintf(cmd.OutOrStdout(), "== %s\n", job.Key)
		}
		if err := app.copyJobLogs(cmd, args[0], number, job.Id, follow); err != nil {
			return err
		}
	}
	return nil
}

func (app *app) copyJobLogs(cmd *cobra.Command, repo string, number, job int64, follow bool) error {
	owner, name := splitRepo(repo)
	params := lyraapi.GetJobLogsParams{Offset: new(int64)}
	for {
		response, err := app.client.streaming.GetJobLogs(cmd.Context(), owner, name, number, job, &params)
		if err != nil {
			return err
		}
		if response.StatusCode >= 400 {
			defer response.Body.Close()
			return responseError(response)
		}
		_, err = io.Copy(cmd.OutOrStdout(), response.Body)
		response.Body.Close()
		if err != nil {
			return err
		}
		next, err := strconv.ParseInt(response.Header.Get("Lyra-Log-Next"), 10, 64)
		if err != nil {
			return fmt.Errorf("Lyra-Log-Next: %w", err)
		}
		params.Offset = &next
		if !follow || response.Header.Get("Lyra-Log-Done") == "true" {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func (app *app) runWaitCommand() *cobra.Command {
	var timeout time.Duration
	wait := &cobra.Command{
		Use:   "wait <repo> <number>",
		Short: "Wait for a run to finish; exit 0 if it succeeded, 1 if not, 3 on timeout",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.waitForRun(cmd, args, timeout)
		},
	}
	wait.Flags().DurationVar(&timeout, "timeout", 0, "give up after this long, such as 30m (default: no limit)")
	return wait
}

func (app *app) latestEvent(cmd *cobra.Command, repo string) (string, error) {
	limit := int64(1)
	params := lyraapi.ListEventsParams{Limit: &limit, Repo: &repo}
	events, _, err := decode[[]lyraapi.Event](app.client.api.ListEvents(cmd.Context(), &params))
	if err != nil {
		return "", err
	}
	if len(events) > 0 {
		return fmt.Sprint(events[0].Id), nil
	}
	return "0", nil
}

func (app *app) waitForRun(cmd *cobra.Command, args []string, timeout time.Duration) error {
	number, err := parseNumber(args[1])
	if err != nil {
		return err
	}
	cursor, err := app.latestEvent(cmd, args[0])
	if err != nil {
		return err
	}
	deadline := time.Now().Add(timeout)
	for {
		info, output, err := app.fetchRun(cmd, args[0], number)
		if err != nil {
			return err
		}
		if slices.Contains(runFinished, info.Status) {
			if err := app.emit(cmd, output); err != nil {
				return err
			}
			if info.Status != "success" {
				return fmt.Errorf("run #%d finished: %s", info.Number, info.Status)
			}
			return nil
		}
		remaining := time.Until(deadline)
		if timeout > 0 && remaining <= 0 {
			return ErrTimeout
		}
		actions := []string{"run.completed"}
		params := lyraapi.StreamEventsParams{Repo: &args[0], Action: &actions, Since: &cursor}
		event, err := app.client.waitForEvent(cmd.Context(), params, max(remaining, 0))
		if err != nil {
			return err
		}
		var completed lyraapi.Event
		if err := json.Unmarshal(event, &completed); err != nil {
			return err
		}
		cursor = fmt.Sprint(completed.Id)
	}
}

func (app *app) runCancelCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <repo> <number>",
		Short: "Cancel a run; running jobs are stopped by their runners",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := parseNumber(args[1])
			if err != nil {
				return err
			}
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.CancelRun(cmd.Context(), owner, repo, number))
		},
	}
}

func (app *app) runRerunCommand() *cobra.Command {
	var failedOnly bool
	rerun := &cobra.Command{
		Use:   "rerun <repo> <number>",
		Short: "Run a finished run's jobs again",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := parseNumber(args[1])
			if err != nil {
				return err
			}
			owner, repo := splitRepo(args[0])
			params := lyraapi.RerunRunParams{Failed: &failedOnly}
			return app.show(cmd)(app.client.api.RerunRun(cmd.Context(), owner, repo, number, &params))
		},
	}
	rerun.Flags().BoolVar(
		&failedOnly,
		"failed",
		false,
		"only jobs that didn't succeed, and the jobs that need them",
	)
	return rerun
}
