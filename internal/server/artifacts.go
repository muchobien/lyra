package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json/v2"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/distribution/distribution/v3/registry/storage/driver"
	"github.com/go-chi/chi/v5"

	"github.com/muchobien/lyra/internal/store"
)

const (
	artifactService = "/twirp/github.actions.results.api.v1.ArtifactService/"
	artifactPending = "pending"
	artifactReady   = "ready"
)

type runtimeClaims struct {
	Scope string `json:"scp"`
	Task  int64  `json:"task"`
}

type artifactRequest struct {
	RunID      string     `json:"workflow_run_backend_id"`
	Name       string     `json:"name"`
	ExpiresAt  *time.Time `json:"expires_at,omitzero"`
	Size       int64      `json:"size,omitzero,string"`
	Hash       string     `json:"hash,omitzero"`
	NameFilter string     `json:"name_filter,omitzero"`
	IDFilter   int64      `json:"id_filter,omitzero,string"`
}

type artifactWritten struct {
	OK         bool  `json:"ok"`
	ArtifactID int64 `json:"artifact_id,string"`
}

type monolithArtifact struct {
	RunID      string    `json:"workflow_run_backend_id"`
	JobID      string    `json:"workflow_job_run_backend_id"`
	DatabaseID int64     `json:"database_id,string"`
	Name       string    `json:"name"`
	Size       int64     `json:"size,string"`
	CreatedAt  time.Time `json:"created_at"`
}

func artifactsPath(repo string) string {
	return "/lyra/artifacts/" + repo
}

func artifactPath(repo string, artifact store.Artifact) string {
	return artifactsPath(repo) + "/" + artifact.Object
}

func (s *Server) artifactMAC(message string) string {
	key := hmac.New(sha256.New, s.secretsKey)
	key.Write([]byte("lyra actions artifacts"))
	mac := hmac.New(sha256.New, key.Sum(nil))
	mac.Write([]byte(message))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) runtimeToken(runID, jobID, taskID int64) (string, error) {
	claims, err := json.Marshal(runtimeClaims{Scope: fmt.Sprintf("Actions.Results:%d:%d", runID, jobID), Task: taskID})
	if err != nil {
		return "", err
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	unsigned := header + "." + base64.RawURLEncoding.EncodeToString(claims)
	return unsigned + "." + s.artifactMAC(unsigned), nil
}

func (s *Server) runningTask(ctx context.Context, id int64) (store.Task, store.Run, error) {
	task, run, err := s.store.Task(ctx, id)
	if err != nil || task.Status != statusRunning {
		return task, run, forbidden("the job isn't running")
	}
	return task, run, nil
}

func (s *Server) runtimeTask(r *http.Request) (store.Task, store.Run, error) {
	token, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	dot := strings.LastIndex(token, ".")
	if dot < 0 || !hmac.Equal([]byte(token[dot+1:]), []byte(s.artifactMAC(token[:dot]))) {
		return store.Task{}, store.Run{}, forbidden("invalid runtime token")
	}
	_, payload, _ := strings.Cut(token[:dot], ".")
	var claims runtimeClaims
	decoded, err := base64.RawURLEncoding.DecodeString(payload)
	if err == nil {
		err = json.Unmarshal(decoded, &claims)
	}
	if err != nil {
		return store.Task{}, store.Run{}, forbidden("invalid runtime token")
	}
	return s.runningTask(r.Context(), claims.Task)
}

func (s *Server) artifactURL(r *http.Request, method string, artifactID, taskID int64) string {
	query := url.Values{
		"artifact": {strconv.FormatInt(artifactID, 10)},
		"task":     {strconv.FormatInt(taskID, 10)},
		"expires":  {strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)},
	}
	query.Set("sig", s.artifactMAC(method+"\n"+query.Encode()))
	return baseURL(r) + artifactService + method + "?" + query.Encode()
}

