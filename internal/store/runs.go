package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"slices"
	"time"

	"github.com/muchobien/lyra/internal/store/queries"
)

var terminal = []string{"success", "failure", "cancelled", "skipped"}

type Run struct {
	ID        int64      `json:"id"`
	Repo      string     `json:"repo"`
	Number    int64      `json:"number"`
	Workflow  string     `json:"workflow"`
	Name      string     `json:"name"`
	Title     string     `json:"title"`
	Event     string     `json:"event"`
	Ref       string     `json:"ref"`
	SHA       string     `json:"sha"`
	Status    string     `json:"status"`
	Pull      int64      `json:"pull,omitempty"`
	HeadRef   string     `json:"head_ref,omitempty"`
	BaseRef   string     `json:"base_ref,omitempty"`
	Author    Actor      `json:"author"`
	CreatedAt time.Time  `json:"created_at"`
	StartedAt *time.Time `json:"started_at"`
	StoppedAt *time.Time `json:"stopped_at"`
	Jobs      []Job      `json:"jobs,omitempty"`
	Payload   string     `json:"-"`
}

type Job struct {
	ID          int64             `json:"id"`
	Key         string            `json:"key"`
	Name        string            `json:"name"`
	Needs       []string          `json:"needs"`
	RunsOn      []string          `json:"runs_on"`
	Status      string            `json:"status"`
	Attempt     int64             `json:"attempt"`
	Outputs     map[string]string `json:"outputs"`
	StartedAt   *time.Time        `json:"started_at"`
	StoppedAt   *time.Time        `json:"stopped_at"`
	TaskID      int64             `json:"task_id,omitempty"`
	Steps       []TaskStep        `json:"steps"`
	Conditional bool              `json:"-"`
	Payload     string            `json:"-"`
}

