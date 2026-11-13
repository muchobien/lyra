package server

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	pingv1 "code.forgejo.org/forgejo/actions-proto/ping/v1"
	runnerv1 "code.forgejo.org/forgejo/actions-proto/runner/v1"
	"code.forgejo.org/forgejo/actions-proto/runner/v1/runnerv1connect"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/muchobien/lyra/internal/store"
)

type runnerService struct {
	runnerv1connect.UnimplementedRunnerServiceHandler
	server *Server
}

type pingService struct{}

var resultNames = map[runnerv1.Result]string{
	runnerv1.Result_RESULT_SUCCESS:   statusSuccess,
	runnerv1.Result_RESULT_FAILURE:   statusFailure,
	runnerv1.Result_RESULT_CANCELLED: statusCancelled,
	runnerv1.Result_RESULT_SKIPPED:   statusSkipped,
}

func unregistered() error {
	return connect.NewError(connect.CodeUnauthenticated, errors.New("unregistered runner"))
}

func internal(err error) error {
	return connect.NewError(connect.CodeInternal, err)
}

func resultValue(name string) runnerv1.Result {
	for value, known := range resultNames {
		if known == name {
			return value
		}
	}
	return runnerv1.Result_RESULT_UNSPECIFIED
}

func logObject(repo string, taskID int64) string {
	return fmt.Sprintf("/lyra/actions/%s/%d.log", repo, taskID)
}

func (pingService) Ping(
	ctx context.Context,
	request *connect.Request[pingv1.PingRequest],
) (*connect.Response[pingv1.PingResponse], error) {
	return connect.NewResponse(&pingv1.PingResponse{Data: "Hello, " + request.Msg.Data + "!"}), nil
}

func runnerMessage(runner store.Runner) *runnerv1.Runner {
	return &runnerv1.Runner{
		Id:        runner.ID,
		Uuid:      runner.UUID,
		Name:      runner.Name,
		Version:   runner.Version,
		Labels:    runner.Labels,
		Ephemeral: runner.Ephemeral,
		Status:    runnerv1.RunnerStatus_RUNNER_STATUS_IDLE,
	}
}

func (service runnerService) authenticate(ctx context.Context, header http.Header) (store.Runner, error) {
	runner, err := service.server.store.RunnerByUUID(ctx, header.Get("x-runner-uuid"))
	if errors.Is(err, store.ErrNotFound) {
		return runner, unregistered()
	}
	if err != nil {
		return runner, internal(err)
	}
	given := store.HashToken(header.Get("x-runner-token"))
	if subtle.ConstantTimeCompare([]byte(given), []byte(runner.TokenHash)) != 1 {
		return runner, unregistered()
	}
	return runner, nil
}

func (service runnerService) Register(
	ctx context.Context,
	request *connect.Request[runnerv1.RegisterRequest],
) (*connect.Response[runnerv1.RegisterResponse], error) {
	db := service.server.store
	registration, err := db.RunnerTokenBySecret(ctx, strings.TrimSpace(request.Msg.Token))
	if errors.Is(err, store.ErrNotFound) {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unknown registration token"))
	}
	if err != nil {
		return nil, internal(err)
	}
	name := strings.TrimSpace(request.Msg.Name)
	if name == "" || utf8.RuneCountInString(name) > maxRunnerName {
		problem := errors.New("runner name must be 1 to 255 characters")
		return nil, connect.NewError(connect.CodeInvalidArgument, problem)
	}
	secret := randomHex(20)
	actor := store.Actor{User: registration.User, Via: "runner:" + name}
	registered := store.Runner{
		UUID:      newUUID(),
		Name:      name,
		Version:   request.Msg.Version,
		Ephemeral: request.Msg.Ephemeral,
	}
	runner, err := db.CreateRunner(ctx, actor, registered, secret, registration.ID)
	if err != nil {
		return nil, internal(err)
	}
	message := runnerMessage(runner)
	message.Token = secret
	return connect.NewResponse(&runnerv1.RegisterResponse{Runner: message}), nil
}

func (service runnerService) Declare(
	ctx context.Context,
	request *connect.Request[runnerv1.DeclareRequest],
) (*connect.Response[runnerv1.DeclareResponse], error) {
	runner, err := service.authenticate(ctx, request.Header())
	if err != nil {
		return nil, err
	}
	runner, err = service.server.store.DeclareRunner(ctx, runner.ID, request.Msg.Labels, request.Msg.Version)
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&runnerv1.DeclareResponse{Runner: runnerMessage(runner)}), nil
}

func (service runnerService) nextTask(ctx context.Context, header http.Header) (*runnerv1.Task, error) {
	runner, err := service.authenticate(ctx, header)
	if err != nil {
		return nil, err
	}
	if err := service.server.store.TouchRunner(ctx, runner.ID); err != nil {
		return nil, internal(err)
	}
	task, err := service.task(ctx, runner)
	if err != nil {
		return nil, internal(err)
	}
	return task, nil
}

