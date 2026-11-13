package cli

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/spf13/cobra"
)

type runInfo struct {
	Number int64    `json:"number"`
	Status string   `json:"status"`
	Jobs   []runJob `json:"jobs"`
}

type runJob struct {
	ID     int64  `json:"id"`
	Key    string `json:"key"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type eventCursor struct {
	ID int64 `json:"id"`
}

var runFinished = []string{"success", "failure", "cancelled", "skipped"}

func runPath(repo, number string) string {
	return repoAPI(repo) + "/runs/" + url.PathEscape(number)
}

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

func (app *app) fetchRun(cmd *cobra.Command, repo, number string) (runInfo, []byte, error) {
	var info runInfo
	output, err := app.client.request(cmd.Context(), http.MethodGet, runPath(repo, number), nil)
	if err != nil {
		return info, nil, err
	}
	return info, output, json.Unmarshal(output, &info)
}

func (app *app) runListCommand() *cobra.Command {
	var limit int
	list := &cobra.Command{
		Use:   "list <repo>",
		Short: "List runs, newest first, with their jobs",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, repoAPI(args[0])+"/runs?limit="+fmt.Sprint(limit), nil)
		},
	}
	list.Flags().IntVar(&limit, "limit", 30, "how many runs, up to 100")
	return list
}

func (app *app) runViewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "view <repo> <number>",
		Short: "Show a run with its jobs and steps",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, runPath(args[0], args[1]), nil)
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
	info, _, err := app.fetchRun(cmd, args[0], args[1])
	if err != nil {
		return err
	}
	jobs := info.Jobs
	if jobFilter != "" {
		jobs = nil
		for _, job := range info.Jobs {
			if job.Key == jobFilter || fmt.Sprint(job.ID) == jobFilter {
				jobs = append(jobs, job)
			}
		}
		if len(jobs) == 0 {
			return fmt.Errorf("run #%d has no job %s", info.Number, jobFilter)
		}
	}
	for _, job := range jobs {
		if len(info.Jobs) > 1 && jobFilter == "" {
			fmt.Fprintf(cmd.OutOrStdout(), "== %s\n", job.Key)
		}
		path := runPath(args[0], args[1]) + "/jobs/" + fmt.Sprint(job.ID) + "/logs"
		if err := app.copyJobLogs(cmd, path, follow); err != nil {
			return err
		}
	}
	return nil
}

func (app *app) copyJobLogs(cmd *cobra.Command, path string, follow bool) error {
	offset := "0"
	for {
		req, err := app.client.newRequest(cmd.Context(), http.MethodGet, path+"?offset="+offset, nil)
		if err != nil {
			return err
		}
		response, err := app.client.streaming.Do(req)
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
		offset = response.Header.Get("Lyra-Log-Next")
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
	path := "/api/events?limit=1&repo=" + url.QueryEscape(repo)
	latest, err := app.client.request(cmd.Context(), http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	var events []eventCursor
	if err := json.Unmarshal(latest, &events); err != nil {
		return "", err
	}
	if len(events) > 0 {
		return fmt.Sprint(events[0].ID), nil
	}
	return "0", nil
}

func (app *app) waitForRun(cmd *cobra.Command, args []string, timeout time.Duration) error {
	cursor, err := app.latestEvent(cmd, args[0])
	if err != nil {
		return err
	}
	deadline := time.Now().Add(timeout)
	for {
		info, output, err := app.fetchRun(cmd, args[0], args[1])
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
		query := url.Values{"repo": {args[0]}, "action": {"run.completed"}, "since": {cursor}}
		path := "/api/events/stream?" + query.Encode()
		event, err := app.client.waitForEvent(cmd.Context(), path, max(remaining, 0))
		if err != nil {
			return err
		}
		var completed eventCursor
		if err := json.Unmarshal(event, &completed); err != nil {
			return err
		}
		cursor = fmt.Sprint(completed.ID)
	}
}

func (app *app) runCancelCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <repo> <number>",
		Short: "Cancel a run; running jobs are stopped by their runners",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodPost, runPath(args[0], args[1])+"/cancel", nil)
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
			path := runPath(args[0], args[1]) + "/rerun"
			if failedOnly {
				path += "?failed=true"
			}
			return app.call(cmd, http.MethodPost, path, nil)
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