func (s *Server) signedArtifact(r *http.Request, method string) (store.Artifact, store.Run, error) {
	query := r.URL.Query()
	signed := url.Values{
		"artifact": {query.Get("artifact")},
		"task":     {query.Get("task")},
		"expires":  {query.Get("expires")},
	}
	if !hmac.Equal([]byte(query.Get("sig")), []byte(s.artifactMAC(method+"\n"+signed.Encode()))) {
		return store.Artifact{}, store.Run{}, forbidden("invalid signature")
	}
	expires, _ := strconv.ParseInt(query.Get("expires"), 10, 64)
	if time.Now().Unix() > expires {
		return store.Artifact{}, store.Run{}, forbidden("the URL expired")
	}
	taskID, _ := strconv.ParseInt(query.Get("task"), 10, 64)
	_, run, err := s.runningTask(r.Context(), taskID)
	if err != nil {
		return store.Artifact{}, run, err
	}
	artifactID, _ := strconv.ParseInt(query.Get("artifact"), 10, 64)
	artifact, err := s.store.ArtifactByID(r.Context(), artifactID)
	if err != nil || artifact.RunID != run.ID {
		return artifact, run, notFound("artifact not found")
	}
	return artifact, run, nil
}

func (s *Server) deleteArtifactObjects(ctx context.Context, repo string, artifact store.Artifact) {
	path := artifactPath(repo, artifact)
	for _, object := range []string{path, path + ".blocks"} {
		err := s.storage.Delete(ctx, object)
		if _, absent := errors.AsType[driver.PathNotFoundError](err); err != nil && !absent {
			slog.Error("delete artifact", "repo", repo, "artifact", artifact.Name, "object", object, "error", err)
		}
	}
}

func writeTwirpError(w http.ResponseWriter, err error) {
	status, message := errorStatus(err)
	codes := map[int]string{
		http.StatusBadRequest: "invalid_argument",
		http.StatusForbidden:  "permission_denied",
		http.StatusNotFound:   "not_found",
		http.StatusConflict:   "already_exists",
	}
	code, known := codes[status]
	if !known {
		code = "internal"
	}
	if err := writeJSON(w, status, map[string]string{"code": code, "msg": message}); err != nil {
		slog.Error("write artifact error response", "status", status, "error", err)
	}
}

func (s *Server) handleArtifactService(w http.ResponseWriter, r *http.Request) {
	response, err := s.artifactCall(r)
	if err != nil {
		writeTwirpError(w, err)
		return
	}
	if err := writeJSON(w, http.StatusOK, response); err != nil {
		slog.Error("write artifact response", "error", err)
	}
}

func (s *Server) artifactCall(r *http.Request) (any, error) {
	task, run, err := s.runtimeTask(r)
	if err != nil {
		return nil, err
	}
	var request artifactRequest
	if err := json.UnmarshalRead(io.LimitReader(r.Body, 1<<20), &request); err != nil {
		return nil, badRequest("request: " + err.Error())
	}
	if request.RunID != strconv.FormatInt(run.ID, 10) {
		return nil, forbidden("the runtime token is for another run")
	}
	switch chi.URLParam(r, "method") {
	case "CreateArtifact":
		return s.createArtifact(r, task, run, request)
	case "FinalizeArtifact":
		return s.finalizeArtifact(r.Context(), run, request)
	case "ListArtifacts":
		return s.listRuntimeArtifacts(r.Context(), run, request)
	case "GetSignedArtifactURL":
		artifact, err := s.store.Artifact(r.Context(), run.ID, request.Name)
		if err != nil || artifact.Status != artifactReady {
			return nil, notFound("artifact " + request.Name + " not found")
		}
		return map[string]string{"signed_url": s.artifactURL(r, "DownloadArtifact", artifact.ID, task.ID)}, nil
	case "DeleteArtifact":
		artifact, err := s.store.Artifact(r.Context(), run.ID, request.Name)
		if err != nil {
			return nil, missing(err, "artifact "+request.Name+" not found")
		}
		if err := s.store.DeleteArtifact(r.Context(), artifact.ID); err != nil {
			return nil, err
		}
		s.deleteArtifactObjects(r.Context(), run.Repo, artifact)
		return artifactWritten{OK: true, ArtifactID: artifact.ID}, nil
	}
	return nil, notFound("no such method")
}