func (service runnerService) FetchTask(
	ctx context.Context,
	request *connect.Request[runnerv1.FetchTaskRequest],
) (*connect.Response[runnerv1.FetchTaskResponse], error) {
	task, err := service.nextTask(ctx, request.Header())
	if err != nil {
		return nil, err
	}
	response := &runnerv1.FetchTaskResponse{Task: task, TasksVersion: request.Msg.TasksVersion}
	return connect.NewResponse(response), nil
}

func (service runnerService) FetchSingleTask(
	ctx context.Context,
	request *connect.Request[runnerv1.FetchSingleTaskRequest],
) (*connect.Response[runnerv1.FetchSingleTaskResponse], error) {
	task, err := service.nextTask(ctx, request.Header())
	if err != nil {
		return nil, err
	}
	response := &runnerv1.FetchSingleTaskResponse{Task: task, TasksVersion: request.Msg.TasksVersion}
	return connect.NewResponse(response), nil
}

func (service runnerService) failStart(ctx context.Context, taskID int64, cause error) error {
	db := service.server.store
	slog.Error("start job", "task", taskID, "error", cause)
	line := store.LogLine{At: time.Now().UTC(), Content: "lyra: the job couldn't start: " + cause.Error()}
	if _, err := db.AppendLog(ctx, taskID, 0, []store.LogLine{line}); err != nil {
		return err
	}
	_, _, err := db.UpdateTask(ctx, taskID, nil, nil, statusFailure)
	return err
}

func (service runnerService) task(ctx context.Context, runner store.Runner) (*runnerv1.Task, error) {
	token := randomHex(20)
	assignment, err := service.server.store.AssignJob(ctx, runner.ID, runner.Labels, store.HashToken(token))
	if err != nil || assignment == nil {
		return nil, err
	}
	task, err := service.buildTask(ctx, *assignment, token)
	if err == nil {
		_, err = proto.Marshal(task)
	}
	if err != nil {
		return nil, service.failStart(ctx, assignment.Task.ID, err)
	}
	return task, nil
}

func refKind(ref string) (string, string) {
	if branch, isBranch := strings.CutPrefix(ref, "refs/heads/"); isBranch {
		return "branch", branch
	}
	if tag, isTag := strings.CutPrefix(ref, "refs/tags/"); isTag {
		return "tag", tag
	}
	return "", strings.TrimPrefix(ref, "refs/pull/")
}

func (service runnerService) taskContext(assignment store.Assignment, token string) (*structpb.Struct, error) {
	run, job := assignment.Run, assignment.Job
	config := service.server.config
	owner, _, _ := strings.Cut(run.Repo, "/")
	event := map[string]any{}
	if err := json.Unmarshal([]byte(run.Payload), &event); err != nil {
		return nil, err
	}
	refType, refName := refKind(run.Ref)
	return structpb.NewStruct(map[string]any{
		"actor":                       run.Author.User,
		"repository":                  run.Repo,
		"repository_owner":            owner,
		"sha":                         run.SHA,
		"ref":                         run.Ref,
		"ref_name":                    refName,
		"ref_type":                    refType,
		"event_name":                  run.Event,
		"event":                       event,
		"run_id":                      fmt.Sprint(run.ID),
		"run_number":                  fmt.Sprint(run.Number),
		"run_attempt":                 fmt.Sprint(job.Attempt),
		"job":                         job.Key,
		"workflow_ref":                run.Repo + "/" + run.Workflow + "@" + run.Ref,
		"token":                       token,
		"retention_days":              fmt.Sprint(int(config.ActionsRetention.Hours() / 24)),
		"base_ref":                    run.BaseRef,
		"head_ref":                    run.HeadRef,
		"forgejo_default_actions_url": config.DefaultActionsURL,
		"gitea_default_actions_url":   config.DefaultActionsURL,
	})
}

func (service runnerService) buildTask(
	ctx context.Context,
	assignment store.Assignment,
	token string,
) (*runnerv1.Task, error) {
	run := assignment.Run
	taskContext, err := service.taskContext(assignment, token)
	if err != nil {
		return nil, err
	}
	needs := map[string]*runnerv1.TaskNeed{}
	for key, need := range assignment.Needs {
		needs[key] = &runnerv1.TaskNeed{Outputs: need.Outputs, Result: resultValue(need.Result)}
	}
	secrets, err := service.server.repoSecrets(ctx, run.Repo)
	if err != nil {
		return nil, err
	}
	secrets["GITHUB_TOKEN"], secrets["FORGEJO_TOKEN"] = token, token
	stored, err := service.server.store.Variables(ctx, run.Repo)
	if err != nil {
		return nil, err
	}
	variables := map[string]string{}
	for _, variable := range stored {
		variables[variable.Name] = variable.Value
	}
	return &runnerv1.Task{
		Id:              assignment.Task.ID,
		WorkflowPayload: []byte(assignment.Job.Payload),
		Context:         taskContext,
		Needs:           needs,
		Vars:            variables,
		Secrets:         secrets,
	}, nil
}