type TaskStep struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Result    string     `json:"result"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	StoppedAt *time.Time `json:"stopped_at,omitempty"`
	LogIndex  int64      `json:"log_index"`
	LogLength int64      `json:"log_length"`
}

type Task struct {
	ID        int64
	JobID     int64
	RunnerID  int64
	Attempt   int64
	Status    string
	LogLength int64
	LogObject string
}

type NeedResult struct {
	Result  string
	Outputs map[string]string
}

type Assignment struct {
	Task  Task
	Job   Job
	Run   Run
	Needs map[string]NeedResult
}

type LogLine struct {
	Index   int64     `json:"index"`
	At      time.Time `json:"at"`
	Content string    `json:"content"`
}

func nullTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func runFromRow(row queries.Run) Run {
	return Run{
		ID: row.ID, Repo: row.Repo, Number: row.Number, Workflow: row.Workflow, Name: row.Name, Title: row.Title, Event: row.Event,
		Ref: row.Ref, SHA: row.Sha, Status: row.Status, Author: Actor{User: row.User, Via: row.Via, Agent: row.Agent, Session: row.Session},
		CreatedAt: row.CreatedAt, StartedAt: nullTime(row.StartedAt), StoppedAt: nullTime(row.StoppedAt), Payload: row.Payload,
		Pull: row.Pull, HeadRef: row.HeadRef, BaseRef: row.BaseRef,
	}
}

func jobFromRow(row queries.Job) (Job, error) {
	job := Job{
		ID: row.ID, Key: row.JobKey, Name: row.Name, Status: row.Status, Attempt: row.Attempt, Conditional: row.Conditional,
		Payload: row.Payload, StartedAt: nullTime(row.StartedAt), StoppedAt: nullTime(row.StoppedAt), Steps: []TaskStep{},
	}
	if err := json.Unmarshal([]byte(row.Needs), &job.Needs); err != nil {
		return job, err
	}
	if err := json.Unmarshal([]byte(row.RunsOn), &job.RunsOn); err != nil {
		return job, err
	}
	return job, json.Unmarshal([]byte(row.Outputs), &job.Outputs)
}

func taskFromRow(row queries.Task) Task {
	return Task{ID: row.ID, JobID: row.JobID, RunnerID: row.RunnerID.Int64, Attempt: row.Attempt, Status: row.Status, LogLength: row.LogLength, LogObject: row.LogObject}
}

func loadJobs(q *queries.Queries, runID int64) ([]Job, error) {
	ctx := context.Background()
	rows, err := q.ListRunJobs(ctx, runID)
	if err != nil {
		return nil, err
	}
	jobs := []Job{}
	for _, row := range rows {
		job, err := jobFromRow(row)
		if err != nil {
			return nil, err
		}
		tasks, err := q.ListJobTasks(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		if len(tasks) > 0 {
			last := tasks[len(tasks)-1]
			job.TaskID = last.ID
			if err := json.Unmarshal([]byte(last.Steps), &job.Steps); err != nil {
				return nil, err
			}
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func saveJob(q *queries.Queries, job Job) error {
	outputs, err := json.Marshal(job.Outputs)
	if err != nil {
		return err
	}
	toNull := func(value *time.Time) sql.NullTime {
		if value == nil {
			return sql.NullTime{}
		}
		return sql.NullTime{Time: *value, Valid: true}
	}
	return q.UpdateJob(context.Background(), queries.UpdateJobParams{
		Status: job.Status, Attempt: job.Attempt, Outputs: string(outputs), StartedAt: toNull(job.StartedAt), StoppedAt: toNull(job.StoppedAt), ID: job.ID,
	})
}

func (s *Store) CreateRun(actor Actor, run Run, jobs []Job) (Run, int64, error) {
	var created Run
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		ctx := context.Background()
		row, err := q.CreateRun(ctx, queries.CreateRunParams{
			Repo: run.Repo, Workflow: run.Workflow, Name: run.Name, Title: run.Title, Event: run.Event, Ref: run.Ref, Sha: run.SHA,
			Payload: run.Payload, User: actor.User, Via: actor.Via, Agent: actor.Agent, Session: actor.Session, CreatedAt: now(),
			Pull: run.Pull, HeadRef: run.HeadRef, BaseRef: run.BaseRef,
		})
		if err != nil {
			return err
		}
		created = runFromRow(row)
		for _, job := range jobs {
			status := "waiting"
			if len(job.Needs) > 0 {
				status = "blocked"
			}
			needs, _ := json.Marshal(job.Needs)
			runsOn, _ := json.Marshal(job.RunsOn)
			if _, err := q.CreateJob(ctx, queries.CreateJobParams{
				RunID: created.ID, JobKey: job.Key, Name: job.Name, Needs: string(needs), RunsOn: string(runsOn),
				Conditional: job.Conditional, Payload: job.Payload, Status: status,
			}); err != nil {
				return err
			}
		}
		if created.Jobs, err = loadJobs(q, created.ID); err != nil {
			return err
		}
		eventID, err = addEvent(q, "run.queued", created.Repo, 0, actor, map[string]any{
			"number": created.Number, "workflow": created.Workflow, "event": created.Event, "ref": created.Ref, "sha": created.SHA,
		})
		return err
	})
	return created, eventID, err
}

func (s *Store) Run(repo string, number int64) (Run, error) {
	row, err := s.queries.GetRun(context.Background(), queries.GetRunParams{Repo: repo, Number: number})
	if err != nil {
		return Run{}, notFound(err)
	}
	run := runFromRow(row)
	run.Jobs, err = loadJobs(s.queries, run.ID)
	return run, err
}

func (s *Store) Runs(repo string, limit int64) ([]Run, error) {
	rows, err := s.queries.ListRuns(context.Background(), queries.ListRunsParams{Repo: repo, Limit: limit})
	if err != nil {
		return nil, err
	}
	runs := []Run{}
	for _, row := range rows {
		run := runFromRow(row)
		if run.Jobs, err = loadJobs(s.queries, run.ID); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func needResults(jobs []Job, keys []string) map[string]NeedResult {
	needs := map[string]NeedResult{}
	for _, key := range keys {
		result := NeedResult{Result: "success", Outputs: map[string]string{}}
		for _, job := range jobs {
			if job.Key != key {
				continue
			}
			if job.Status != "success" && result.Result == "success" {
				result.Result = job.Status
			}
			for name, value := range job.Outputs {
				result.Outputs[name] = value
			}
		}
		needs[key] = result
	}
	return needs
}

func (s *Store) AssignJob(runnerID int64, labels []string, tokenHash string) (*Assignment, error) {
	var assignment *Assignment
	err := s.inTx(func(q *queries.Queries) error {
		ctx := context.Background()
		waiting, err := q.ListWaitingJobs(ctx)
		if err != nil {
			return err
		}
		for _, row := range waiting {
			job, err := jobFromRow(row)
			if err != nil {
				return err
			}
			if !slices.ContainsFunc(job.RunsOn, func(label string) bool { return !slices.Contains(labels, label) }) {
				started := now()
				job.Status, job.Attempt, job.StartedAt, job.StoppedAt = "running", job.Attempt+1, &started, nil
				if err := saveJob(q, job); err != nil {
					return err
				}
				task, err := q.CreateTask(ctx, queries.CreateTaskParams{
					JobID: job.ID, RunnerID: sql.NullInt64{Int64: runnerID, Valid: true}, Attempt: job.Attempt, TokenHash: tokenHash, StartedAt: started,
				})
				if err != nil {
					return err
				}
				runRow, err := q.GetRunByID(ctx, row.RunID)
				if err != nil {
					return err
				}
				run := runFromRow(runRow)
				if run.Status != "running" {
					startedAt := sql.NullTime{Time: started, Valid: true}
					if run.StartedAt != nil {
						startedAt.Time = *run.StartedAt
					}
					if err := q.UpdateRunStatus(ctx, queries.UpdateRunStatusParams{Status: "running", StartedAt: startedAt, ID: run.ID}); err != nil {
						return err
					}
					run.Status = "running"
				}
				jobs, err := loadJobs(q, run.ID)
				if err != nil {
					return err
				}
				assignment = &Assignment{Task: taskFromRow(task), Job: job, Run: run, Needs: needResults(jobs, job.Needs)}
				return nil
			}
		}
		return nil
	})
	return assignment, err
}

func (s *Store) RunningTask(tokenHash string) (Task, Run, error) {
	row, err := s.queries.GetRunningTaskByToken(context.Background(), tokenHash)
	if err != nil {
		return Task{}, Run{}, notFound(err)
	}
	job, err := s.queries.GetJob(context.Background(), row.JobID)
	if err != nil {
		return Task{}, Run{}, err
	}
	run, err := s.queries.GetRunByID(context.Background(), job.RunID)
	return taskFromRow(row), runFromRow(run), err
}

func (s *Store) Task(id int64) (Task, Run, error) {
	row, err := s.queries.GetTask(context.Background(), id)
	if err != nil {
		return Task{}, Run{}, notFound(err)
	}
	job, err := s.queries.GetJob(context.Background(), row.JobID)
	if err != nil {
		return Task{}, Run{}, err
	}
	run, err := s.queries.GetRunByID(context.Background(), job.RunID)
	return taskFromRow(row), runFromRow(run), err
}

func settleRun(q *queries.Queries, run Run, actor Actor) error {
	ctx := context.Background()
	for {
		jobs, err := loadJobs(q, run.ID)
		if err != nil {
			return err
		}
		changed := false
		for _, job := range jobs {
			if job.Status != "blocked" {
				continue
			}
			needs := needResults(jobs, job.Needs)
			done, failed := true, false
			for _, key := range job.Needs {
				for _, other := range jobs {
					if other.Key == key && !slices.Contains(terminal, other.Status) {
						done = false
					}
				}
				failed = failed || needs[key].Result != "success"
			}
			if !done {
				continue
			}
			job.Status = "waiting"
			if failed && !job.Conditional {
				stopped := now()
				job.Status, job.StoppedAt = "skipped", &stopped
			}
			if err := saveJob(q, job); err != nil {
				return err
			}
			changed = true
		}
		if changed {
			continue
		}
		status, active := "success", false
		for _, job := range jobs {
			switch job.Status {
			case "failure":
				status = "failure"
			case "cancelled":
				if status != "failure" {
					status = "cancelled"
				}
			case "waiting", "blocked", "running":
				active = true
			}
		}
		if active || slices.Contains(terminal, run.Status) {
			return nil
		}
		started := sql.NullTime{}
		if run.StartedAt != nil {
			started = sql.NullTime{Time: *run.StartedAt, Valid: true}
		}
		if err := q.UpdateRunStatus(ctx, queries.UpdateRunStatusParams{Status: status, StartedAt: started, StoppedAt: sql.NullTime{Time: now(), Valid: true}, ID: run.ID}); err != nil {
			return err
		}
		if _, err := addEvent(q, "run.completed", run.Repo, 0, actor, map[string]any{"number": run.Number, "workflow": run.Workflow, "result": status, "sha": run.SHA}); err != nil {
			return err
		}
		return announceChecks(q, run.Repo, run.SHA)
	}
}

func (s *Store) UpdateTask(taskID int64, steps []TaskStep, outputs map[string]string, result string) (string, []string, error) {
	var status string
	var stored []string
	err := s.inTx(func(q *queries.Queries) error {
		ctx := context.Background()
		taskRow, err := q.GetTask(ctx, taskID)
		if err != nil {
			return notFound(err)
		}
		jobRow, err := q.GetJob(ctx, taskRow.JobID)
		if err != nil {
			return err
		}
		job, err := jobFromRow(jobRow)
		if err != nil {
			return err
		}
		runRow, err := q.GetRunByID(ctx, jobRow.RunID)
		if err != nil {
			return err
		}
		run := runFromRow(runRow)
		encoded, err := json.Marshal(steps)
		if err != nil {
			return err
		}
		if steps == nil {
			encoded = []byte(taskRow.Steps)
		}
		stopped := sql.NullTime{}
		taskStatus := taskRow.Status
		if result != "" && taskStatus == "running" {
			taskStatus, stopped = result, sql.NullTime{Time: now(), Valid: true}
		}
		if err := q.UpdateTask(ctx, queries.UpdateTaskParams{Status: taskStatus, Steps: string(encoded), StoppedAt: stopped, UpdatedAt: sql.NullTime{Time: now(), Valid: true}, ID: taskID}); err != nil {
			return err
		}
		if job.Attempt != taskRow.Attempt {
			status = "cancelled"
			return nil
		}
		for name, value := range outputs {
			job.Outputs[name] = value
		}
		if result != "" && job.Status == "running" {
			job.Status, job.StoppedAt = result, &stopped.Time
		}
		if result != "" && job.Status == "cancelled" && job.StoppedAt == nil {
			job.StoppedAt = &stopped.Time
		}
		if err := saveJob(q, job); err != nil {
			return err
		}
		status = job.Status
		for name := range job.Outputs {
			stored = append(stored, name)
		}
		if result == "" {
			return nil
		}
		actor := Actor{User: run.Author.User, Via: "job:" + job.Key}
		if _, err := addEvent(q, "job.completed", run.Repo, 0, actor, map[string]any{"run": run.Number, "job": job.Key, "name": job.Name, "result": job.Status}); err != nil {
			return err
		}
		return settleRun(q, run, actor)
	})
	return status, stored, err
}

func (s *Store) AppendLog(taskID, index int64, lines []LogLine) (int64, error) {
	var ack int64
	err := s.inTx(func(q *queries.Queries) error {
		ctx := context.Background()
		task, err := q.GetTask(ctx, taskID)
		if err != nil {
			return notFound(err)
		}
		if err := q.TouchTask(ctx, queries.TouchTaskParams{UpdatedAt: sql.NullTime{Time: now(), Valid: true}, ID: taskID}); err != nil {
			return err
		}
		ack = task.LogLength
		if index > ack || task.LogObject != "" {
			return nil
		}
		for offset, line := range lines {
			position := index + int64(offset)
			if position < ack {
				continue
			}
			if err := q.InsertLogRow(ctx, queries.InsertLogRowParams{TaskID: taskID, Idx: position, At: line.At, Content: line.Content}); err != nil {
				return err
			}
			ack = position + 1
		}
		return q.SetTaskLog(ctx, queries.SetTaskLogParams{LogLength: ack, LogObject: "", ID: taskID})
	})
	return ack, err
}

func (s *Store) LogLines(taskID, from int64) ([]LogLine, error) {
	rows, err := s.queries.ListLogRows(context.Background(), queries.ListLogRowsParams{TaskID: taskID, Idx: from})
	if err != nil {
		return nil, err
	}
	lines := []LogLine{}
	for _, row := range rows {
		lines = append(lines, LogLine{Index: row.Idx, At: row.At, Content: row.Content})
	}
	return lines, nil
}

func (s *Store) ArchiveLog(taskID, length int64, object string) error {
	return s.inTx(func(q *queries.Queries) error {
		ctx := context.Background()
		if err := q.SetTaskLog(ctx, queries.SetTaskLogParams{LogLength: length, LogObject: object, ID: taskID}); err != nil {
			return err
		}
		return q.DeleteLogRows(ctx, taskID)
	})
}

func (s *Store) CancelRun(actor Actor, repo string, number int64) (int64, error) {
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		row, err := q.GetRun(context.Background(), queries.GetRunParams{Repo: repo, Number: number})
		if err != nil {
			return notFound(err)
		}
		run := runFromRow(row)
		if slices.Contains(terminal, run.Status) {
			return InvalidError("run is already " + run.Status)
		}
		jobs, err := loadJobs(q, run.ID)
		if err != nil {
			return err
		}
		for _, job := range jobs {
			if slices.Contains(terminal, job.Status) {
				continue
			}
			if job.Status != "running" {
				stopped := now()
				job.StoppedAt = &stopped
			}
			job.Status = "cancelled"
			if err := saveJob(q, job); err != nil {
				return err
			}
		}
		if eventID, err = addEvent(q, "run.cancelled", repo, 0, actor, map[string]any{"number": number}); err != nil {
			return err
		}
		return settleRun(q, run, actor)
	})
	return eventID, err
}

func (s *Store) RerunRun(actor Actor, repo string, number int64, failedOnly bool) (int64, error) {
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		ctx := context.Background()
		row, err := q.GetRun(ctx, queries.GetRunParams{Repo: repo, Number: number})
		if err != nil {
			return notFound(err)
		}
		run := runFromRow(row)
		if !slices.Contains(terminal, run.Status) {
			return InvalidError("run is still " + run.Status + "; cancel it first")
		}
		jobs, err := loadJobs(q, run.ID)
		if err != nil {
			return err
		}
		reset := map[string]bool{}
		for changed := true; changed; {
			changed = false
			for _, job := range jobs {
				again := !failedOnly || job.Status != "success" || slices.ContainsFunc(job.Needs, func(key string) bool { return reset[key] })
				if again && !reset[job.Key] {
					reset[job.Key], changed = true, true
				}
			}
		}
		for _, job := range jobs {
			if !reset[job.Key] {
				continue
			}
			job.Status, job.Outputs, job.StartedAt, job.StoppedAt = "waiting", map[string]string{}, nil, nil
			if len(job.Needs) > 0 {
				job.Status = "blocked"
			}
			if err := saveJob(q, job); err != nil {
				return err
			}
		}
		if err := q.UpdateRunStatus(ctx, queries.UpdateRunStatusParams{Status: "waiting", ID: run.ID}); err != nil {
			return err
		}
		run.Status = "waiting"
		if err := settleRun(q, run, actor); err != nil {
			return err
		}
		eventID, err = addEvent(q, "run.queued", repo, 0, actor, map[string]any{"number": number, "workflow": run.Workflow, "rerun": true, "failed_only": failedOnly})
		return err
	})
	return eventID, err
}

type Check struct {
	Run      int64  `json:"run"`
	Workflow string `json:"workflow"`
	Event    string `json:"event"`
	JobID    int64  `json:"job_id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
}