func (s *Server) createArtifact(r *http.Request, task store.Task, run store.Run, request artifactRequest) (any, error) {
	if !validAssetName(request.Name) {
		return nil, badRequest("artifact name must be 1 to 255 characters, " +
			"with no slashes, control characters or surrounding spaces")
	}
	created, replaced, err := s.store.CreateArtifact(r.Context(), store.Artifact{
		RunID:     run.ID,
		JobID:     task.JobID,
		Name:      request.Name,
		Object:    randomHex(16),
		ExpiresAt: request.ExpiresAt,
	})
	if errors.Is(err, store.ErrExists) {
		return nil, conflict(fmt.Sprintf("another job of run #%d already uploaded %s", run.Number, request.Name))
	}
	if err != nil {
		return nil, err
	}
	if replaced != nil {
		s.deleteArtifactObjects(r.Context(), run.Repo, *replaced)
	}
	return map[string]any{
		"ok":                true,
		"signed_upload_url": s.artifactURL(r, "UploadArtifact", created.ID, task.ID),
	}, nil
}

func (s *Server) finalizeArtifact(ctx context.Context, run store.Run, request artifactRequest) (any, error) {
	artifact, err := s.store.Artifact(ctx, run.ID, request.Name)
	if err != nil {
		return nil, missing(err, "artifact "+request.Name+" not found")
	}
	if artifact.Status != artifactPending || artifact.SHA256 == "" {
		return nil, badRequest("artifact " + request.Name + " has no uploaded content to finalize")
	}
	if request.Size != artifact.Size {
		return nil, badRequest(fmt.Sprintf("uploaded %d bytes, but the size given is %d", artifact.Size, request.Size))
	}
	if request.Hash != "" && request.Hash != "sha256:"+artifact.SHA256 {
		return nil, badRequest("the uploaded content's sha256 is " + artifact.SHA256 + ", not " + request.Hash)
	}
	artifact.Status = artifactReady
	if err := s.store.UpdateArtifact(ctx, artifact); err != nil {
		return nil, err
	}
	return artifactWritten{OK: true, ArtifactID: artifact.ID}, nil
}