func (service runnerService) ownTask(
	ctx context.Context,
	runner store.Runner,
	id int64,
) (store.Task, store.Run, error) {
	task, run, err := service.server.store.Task(ctx, id)
	if errors.Is(err, store.ErrNotFound) || (err == nil && task.RunnerID != runner.ID) {
		return task, run, connect.NewError(connect.CodeNotFound, fmt.Errorf("task %d not found", id))
	}
	if err != nil {
		return task, run, internal(err)
	}
	return task, run, nil
}

func taskSteps(states []*runnerv1.StepState) []store.TaskStep {
	steps := []store.TaskStep{}
	for _, state := range states {
		step := store.TaskStep{
			ID:        state.Id,
			Result:    resultNames[state.Result],
			LogIndex:  state.LogIndex,
			LogLength: state.LogLength,
		}
		if state.StartedAt != nil {
			started := state.StartedAt.AsTime()
			step.StartedAt = &started
		}
		if state.StoppedAt != nil {
			stopped := state.StoppedAt.AsTime()
			step.StoppedAt = &stopped
		}
		steps = append(steps, step)
	}
	return steps
}

func (service runnerService) UpdateTask(
	ctx context.Context,
	request *connect.Request[runnerv1.UpdateTaskRequest],
) (*connect.Response[runnerv1.UpdateTaskResponse], error) {
	db := service.server.store
	runner, err := service.authenticate(ctx, request.Header())
	if err != nil {
		return nil, err
	}
	state := request.Msg.State
	if _, _, err := service.ownTask(ctx, runner, state.GetId()); err != nil {
		return nil, err
	}
	result := resultNames[state.Result]
	status, stored, err := db.UpdateTask(ctx, state.Id, taskSteps(state.Steps), request.Msg.Outputs, result)
	if err != nil {
		return nil, internal(err)
	}
	if result != "" && runner.Ephemeral {
		actor := store.Actor{User: runner.User, Via: "runner:" + runner.Name}
		if _, err := db.DeleteRunner(ctx, actor, runner.ID); err != nil {
			slog.Error("delete ephemeral runner", "runner", runner.ID, "error", err)
		}
	}
	return connect.NewResponse(&runnerv1.UpdateTaskResponse{
		State:       &runnerv1.TaskState{Id: state.Id, Result: resultValue(status)},
		SentOutputs: stored,
	}), nil
}

func (service runnerService) archiveLog(ctx context.Context, repo string, taskID, length int64) error {
	stored, err := service.server.store.LogLines(ctx, taskID, 0)
	if err != nil {
		return err
	}
	var text bytes.Buffer
	for _, line := range stored {
		text.WriteString(line.At.UTC().Format(time.RFC3339Nano) + "\t" + line.Content + "\n")
	}
	object := logObject(repo, taskID)
	if err := service.server.storage.PutContent(ctx, object, text.Bytes()); err != nil {
		return err
	}
	return service.server.store.ArchiveLog(ctx, taskID, length, object)
}

func (service runnerService) UpdateLog(
	ctx context.Context,
	request *connect.Request[runnerv1.UpdateLogRequest],
) (*connect.Response[runnerv1.UpdateLogResponse], error) {
	runner, err := service.authenticate(ctx, request.Header())
	if err != nil {
		return nil, err
	}
	task, run, err := service.ownTask(ctx, runner, request.Msg.TaskId)
	if err != nil {
		return nil, err
	}
	lines := []store.LogLine{}
	for _, row := range request.Msg.Rows {
		lines = append(lines, store.LogLine{At: row.Time.AsTime(), Content: row.Content})
	}
	ack, err := service.server.store.AppendLog(ctx, task.ID, request.Msg.Index, lines)
	if err != nil {
		return nil, internal(err)
	}
	if request.Msg.NoMore && ack == request.Msg.Index+int64(len(lines)) {
		if err := service.archiveLog(ctx, run.Repo, task.ID, ack); err != nil {
			return nil, internal(err)
		}
	}
	return connect.NewResponse(&runnerv1.UpdateLogResponse{AckIndex: ack}), nil
}

func (service runnerService) UpdateStepSummary(
	ctx context.Context,
	request *connect.Request[runnerv1.UpdateStepSummaryRequest],
) (*connect.Response[runnerv1.UpdateStepSummaryResponse], error) {
	if _, err := service.authenticate(ctx, request.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&runnerv1.UpdateStepSummaryResponse{}), nil
}

func (s *Server) readLog(ctx context.Context, task store.Task, from int64) ([]store.LogLine, error) {
	if task.LogObject == "" {
		return s.store.LogLines(ctx, task.ID, from)
	}
	reader, err := s.storage.Reader(ctx, task.LogObject, 0)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	lines := []store.LogLine{}
	index := int64(-1)
	for raw := range strings.SplitSeq(strings.TrimSuffix(string(content), "\n"), "\n") {
		index++
		if index < from || raw == "" {
			continue
		}
		stamp, text, _ := strings.Cut(raw, "\t")
		at, _ := time.Parse(time.RFC3339Nano, stamp)
		lines = append(lines, store.LogLine{Index: index, At: at, Content: text})
	}
	return lines, nil
}
