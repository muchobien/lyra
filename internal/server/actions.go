package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
	"slices"
	"strings"
	"time"

	runnerv1 "code.forgejo.org/forgejo/actions-proto/runner/v1"
	"code.forgejo.org/forgejo/runner/v13/act/jobparser"
	"code.forgejo.org/forgejo/runner/v13/act/model"
	"code.forgejo.org/forgejo/runner/v13/act/workflowpattern"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

var workflowDirs = []string{".forgejo/workflows", ".github/workflows"}

var resultNames = map[runnerv1.Result]string{
	runnerv1.Result_RESULT_SUCCESS:   "success",
	runnerv1.Result_RESULT_FAILURE:   "failure",
	runnerv1.Result_RESULT_CANCELLED: "cancelled",
	runnerv1.Result_RESULT_SKIPPED:   "skipped",
}

func resultValue(name string) runnerv1.Result {
	for value, known := range resultNames {
		if known == name {
			return value
		}
	}
	return runnerv1.Result_RESULT_UNSPECIFIED
}

func workflowFiles(git repos.Git, dir, sha string) []string {
	for _, workflowDir := range workflowDirs {
		entries, err := git.Tree(dir, sha, workflowDir)
		if err != nil {
			continue
		}
		files := []string{}
		for _, entry := range entries {
			if entry.Type == "blob" && (strings.HasSuffix(entry.Name, ".yml") || strings.HasSuffix(entry.Name, ".yaml")) {
				files = append(files, workflowDir+"/"+entry.Name)
			}
		}
		return files
	}
	return nil
}

func matchesPush(event *jobparser.Event, ref string, changed []string) bool {
	branch, isBranch := strings.CutPrefix(ref, "refs/heads/")
	tag, isTag := strings.CutPrefix(ref, "refs/tags/")
	acts := event.Acts()
	_, branches := acts["branches"]
	_, branchesIgnore := acts["branches-ignore"]
	_, tags := acts["tags"]
	_, tagsIgnore := acts["tags-ignore"]
	if (branches || branchesIgnore) && !(tags || tagsIgnore) && !isBranch {
		return false
	}
	if (tags || tagsIgnore) && !(branches || branchesIgnore) && !isTag {
		return false
	}
	trace := &workflowpattern.EmptyTraceWriter{}
	for name, patterns := range acts {
		compiled, err := workflowpattern.CompilePatterns(patterns...)
		if err != nil {
			return false
		}
		switch {
		case name == "branches" && isBranch && workflowpattern.Skip(compiled, []string{branch}, trace):
			return false
		case name == "branches-ignore" && isBranch && workflowpattern.Filter(compiled, []string{branch}, trace):
			return false
		case name == "tags" && isTag && workflowpattern.Skip(compiled, []string{tag}, trace):
			return false
		case name == "tags-ignore" && isTag && workflowpattern.Filter(compiled, []string{tag}, trace):
			return false
		case name == "paths" && changed != nil && workflowpattern.Skip(compiled, changed, trace):
			return false
		case name == "paths-ignore" && changed != nil && workflowpattern.Filter(compiled, changed, trace):
			return false
		}
	}
	return true
}