type Checks struct {
	State string  `json:"state"`
	SHA   string  `json:"sha"`
	Jobs  []Check `json:"jobs"`
}

func loadChecks(q *queries.Queries, repo, sha string) (Checks, error) {
	rows, err := q.ListRunsForSHA(context.Background(), queries.ListRunsForSHAParams{Repo: repo, Sha: sha})
	if err != nil {
		return Checks{}, err
	}
	latest := map[string]Run{}
	order := []string{}
	for _, row := range rows {
		run := runFromRow(row)
		key := run.Workflow + "\x00" + run.Event
		if _, seen := latest[key]; !seen {
			order = append(order, key)
		}
		latest[key] = run
	}
	checks := Checks{State: "none", SHA: sha, Jobs: []Check{}}
	pending, failed := false, false
	for _, key := range order {
		run := latest[key]
		jobs, err := loadJobs(q, run.ID)
		if err != nil {
			return checks, err
		}
		for _, job := range jobs {
			checks.Jobs = append(checks.Jobs, Check{Run: run.Number, Workflow: run.Workflow, Event: run.Event, JobID: job.ID, Name: job.Name, Status: job.Status})
			pending = pending || !slices.Contains(terminal, job.Status)
			failed = failed || job.Status == "failure" || job.Status == "cancelled"
		}
	}
	switch {
	case len(checks.Jobs) == 0:
	case pending:
		checks.State = "pending"
	case failed:
		checks.State = "failure"
	default:
		checks.State = "success"
	}
	return checks, nil
}

