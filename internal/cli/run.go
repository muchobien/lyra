package cli

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

var runFinished = []string{"success", "failure", "cancelled", "skipped"}

var (
	logEscapes  = regexp.MustCompile(`\x1b(?:\[[0-9;?]*[A-Za-z]|\][^\x07\x1b]*(?:\x07|\x1b\\)|[()][A-Za-z0-9])`)
	logCommand  = regexp.MustCompile(`^(?:::([a-z-]+)(?: ([^:]*))?::(.*)|##\[([a-z]+)\](.*))$`)
	logUnescape = strings.NewReplacer("%0D", "\r", "%0A", "\n", "%3A", ":", "%2C", ",", "%25", "%")
	logHidden   = []string{
		"endgroup", "add-matcher", "remove-matcher", "debug", "add-mask", "save-state", "set-output",
	}
	logAnnotations = []string{"warning", "error", "notice"}
)

func (app *app) runCommand() *cobra.Command {
	run := &cobra.Command{Use: "run", Short: "Follow and control Actions runs"}
	run.AddCommand(app.repoScoped(
		app.runListCommand(),
		app.runViewCommand(),
		app.runLogsCommand(),
		app.runWatchCommand(),
		app.runCancelCommand(),
		app.runRerunCommand(),
	)...)
	return run
}

func (app *app) fetchRun(cmd *cobra.Command, repo string, number int64) (lyraapi.Run, []byte, error) {
	owner, name := splitRepo(repo)
	return decode[lyraapi.Run](app.client.api.GetRun(cmd.Context(), owner, name, number))
}

func (app *app) runListCommand() *cobra.Command {
	var limit int64
	list := &cobra.Command{
		Use:   "list",
		Short: "List runs, newest first, with their jobs",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			params := lyraapi.ListRunsParams{Limit: &limit}
			return app.show(cmd)(app.client.api.ListRuns(cmd.Context(), owner, repo, &params))
		},
	}
	list.Flags().Int64VarP(&limit, "limit", "L", 30, "how many runs, up to 100")
	return list
}

func (app *app) runViewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "view <number>",
		Short: "Show a run with its jobs and steps",
		Args:  ExactArgs(1),
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
	var follow, raw bool
	logs := &cobra.Command{
		Use:   "logs <number>",
		Short: "Print a run's job logs as plain text",
		Long: `Print a run's job logs as plain text. With several jobs, each starts with a "== <key>"
line. --job picks one job by key or id. --follow keeps printing until the jobs finish.

Colours and other terminal escapes are removed, and so are the runner's hidden workflow
commands (::add-matcher::, ::debug:: and the like) and group ends. A group start prints
as its title, a "[command]" line as the command, and an annotation as
"<level>: <file>:<line>:<col>: <title>: <message>". --raw prints the log as the runner
sent it.`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.printRunLogs(cmd, args, jobFilter, follow, raw)
		},
	}
	logs.Flags().StringVar(&jobFilter, "job", "", "only this job, by key or id")
	logs.Flags().BoolVar(&follow, "follow", false, "keep printing until the jobs finish")
	logs.Flags().BoolVar(&raw, "raw", false, "print the log as the runner sent it")
	return logs
}

func (app *app) printRunLogs(cmd *cobra.Command, args []string, jobFilter string, follow, raw bool) error {
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
		if err := app.copyJobLogs(cmd, args[0], number, job.Id, follow, raw); err != nil {
			return err
		}
	}
	return nil
}

func (app *app) copyJobLogs(cmd *cobra.Command, repo string, number, job int64, follow, raw bool) error {
	owner, name := splitRepo(repo)
	params := lyraapi.GetJobLogsParams{Offset: new(int64)}
	plain := &plainLog{out: cmd.OutOrStdout()}
	var out io.Writer = plain
	if raw {
		out = cmd.OutOrStdout()
	}
	for {
		response, err := app.client.streaming.GetJobLogs(cmd.Context(), owner, name, number, job, &params)
		if err != nil {
			return err
		}
		if response.StatusCode >= 400 {
			defer response.Body.Close()
			return responseError(response)
		}
		_, err = io.Copy(out, response.Body)
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
			return plain.flush()
		}
		time.Sleep(time.Second)
	}
}

type plainLog struct {
	out     io.Writer
	pending []byte
}

func (plain *plainLog) Write(data []byte) (int, error) {
	plain.pending = append(plain.pending, data...)
	for {
		end := bytes.IndexByte(plain.pending, '\n')
		if end < 0 {
			return len(data), nil
		}
		if err := plain.line(string(plain.pending[:end])); err != nil {
			return 0, err
		}
		plain.pending = plain.pending[end+1:]
	}
}

func (plain *plainLog) flush() error {
	if len(plain.pending) == 0 {
		return nil
	}
	line := string(plain.pending)
	plain.pending = nil
	return plain.line(line)
}

func (plain *plainLog) line(raw string) error {
	text := strings.TrimPrefix(logEscapes.ReplaceAllString(strings.TrimSuffix(raw, "\r"), ""), "[command]")
	if match := logCommand.FindStringSubmatch(text); match != nil {
		name, message := match[1]+match[4], match[3]+match[5]
		if slices.Contains(logHidden, name) {
			return nil
		}
		if name == "group" {
			text = message
		}
		if slices.Contains(logAnnotations, name) {
			fields := map[string]string{}
			for pair := range strings.SplitSeq(match[2], ",") {
				key, value, _ := strings.Cut(pair, "=")
				fields[strings.TrimSpace(key)] = logUnescape.Replace(value)
			}
			message = logUnescape.Replace(message)
			parts := []string{name}
			location := fields["file"]
			for _, key := range []string{"line", "col"} {
				if location != "" && fields[key] != "" {
					location += ":" + fields[key]
				}
			}
			if location != "" && !strings.Contains(message, location) {
				parts = append(parts, location)
			}
			if fields["title"] != "" {
				parts = append(parts, fields["title"])
			}
			text = strings.Join(append(parts, message), ": ")
		}
	}
	_, err := fmt.Fprintln(plain.out, text)
	return err
}

func (app *app) runWatchCommand() *cobra.Command {
	var timeout time.Duration
	var exitStatus bool
	watch := &cobra.Command{
		Use:   "watch <number>",
		Short: "Wait for a run to finish and print it",
		Long: `Wait for a run to finish and print it. With --exit-status, exit 1 if it didn't succeed.
It exits 8 when --timeout passes.`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.waitForRun(cmd, args, timeout, exitStatus)
		},
	}
	addTimeoutFlag(watch, &timeout)
	watch.Flags().BoolVar(&exitStatus, "exit-status", false, "exit 1 if the run didn't succeed")
	return watch
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

func (app *app) waitForRun(cmd *cobra.Command, args []string, timeout time.Duration, exitStatus bool) error {
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
			if exitStatus && info.Status != "success" {
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
		Use:   "cancel <number>",
		Short: "Cancel a run; running jobs are stopped by their runners",
		Args:  ExactArgs(1),
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
		Use:   "rerun <number>",
		Short: "Run a finished run's jobs again",
		Args:  ExactArgs(1),
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