func QueueRuns(db *store.Store, git repos.Git, dir, repo string, actor store.Actor, updates []RefUpdate) []string {
	messages := []string{}
	owner, name, _ := strings.Cut(repo, "/")
	for _, update := range updates {
		if zeroSHA(update.New) || !(strings.HasPrefix(update.Ref, "refs/heads/") || strings.HasPrefix(update.Ref, "refs/tags/")) {
			continue
		}
		var changed []string
		if !zeroSHA(update.Old) {
			files, err := git.DiffFiles(dir, update.Old, update.New)
			if err == nil {
				changed = []string{}
				for _, file := range files {
					changed = append(changed, file.Path)
				}
			}
		}
		commit, err := git.Commit(dir, update.New)
		if err != nil {
			messages = append(messages, "actions: "+err.Error())
			continue
		}
		payload, _ := json.Marshal(map[string]any{
			"ref": update.Ref, "before": update.Old, "after": update.New,
			"repository": map[string]any{
				"full_name": repo, "name": name, "owner": map[string]any{"login": owner, "username": owner},
				"default_branch": git.DefaultBranch(dir),
			},
			"head_commit": map[string]any{"id": commit.SHA, "message": strings.TrimSpace(commit.Subject + "\n\n" + commit.Body), "timestamp": commit.Committer.Date},
			"pusher":      map[string]any{"name": actor.User, "username": actor.User},
			"sender":      map[string]any{"login": actor.User, "username": actor.User},
		})
		for _, file := range workflowFiles(git, dir, update.New) {
			content, err := git.Blob(dir, update.New, file)
			if err != nil {
				messages = append(messages, file+": "+err.Error())
				continue
			}
			workflow, err := model.ReadWorkflow(bytes.NewReader(content), false)
			if err != nil {
				messages = append(messages, file+": "+err.Error())
				continue
			}
			events, err := jobparser.ParseRawOn(&workflow.RawOn)
			if err != nil {
				messages = append(messages, file+": "+err.Error())
				continue
			}
			if !slices.ContainsFunc(events, func(event *jobparser.Event) bool {
				return event.Name == "push" && matchesPush(event, update.Ref, changed)
			}) {
				continue
			}
			singles, err := jobparser.Parse(content, false)
			if err != nil {
				messages = append(messages, file+": "+err.Error())
				continue
			}
			jobs := []store.Job{}
			for _, single := range singles {
				key, job := single.Job()
				encoded, err := single.Marshal()
				if err != nil {
					return append(messages, file+": "+err.Error())
				}
				jobName := job.Name
				if jobName == "" {
					jobName = key
				}
				jobs = append(jobs, store.Job{
					Key: key, Name: jobName, Needs: job.Needs(), RunsOn: job.RunsOn(), Conditional: job.If.Value != "", Payload: string(encoded),
				})
			}
			runName := workflow.Name
			if runName == "" {
				runName = path.Base(file)
			}
			run, _, err := db.CreateRun(actor, store.Run{
				Repo: repo, Workflow: file, Name: runName, Title: commit.Subject, Event: "push", Ref: update.Ref, SHA: update.New, Payload: string(payload),
			}, jobs)
			if err != nil {
				messages = append(messages, file+": "+err.Error())
				continue
			}
			messages = append(messages, fmt.Sprintf("queued run #%d (%s)", run.Number, runName))
		}
	}
	return messages
}

func (service runnerService) task(runner store.Runner) (*runnerv1.Task, error) {
	token := randomHex(20)
	assignment, err := service.server.store.AssignJob(runner.ID, runner.Labels, store.HashToken(token))
	if err != nil || assignment == nil {
		return nil, err
	}
	run, job := assignment.Run, assignment.Job
	owner, _, _ := strings.Cut(run.Repo, "/")
	event := map[string]any{}
	if err := json.Unmarshal([]byte(run.Payload), &event); err != nil {
		return nil, err
	}
	refType, refName := "branch", strings.TrimPrefix(run.Ref, "refs/heads/")
	if tag, isTag := strings.CutPrefix(run.Ref, "refs/tags/"); isTag {
		refType, refName = "tag", tag
	}
	defaultActions := service.server.config.DefaultActionsURL
	taskContext, err := structpb.NewStruct(map[string]any{
		"actor": run.Author.User, "repository": run.Repo, "repository_owner": owner, "sha": run.SHA, "ref": run.Ref,
		"ref_name": refName, "ref_type": refType, "event_name": run.Event, "event": event, "run_id": fmt.Sprint(run.ID),
		"run_number": fmt.Sprint(run.Number), "run_attempt": fmt.Sprint(job.Attempt), "job": job.Key,
		"workflow_ref": run.Repo + "/" + run.Workflow + "@" + run.Ref, "token": token, "retention_days": "90",
		"base_ref": "", "head_ref": "", "forgejo_default_actions_url": defaultActions, "gitea_default_actions_url": defaultActions,
	})
	if err != nil {
		return nil, err
	}
	needs := map[string]*runnerv1.TaskNeed{}
	for key, need := range assignment.Needs {
		needs[key] = &runnerv1.TaskNeed{Outputs: need.Outputs, Result: resultValue(need.Result)}
	}
	return &runnerv1.Task{
		Id: assignment.Task.ID, WorkflowPayload: []byte(job.Payload), Context: taskContext, Needs: needs, Vars: map[string]string{},
		Secrets: map[string]string{"GITHUB_TOKEN": token, "FORGEJO_TOKEN": token},
	}, nil
}