func (s *Store) Checks(repo, sha string) (Checks, error) {
	return loadChecks(s.queries, repo, sha)
}

func announceChecks(q *queries.Queries, repo, sha string) error {
	checks, err := loadChecks(q, repo, sha)
	if err != nil || checks.State == "pending" || checks.State == "none" {
		return err
	}
	pulls, err := q.ListPullItems(context.Background(), queries.ListPullItemsParams{Repo: repo, State: "open"})
	if err != nil {
		return err
	}
	for _, pull := range pulls {
		if pull.Pull.HeadSha != sha {
			continue
		}
		if _, err := addEvent(q, "pr.checks", repo, pull.Item.Number, Actor{User: "lyra", Via: "actions"}, map[string]any{"state": checks.State, "sha": sha}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) AnnounceChecks(repo, sha string) error {
	return s.inTx(func(q *queries.Queries) error {
		return announceChecks(q, repo, sha)
	})
}

type ExpiredRun struct {
	ID      int64
	Repo    string
	Objects []string
}

func (s *Store) StaleTasks(before time.Time) ([]int64, error) {
	return s.queries.ListStaleTasks(context.Background(), sql.NullTime{Time: before, Valid: true})
}

func (s *Store) ExpiredRuns(before time.Time) ([]ExpiredRun, error) {
	ctx := context.Background()
	rows, err := s.queries.ListExpiredRuns(ctx, sql.NullTime{Time: before, Valid: true})
	if err != nil {
		return nil, err
	}
	runs := []ExpiredRun{}
	for _, row := range rows {
		objects, err := s.queries.ListRunLogObjects(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		runs = append(runs, ExpiredRun{ID: row.ID, Repo: row.Repo, Objects: objects})
	}
	return runs, nil
}

func (s *Store) DeleteRun(id int64) error {
	return s.inTx(func(q *queries.Queries) error {
		return q.DeleteRun(context.Background(), id)
	})
}