func (s *Server) listRuntimeArtifacts(ctx context.Context, run store.Run, request artifactRequest) (any, error) {
	artifacts, err := s.store.Artifacts(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	listed := []monolithArtifact{}
	for _, artifact := range artifacts {
		if request.NameFilter != "" && artifact.Name != request.NameFilter ||
			request.IDFilter != 0 && artifact.ID != request.IDFilter {
			continue
		}
		listed = append(listed, monolithArtifact{
			RunID:      request.RunID,
			JobID:      strconv.FormatInt(artifact.JobID, 10),
			DatabaseID: artifact.ID,
			Name:       artifact.Name,
			Size:       artifact.Size,
			CreatedAt:  artifact.CreatedAt,
		})
	}
	return map[string][]monolithArtifact{"artifacts": listed}, nil
}

func (s *Server) handleArtifactUpload(w http.ResponseWriter, r *http.Request) {
	if err := s.uploadArtifact(r); err != nil {
		writeTwirpError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) uploadArtifact(r *http.Request) error {
	artifact, run, err := s.signedArtifact(r, "UploadArtifact")
	if err != nil {
		return err
	}
	if artifact.Status != artifactPending {
		return conflict("artifact " + artifact.Name + " is already finalized")
	}
	path := artifactPath(run.Repo, artifact)
	switch r.URL.Query().Get("comp") {
	case "block":
		block := r.URL.Query().Get("blockid")
		if block == "" {
			return badRequest("blockid is required")
		}
		_, _, err := s.storeObject(r.Context(), path+".blocks/"+hex.EncodeToString([]byte(block)), r.Body)
		return err
	case "blocklist":
		var list struct {
			Latest []string `xml:"Latest"`
		}
		if err := xml.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&list); err != nil {
			return badRequest("block list: " + err.Error())
		}
		return s.assembleArtifact(r.Context(), path, artifact, list.Latest)
	}
	return badRequest("comp must be block or blocklist")
}

func (s *Server) assembleArtifact(ctx context.Context, path string, artifact store.Artifact, blocks []string) error {
	writer, err := s.storage.Writer(ctx, path, false)
	if err != nil {
		return err
	}
	hash := sha256.New()
	artifact.Size = 0
	for _, block := range blocks {
		reader, err := s.storage.Reader(ctx, path+".blocks/"+hex.EncodeToString([]byte(block)), 0)
		if err != nil {
			writer.Cancel(ctx)
			writer.Close()
			return badRequest("block " + block + " wasn't uploaded")
		}
		written, err := io.Copy(io.MultiWriter(writer, hash), reader)
		reader.Close()
		artifact.Size += written
		if err != nil {
			writer.Cancel(ctx)
			writer.Close()
			return err
		}
	}
	if err := writer.Commit(ctx); err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	err = s.storage.Delete(ctx, path+".blocks")
	if _, absent := errors.AsType[driver.PathNotFoundError](err); err != nil && !absent {
		slog.Error("delete artifact blocks", "object", path, "error", err)
	}
	artifact.SHA256 = hex.EncodeToString(hash.Sum(nil))
	return s.store.UpdateArtifact(ctx, artifact)
}

func (s *Server) handleArtifactDownload(w http.ResponseWriter, r *http.Request) {
	artifact, run, err := s.signedArtifact(r, "DownloadArtifact")
	if err == nil && artifact.Status != artifactReady {
		err = notFound("artifact not found")
	}
	if err != nil {
		writeTwirpError(w, err)
		return
	}
	s.serveArtifact(w, r, run.Repo, artifact)
}

func (s *Server) serveArtifact(w http.ResponseWriter, r *http.Request, repo string, artifact store.Artifact) {
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": artifact.Name + ".zip"})
	w.Header().Set("ETag", `"`+artifact.SHA256+`"`)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", disposition)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	file := &storageFile{
		ctx:     r.Context(),
		storage: s.storage,
		path:    artifactPath(repo, artifact),
		size:    artifact.Size,
	}
	defer file.Close()
	http.ServeContent(w, r, "", artifact.CreatedAt, file)
}

type artifactInput struct {
	RunPath
	Name string `path:"name"`
}

func (s *Server) handleListArtifacts(r *http.Request, input *RunPath) (*body[[]store.Artifact], error) {
	run, err := s.lookupRun(r, scopeRead, *input)
	if err != nil {
		return nil, err
	}
	artifacts, err := s.store.Artifacts(r.Context(), run.ID)
	if err != nil {
		return nil, err
	}
	return reply(artifacts)
}

func (s *Server) handleDownloadArtifact(w http.ResponseWriter, r *http.Request, input *artifactInput) error {
	run, err := s.lookupRun(r, scopeRead, input.RunPath)
	if err != nil {
		return err
	}
	artifact, err := s.store.Artifact(r.Context(), run.ID, input.Name)
	if err != nil || artifact.Status != artifactReady {
		return notFound(fmt.Sprintf("artifact %s not found in run #%d", input.Name, run.Number))
	}
	s.serveArtifact(w, r, run.Repo, artifact)
	return nil
}

func (s *Server) pruneArtifacts(ctx context.Context) {
	now := time.Now().UTC()
	expired, err := s.store.ExpiredArtifacts(ctx, now, now.Add(-24*time.Hour))
	if err != nil {
		slog.Error("expired artifacts", "error", err)
		return
	}
	for _, artifact := range expired {
		s.deleteArtifactObjects(ctx, artifact.Repo, artifact)
		if err := s.store.DeleteArtifact(ctx, artifact.ID); err != nil {
			slog.Error("delete artifact", "artifact", artifact.ID, "error", err)
		}
	}
	if len(expired) > 0 {
		slog.Info("deleted expired artifacts", "count", len(expired))
	}
}