func (service runnerService) ownTask(runner store.Runner, id int64) (store.Task, store.Run, error) {
	task, run, err := service.server.store.Task(id)
	if errors.Is(err, store.ErrNotFound) || (err == nil && task.RunnerID != runner.ID) {
		return task, run, connect.NewError(connect.CodeNotFound, fmt.Errorf("task %d not found", id))
	}
	if err != nil {
		return task, run, connect.NewError(connect.CodeInternal, err)
	}
	return task, run, nil
}

func (service runnerService) UpdateTask(ctx context.Context, request *connect.Request[runnerv1.UpdateTaskRequest]) (*connect.Response[runnerv1.UpdateTaskResponse], error) {
	runner, err := service.authenticate(request.Header())
	if err != nil {
		return nil, err
	}
	state := request.Msg.State
	if _, _, err := service.ownTask(runner, state.GetId()); err != nil {
		return nil, err
	}
	steps := []store.TaskStep{}
	for _, step := range state.Steps {
		taskStep := store.TaskStep{ID: step.Id, Result: resultNames[step.Result], LogIndex: step.LogIndex, LogLength: step.LogLength}
		if step.StartedAt != nil {
			started := step.StartedAt.AsTime()
			taskStep.StartedAt = &started
		}
		if step.StoppedAt != nil {
			stopped := step.StoppedAt.AsTime()
			taskStep.StoppedAt = &stopped
		}
		steps = append(steps, taskStep)
	}
	result := resultNames[state.Result]
	status, stored, err := service.server.store.UpdateTask(state.Id, steps, request.Msg.Outputs, result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if result != "" && runner.Ephemeral {
		if _, err := service.server.store.DeleteRunner(store.Actor{User: runner.User, Via: "runner:" + runner.Name}, runner.ID); err != nil {
			slog.Error("delete ephemeral runner", "runner", runner.ID, "error", err)
		}
	}
	return connect.NewResponse(&runnerv1.UpdateTaskResponse{
		State: &runnerv1.TaskState{Id: state.Id, Result: resultValue(status)}, SentOutputs: stored,
	}), nil
}

func logObject(repo string, taskID int64) string {
	return fmt.Sprintf("/lyra/actions/%s/%d.log", repo, taskID)
}

func (service runnerService) UpdateLog(ctx context.Context, request *connect.Request[runnerv1.UpdateLogRequest]) (*connect.Response[runnerv1.UpdateLogResponse], error) {
	runner, err := service.authenticate(request.Header())
	if err != nil {
		return nil, err
	}
	task, run, err := service.ownTask(runner, request.Msg.TaskId)
	if err != nil {
		return nil, err
	}
	lines := []store.LogLine{}
	for _, row := range request.Msg.Rows {
		lines = append(lines, store.LogLine{At: row.Time.AsTime(), Content: row.Content})
	}
	ack, err := service.server.store.AppendLog(task.ID, request.Msg.Index, lines)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if request.Msg.NoMore && ack == request.Msg.Index+int64(len(lines)) {
		stored, err := service.server.store.LogLines(task.ID, 0)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		var text bytes.Buffer
		for _, line := range stored {
			text.WriteString(line.At.UTC().Format(time.RFC3339Nano) + "\t" + line.Content + "\n")
		}
		object := logObject(run.Repo, task.ID)
		if err := service.server.storage.PutContent(ctx, object, text.Bytes()); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if err := service.server.store.ArchiveLog(task.ID, ack, object); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&runnerv1.UpdateLogResponse{AckIndex: ack}), nil
}

func (service runnerService) UpdateStepSummary(ctx context.Context, request *connect.Request[runnerv1.UpdateStepSummaryRequest]) (*connect.Response[runnerv1.UpdateStepSummaryResponse], error) {
	if _, err := service.authenticate(request.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&runnerv1.UpdateStepSummaryResponse{}), nil
}

func readLog(ctx context.Context, server *Server, task store.Task, from int64) ([]store.LogLine, error) {
	if task.LogObject == "" {
		return server.store.LogLines(task.ID, from)
	}
	reader, err := server.storage.Reader(ctx, task.LogObject, 0)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	lines := []store.LogLine{}
	for index, raw := range strings.Split(strings.TrimSuffix(string(content), "\n"), "\n") {
		if int64(index) < from || raw == "" {
			continue
		}
		stamp, text, _ := strings.Cut(raw, "\t")
		at, _ := time.Parse(time.RFC3339Nano, stamp)
		lines = append(lines, store.LogLine{Index: int64(index), At: at, Content: text})
	}
	return lines, nil
}
