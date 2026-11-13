package server

import (
	"encoding/json/jsontext"
	"net/http"
	"reflect"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/muchobien/lyra/internal/store"
)

const apiDescription = `The JSON API behind the lyra CLI. Errors are {"error": "..."} with a 4xx or 5xx status.

Tokens have a scope (read < write < admin) and an optional list of repositories (owner/repo, or owner/*). Each
operation names the scope it needs; "unrestricted" means a token without a repository list. The tailnet identity
has full access.

Lyra-Agent and Lyra-Session headers are recorded with every write, and writes return the event_id they recorded.
While read-only mode is on, every write returns 503.`

func newAPI(router chi.Router) huma.API {
	huma.NewError = newAPIError
	huma.DefaultArrayNullable = false
	config := huma.DefaultConfig("Lyra API", "0")
	config.CreateHooks = nil
	config.DocsPath = ""
	config.SchemasPath = ""
	config.OpenAPIPath = "/api/openapi"
	config.Formats = map[string]huma.Format{
		"application/json":                  jsonFormat,
		"json":                              jsonFormat,
		"application/x-www-form-urlencoded": jsonFormat,
	}
	config.Info.Description = apiDescription
	config.Servers = []*huma.Server{{URL: "/"}}
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"token": {Type: "http", Scheme: "bearer", Description: "A Lyra token."},
		"basic": {
			Type:        "http",
			Scheme:      "basic",
			Description: "Any user name, with a Lyra token as the password. Meant for Git and registry clients.",
		},
		"tailnet": {
			Type:        "apiKey",
			In:          "header",
			Name:        "Tailscale-User-Login",
			Description: "Honoured only from a --trusted-proxy address, for logins in --allow-login.",
		},
	}
	config.Security = []map[string][]string{{"token": {}}, {"basic": {}}, {"tailnet": {}}}
	config.Components.Schemas.RegisterTypeAlias(reflect.TypeFor[jsontext.Value](), reflect.TypeFor[any]())
	return humachi.New(router.With(withRequest), config)
}

func OpenAPI() ([]byte, error) {
	api := newAPI(chi.NewRouter())
	(&Server{}).registerAPI(api)
	return api.OpenAPI().YAML()
}

func (s *Server) registerAPI(api huma.API) {
	s.registerAccount(api)
	s.registerOwners(api)
	s.registerRepos(api)
	s.registerEvents(api)
	s.registerIssues(api)
	s.registerPulls(api)
	s.registerReviews(api)
	s.registerReleases(api)
	s.registerReleaseFiles(api)
	s.registerRuns(api)
	s.registerActionsSettings(api)
	s.registerRunners(api)
	s.registerAdmin(api)
}

func text(contentType, description string) *huma.Response {
	return &huma.Response{
		Description: description,
		Content:     map[string]*huma.MediaType{contentType: {Schema: &huma.Schema{Type: "string"}}},
	}
}

func (s *Server) registerAccount(api huma.API) {
	tags := []string{"Account"}
	register(api, huma.Operation{
		OperationID: "getMe", Method: http.MethodGet, Path: "/api/me", Tags: tags,
		Summary: "The caller's identity", Description: "Scope: any.",
	}, s.handleMe)
	register(api, huma.Operation{
		OperationID: "listTokens", Method: http.MethodGet, Path: "/api/tokens", Tags: tags,
		Summary: "List the caller's tokens", Description: "Scope: admin, unrestricted.",
	}, s.handleListTokens)
	register(api, huma.Operation{
		OperationID: "createToken", Method: http.MethodPost, Path: "/api/tokens", Tags: tags,
		DefaultStatus: http.StatusCreated, Summary: "Create a token",
		Description: "Scope: admin, unrestricted. The secret is returned once; only its SHA-256 hash is stored.",
	}, s.handleCreateToken)
	register(api, huma.Operation{
		OperationID: "revokeToken", Method: http.MethodDelete, Path: "/api/tokens/{id}", Tags: tags,
		Summary: "Revoke a token", Description: "Scope: admin, unrestricted.",
	}, s.handleDeleteToken)
	register(api, huma.Operation{
		OperationID: "listKeys", Method: http.MethodGet, Path: "/api/keys", Tags: tags,
		Summary: "List the caller's SSH keys", Description: "Scope: read, unrestricted.",
	}, s.handleListKeys)
	register(api, huma.Operation{
		OperationID: "createKey", Method: http.MethodPost, Path: "/api/keys", Tags: tags,
		DefaultStatus: http.StatusCreated, Summary: "Register an SSH signing key",
		Description: "Scope: admin, unrestricted. Signing keys verify commit and tag signatures.",
	}, s.handleCreateKey)
	register(api, huma.Operation{
		OperationID: "deleteKey", Method: http.MethodDelete, Path: "/api/keys/{id}", Tags: tags,
		Summary: "Delete an SSH key", Description: "Scope: admin, unrestricted.",
	}, s.handleDeleteKey)
	register(api, huma.Operation{
		OperationID: "getSigningKey", Method: http.MethodGet, Path: "/api/signing-key", Tags: tags,
		Summary:     "Lyra's own commit signing key",
		Description: "Scope: any. Includes an allowed_signers line for git verify-commit.",
	}, s.handleSigningKey)
}

func (s *Server) registerOwners(api huma.API) {
	tags := []string{"Owners"}
	register(api, huma.Operation{
		OperationID: "listOwners", Method: http.MethodGet, Path: "/api/owners", Tags: tags,
		Summary: "List owners with the repositories the caller can read", Description: "Scope: any.",
	}, s.handleListOwners)
	register(api, huma.Operation{
		OperationID: "createOwner", Method: http.MethodPost, Path: "/api/owners", Tags: tags,
		DefaultStatus: http.StatusCreated, Summary: "Create an owner", Description: "Scope: admin, unrestricted.",
	}, s.handleCreateOwner)
	register(api, huma.Operation{
		OperationID: "getOwner", Method: http.MethodGet, Path: "/api/owners/{owner}", Tags: tags,
		Summary: "Get an owner with its repositories", Description: "Scope: any.",
	}, s.handleGetOwner)
	register(api, huma.Operation{
		OperationID: "deleteOwner", Method: http.MethodDelete, Path: "/api/owners/{owner}", Tags: tags,
		Summary: "Delete an owner that has no repositories", Description: "Scope: admin, unrestricted.",
	}, s.handleDeleteOwner)
}

func (s *Server) registerRepos(api huma.API) {
	tags := []string{"Repositories"}
	register(api, huma.Operation{
		OperationID: "listRepos", Method: http.MethodGet, Path: "/api/repos", Tags: tags,
		Summary: "List the repositories the caller can read", Description: "Scope: read.",
	}, s.handleListRepos)
	register(api, huma.Operation{
		OperationID: "createRepo", Method: http.MethodPost, Path: "/api/repos", Tags: tags,
		DefaultStatus: http.StatusCreated, Summary: "Create a repository",
		Description: "Scope: admin. The owner must exist.",
	}, s.handleCreateRepo)
	register(api, huma.Operation{
		OperationID: "getRepo", Method: http.MethodGet, Path: "/api/repos/{owner}/{repo}", Tags: tags,
		Summary: "Get a repository", Description: "Scope: read.",
	}, s.handleGetRepo)
	register(api, huma.Operation{
		OperationID: "deleteRepo", Method: http.MethodDelete, Path: "/api/repos/{owner}/{repo}", Tags: tags,
		Summary: "Delete a repository with its releases and images", Description: "Scope: admin.",
	}, s.handleDeleteRepo)
	register(api, huma.Operation{
		OperationID: "listCommits", Method: http.MethodGet, Path: "/api/repos/{owner}/{repo}/commits", Tags: tags,
		Summary: "List commits", Description: "Scope: read.",
	}, s.handleListCommits)
	register(api, huma.Operation{
		OperationID: "getCommit", Method: http.MethodGet, Path: "/api/repos/{owner}/{repo}/commits/{rev}",
		Tags: tags, Summary: "Get a commit", Description: "Scope: read.",
	}, s.handleGetCommit)
	register(api, huma.Operation{
		OperationID: "listTags", Method: http.MethodGet, Path: "/api/repos/{owner}/{repo}/tags", Tags: tags,
		Summary: "List tags", Description: "Scope: read.",
	}, s.handleListTags)
}

func (s *Server) registerEvents(api huma.API) {
	tags := []string{"Events"}
	register(api, huma.Operation{
		OperationID: "listEvents", Method: http.MethodGet, Path: "/api/events", Tags: tags,
		Summary:     "List recent events",
		Description: "Scope: read. Account events (tokens, keys) need an unrestricted token.",
	}, s.handleListEvents)
	register(api, huma.Operation{
		OperationID: "streamEvents", Method: http.MethodGet, Path: "/api/events/stream", Tags: tags,
		Summary: "Stream events as server-sent events",
		Description: "Scope: read. Each message's id is the event id, its event the action, and its data an Event. " +
			"The stream starts with an id line holding the current position and sends a keepalive every 25 " +
			"seconds. Tokens see only events for repositories they can read. Every event delivered on an item " +
			"stream moves the position since=mine starts from.",
		Responses: map[string]*huma.Response{"200": text("text/event-stream", "Server-sent events")},
	}, stream[eventStreamInput](s.handleEventStream))
}

func (s *Server) registerIssues(api huma.API) {
	tags := []string{"Issues"}
	issues := "/api/repos/{owner}/{repo}/issues"
	labels := "/api/repos/{owner}/{repo}/labels"
	register(api, huma.Operation{
		OperationID: "listIssues", Method: http.MethodGet, Path: issues, Tags: tags,
		Summary: "List issues", Description: "Scope: read.",
	}, s.handleListIssues)
	register(api, huma.Operation{
		OperationID: "createIssue", Method: http.MethodPost, Path: issues, Tags: tags,
		DefaultStatus: http.StatusCreated, Summary: "Open an issue", Description: "Scope: write.",
	}, s.handleCreateIssue)
	register(api, huma.Operation{
		OperationID: "getIssue", Method: http.MethodGet, Path: issues + "/{number}", Tags: tags,
		Summary: "Get an issue with its comments", Description: "Scope: read.",
	}, s.handleGetIssue)
	register(api, huma.Operation{
		OperationID: "editIssue", Method: http.MethodPatch, Path: issues + "/{number}", Tags: tags,
		Summary:     "Edit an issue",
		Description: "Scope: write. Each change is recorded as its own event; event_id is the last one.",
	}, s.handleEditIssue)
	register(api, huma.Operation{
		OperationID: "commentIssue", Method: http.MethodPost, Path: issues + "/{number}/comments", Tags: tags,
		DefaultStatus: http.StatusCreated, Summary: "Comment on an issue", Description: "Scope: write.",
	}, s.handleCommentItem(kindIssue))
	register(api, huma.Operation{
		OperationID: "listLabels", Method: http.MethodGet, Path: labels, Tags: tags,
		Summary: "List labels", Description: "Scope: read.",
	}, s.handleListLabels)
	register(api, huma.Operation{
		OperationID: "createLabel", Method: http.MethodPost, Path: labels, Tags: tags,
		DefaultStatus: http.StatusCreated, Summary: "Create a label", Description: "Scope: write.",
	}, s.handleCreateLabel)
	register(api, huma.Operation{
		OperationID: "deleteLabel", Method: http.MethodDelete, Path: labels + "/{label}", Tags: tags,
		Summary: "Delete a label", Description: "Scope: write.",
	}, s.handleDeleteLabel)
}

func (s *Server) registerPulls(api huma.API) {
	tags := []string{"Pull requests"}
	pulls := "/api/repos/{owner}/{repo}/pulls"
	register(api, huma.Operation{
		OperationID: "listPulls", Method: http.MethodGet, Path: pulls, Tags: tags,
		Summary: "List pull requests", Description: "Scope: read.",
	}, s.handleListPulls)
	register(api, huma.Operation{
		OperationID: "createPull", Method: http.MethodPost, Path: pulls, Tags: tags,
		DefaultStatus: http.StatusCreated, Summary: "Open a pull request",
		Description: "Scope: write. actions lists the Actions runs it queued.",
	}, s.handleCreatePull)
	register(api, huma.Operation{
		OperationID: "getPull", Method: http.MethodGet, Path: pulls + "/{number}", Tags: tags,
		Summary:     "Get a pull request with its merge status, reviews, comments and checks",
		Description: "Scope: read.",
	}, s.handleGetPull)
	register(api, huma.Operation{
		OperationID: "editPull", Method: http.MethodPatch, Path: pulls + "/{number}", Tags: tags,
		Summary: "Edit a pull request",
		Description: "Scope: write. Each change is recorded as its own event; event_id is the last one. " +
			"Reopening syncs the pull request with its head branch, and is refused if that branch was deleted.",
	}, s.handleEditPull)
	register(api, huma.Operation{
		OperationID: "commentPull", Method: http.MethodPost, Path: pulls + "/{number}/comments", Tags: tags,
		DefaultStatus: http.StatusCreated, Summary: "Comment on a pull request, optionally on a line of its diff",
		Description: "Scope: write.",
	}, s.handleCommentItem(kindPull))
	register(api, huma.Operation{
		OperationID: "listPullCommits", Method: http.MethodGet, Path: pulls + "/{number}/commits", Tags: tags,
		Summary: "List a pull request's commits", Description: "Scope: read.",
	}, s.handlePullCommits)
	register(api, huma.Operation{
		OperationID: "getPullDiff", Method: http.MethodGet, Path: pulls + "/{number}/diff", Tags: tags,
		Summary: "The unified diff against the merge base", Description: "Scope: read.",
		Responses: map[string]*huma.Response{"200": text("text/x-diff", "The diff")},
	}, s.handlePullDiff)
	register(api, huma.Operation{
		OperationID: "listPullFiles", Method: http.MethodGet, Path: pulls + "/{number}/files", Tags: tags,
		Summary: "List changed files", Description: "Scope: read.",
	}, s.handlePullFiles)
}

func (s *Server) registerReviews(api huma.API) {
	pulls := "/api/repos/{owner}/{repo}/pulls"
	register(api, huma.Operation{
		OperationID: "mergePull", Method: http.MethodPost, Path: pulls + "/{number}/merge",
		Tags: []string{"Pull requests"}, Summary: "Merge a pull request",
		Description: "Scope: write. Returns 409 on conflicts, when a branch moved, or when head_sha doesn't " +
			"match the head branch.",
	}, s.handleMergePull)
	register(api, huma.Operation{
		OperationID: "reviewPull", Method: http.MethodPost, Path: pulls + "/{number}/reviews",
		Tags: []string{"Pull requests"}, DefaultStatus: http.StatusCreated,
		Summary:     "Review a pull request's current head commit",
		Description: "Scope: write. Authors can't approve or request changes on their own pull requests.",
	}, s.handleReviewPull)
	rules := "/api/repos/{owner}/{repo}/rules"
	tags := []string{"Rules"}
	register(api, huma.Operation{
		OperationID: "listRules", Method: http.MethodGet, Path: rules, Tags: tags,
		Summary: "List branch rules", Description: "Scope: read.",
	}, s.handleListRules)
	register(api, huma.Operation{
		OperationID: "setRule", Method: http.MethodPost, Path: rules, Tags: tags,
		Summary: "Create or replace the rule for a branch pattern", Description: "Scope: admin.",
	}, s.handleSetRule)
	register(api, huma.Operation{
		OperationID: "deleteRule", Method: http.MethodDelete, Path: rules, Tags: tags,
		Summary: "Delete a branch rule", Description: "Scope: admin.",
	}, s.handleDeleteRule)
}

func pathParams(names ...string) []*huma.Param {
	params := []*huma.Param{}
	for _, name := range names {
		schema := &huma.Schema{Type: "string"}
		params = append(params, &huma.Param{Name: name, In: "path", Required: true, Schema: schema})
	}
	return params
}

func (s *Server) registerReleases(api huma.API) {
	tags := []string{"Releases"}
	releases := "/api/repos/{owner}/{repo}/releases"
	register(api, huma.Operation{
		OperationID: "listReleases", Method: http.MethodGet, Path: releases, Tags: tags,
		Summary: "List releases", Description: "Scope: read. Drafts are listed only with write access.",
	}, s.handleListReleases)
	register(api, huma.Operation{
		OperationID: "createRelease", Method: http.MethodPost, Path: releases, Tags: tags,
		DefaultStatus: http.StatusCreated, Summary: "Create a release",
		Description: "Scope: write. Publishing creates the tag at target if it doesn't exist; an existing tag is " +
			"never moved, so target must match it. actions lists the Actions runs it queued.",
	}, s.handleCreateRelease)
	register(api, huma.Operation{
		OperationID: "getLatestRelease", Method: http.MethodGet, Path: releases + "/latest", Tags: tags,
		Summary: "The most recently published release that isn't a prerelease", Description: "Scope: read.",
	}, s.handleLatestRelease)
	register(api, huma.Operation{
		OperationID: "previewReleaseNotes", Method: http.MethodGet, Path: releases + "/notes", Tags: tags,
		Summary: "Preview generated release notes", Description: "Scope: read.",
	}, s.handleReleaseNotes)
	api.OpenAPI().AddOperation(&huma.Operation{
		OperationID: "getReleaseByTag", Method: http.MethodGet, Path: releases + "/tags/{tag}", Tags: tags,
		Summary: "Get a release by tag", Description: "Scope: read. The tag may contain slashes.",
		Parameters: pathParams("owner", "repo", "tag"),
		Responses: map[string]*huma.Response{"200": {
			Description: "OK",
			Content: map[string]*huma.MediaType{"application/json": {
				Schema: api.OpenAPI().Components.Schemas.Schema(reflect.TypeFor[store.Release](), true, ""),
			}},
		}},
	})
	register(api, huma.Operation{
		OperationID: "getRelease", Method: http.MethodGet, Path: releases + "/{id}", Tags: tags,
		Summary: "Get a release", Description: "Scope: read.",
	}, s.handleGetRelease)
	register(api, huma.Operation{
		OperationID: "editRelease", Method: http.MethodPatch, Path: releases + "/{id}", Tags: tags,
		Summary: "Edit or publish a release", Description: "Scope: write. draft: false publishes it.",
	}, s.handleEditRelease)
	register(api, huma.Operation{
		OperationID: "deleteRelease", Method: http.MethodDelete, Path: releases + "/{id}", Tags: tags,
		Summary: "Delete a release and its assets", Description: "Scope: write.",
	}, s.handleDeleteRelease)
}

func (s *Server) registerReleaseFiles(api huma.API) {
	tags := []string{"Releases"}
	assets := "/api/repos/{owner}/{repo}/releases/{id}/assets"
	file := map[string]*huma.Response{"200": text("application/octet-stream", "The file; its ETag is its sha256")}
	register(api, huma.Operation{
		OperationID: "uploadAsset", Method: http.MethodPost, Path: assets, Tags: tags,
		DefaultStatus: http.StatusCreated, Summary: "Upload an asset",
		Description: "Scope: write. The request body is the file; its Content-Type is stored with it.",
		RequestBody: &huma.RequestBody{Required: true, Content: map[string]*huma.MediaType{
			"application/octet-stream": {Schema: &huma.Schema{Type: "string", Format: "binary"}},
		}},
		Responses: map[string]*huma.Response{"201": {
			Description: "Created",
			Content: map[string]*huma.MediaType{"application/json": {
				Schema: api.OpenAPI().Components.Schemas.Schema(reflect.TypeFor[assetWrite](), true, ""),
			}},
		}},
	}, stream[uploadAssetInput](s.handleUploadAsset))
	register(api, huma.Operation{
		OperationID: "downloadAsset", Method: http.MethodGet, Path: assets + "/{name}", Tags: tags,
		Summary: "Download an asset", Description: "Scope: read. Supports range requests.", Responses: file,
	}, stream[assetInput](s.handleDownloadAsset))
	register(api, huma.Operation{
		OperationID: "deleteAsset", Method: http.MethodDelete, Path: assets + "/{name}", Tags: tags,
		Summary: "Delete an asset", Description: "Scope: write.",
	}, s.handleDeleteAsset)
	api.OpenAPI().AddOperation(&huma.Operation{
		OperationID: "downloadAssetByTag", Method: http.MethodGet,
		Path: "/{owner}/{repo}/releases/download/{tag}/{name}", Tags: tags, Summary: "Download an asset by tag",
		Description: "Scope: read. A stable URL with range support; the tag may contain slashes.",
		Parameters:  pathParams("owner", "repo", "tag", "name"), Responses: file,
	})
	api.OpenAPI().AddOperation(&huma.Operation{
		OperationID: "downloadLatestAsset", Method: http.MethodGet,
		Path: "/{owner}/{repo}/releases/latest/download/{name}", Tags: tags,
		Summary: "Download an asset of the latest release", Description: "Scope: read.",
		Parameters: pathParams("owner", "repo", "name"), Responses: file,
	})
	register(api, huma.Operation{
		OperationID: "listImages", Method: http.MethodGet, Path: "/api/repos/{owner}/{repo}/images",
		Tags: []string{"Images"}, Summary: "List container image tags",
		Description: "Scope: read. Images are pushed and pulled through the OCI distribution API at /v2/.",
	}, s.handleListImages)
}

func (s *Server) registerRuns(api huma.API) {
	tags := []string{"Actions"}
	runs := "/api/repos/{owner}/{repo}/runs"
	register(api, huma.Operation{
		OperationID: "listRuns", Method: http.MethodGet, Path: runs, Tags: tags,
		Summary: "List Actions runs", Description: "Scope: read.",
	}, s.handleListRuns)
	register(api, huma.Operation{
		OperationID: "getRun", Method: http.MethodGet, Path: runs + "/{number}", Tags: tags,
		Summary: "Get a run with its jobs and steps", Description: "Scope: read.",
	}, s.handleGetRun)
	register(api, huma.Operation{
		OperationID: "getJobLogs", Method: http.MethodGet, Path: runs + "/{number}/jobs/{job}/logs", Tags: tags,
		Summary: "A job's log lines", Description: "Scope: read.",
		Responses: map[string]*huma.Response{"200": text("text/plain", "One log line per line")},
	}, s.handleRunLogs)
	register(api, huma.Operation{
		OperationID: "cancelRun", Method: http.MethodPost, Path: runs + "/{number}/cancel", Tags: tags,
		Summary: "Cancel a run", Description: "Scope: write.",
	}, s.handleCancelRun)
	register(api, huma.Operation{
		OperationID: "rerunRun", Method: http.MethodPost, Path: runs + "/{number}/rerun", Tags: tags,
		Summary: "Rerun a run", Description: "Scope: write.",
	}, s.handleRerunRun)
	register(api, huma.Operation{
		OperationID: "dispatchWorkflow", Method: http.MethodPost, Path: "/api/repos/{owner}/{repo}/workflows/dispatch",
		Tags: tags, DefaultStatus: http.StatusCreated, Summary: "Run a workflow with workflow_dispatch",
		Description: "Scope: write. Unknown inputs are refused, and choice and boolean values are checked.",
	}, s.handleDispatch)
}

func (s *Server) registerActionsSettings(api huma.API) {
	tags := []string{"Actions"}
	secrets := "/api/repos/{owner}/{repo}/secrets"
	variables := "/api/repos/{owner}/{repo}/variables"
	register(api, huma.Operation{
		OperationID: "listSecrets", Method: http.MethodGet, Path: secrets, Tags: tags,
		Summary: "List secrets, without their values", Description: "Scope: admin.",
	}, s.handleListSecrets)
	register(api, huma.Operation{
		OperationID: "setSecret", Method: http.MethodPut, Path: secrets + "/{name}", Tags: tags,
		Summary: "Set a secret", Description: "Scope: admin.",
	}, s.handleSetSecret)
	register(api, huma.Operation{
		OperationID: "deleteSecret", Method: http.MethodDelete, Path: secrets + "/{name}", Tags: tags,
		Summary: "Delete a secret", Description: "Scope: admin.",
	}, s.handleDeleteSecret)
	register(api, huma.Operation{
		OperationID: "listVariables", Method: http.MethodGet, Path: variables, Tags: tags,
		Summary: "List variables", Description: "Scope: read.",
	}, s.handleListVariables)
	register(api, huma.Operation{
		OperationID: "setVariable", Method: http.MethodPut, Path: variables + "/{name}", Tags: tags,
		Summary: "Set a variable", Description: "Scope: write.",
	}, s.handleSetVariable)
	register(api, huma.Operation{
		OperationID: "deleteVariable", Method: http.MethodDelete, Path: variables + "/{name}", Tags: tags,
		Summary: "Delete a variable", Description: "Scope: write.",
	}, s.handleDeleteVariable)
}

func (s *Server) registerRunners(api huma.API) {
	tags := []string{"Runners"}
	register(api, huma.Operation{
		OperationID: "listRunners", Method: http.MethodGet, Path: "/api/runners", Tags: tags,
		Summary:     "List runners",
		Description: "Scope: admin, unrestricted. online means seen in the last minute.",
	}, s.handleListRunners)
	register(api, huma.Operation{
		OperationID: "createRunner", Method: http.MethodPost, Path: "/api/runners", Tags: tags,
		DefaultStatus: http.StatusCreated, Summary: "Create a runner for a config-file connection",
		Description: "Scope: admin, unrestricted. The token is returned once.",
	}, s.handleCreateRunner)
	register(api, huma.Operation{
		OperationID: "deleteRunner", Method: http.MethodDelete, Path: "/api/runners/{id}", Tags: tags,
		Summary: "Delete a runner", Description: "Scope: admin, unrestricted.",
	}, s.handleDeleteRunner)
	register(api, huma.Operation{
		OperationID: "listRunnerTokens", Method: http.MethodGet, Path: "/api/runners/tokens", Tags: tags,
		Summary: "List registration tokens", Description: "Scope: admin, unrestricted.",
	}, s.handleListRunnerTokens)
	register(api, huma.Operation{
		OperationID: "createRunnerToken", Method: http.MethodPost, Path: "/api/runners/tokens", Tags: tags,
		DefaultStatus: http.StatusCreated, Summary: "Create a registration token for forgejo-runner register",
		Description: "Scope: admin, unrestricted. The secret is returned once.",
	}, s.handleCreateRunnerToken)
	register(api, huma.Operation{
		OperationID: "deleteRunnerToken", Method: http.MethodDelete, Path: "/api/runners/tokens/{id}", Tags: tags,
		Summary:     "Delete a registration token",
		Description: "Scope: admin, unrestricted. Runners it registered are kept.",
	}, s.handleDeleteRunnerToken)
}

func (s *Server) registerAdmin(api huma.API) {
	tags := []string{"Admin"}
	register(api, huma.Operation{
		OperationID: "createBackup", Method: http.MethodPost, Path: "/api/admin/backup", Tags: tags,
		Summary:     "Stream a backup as a tar",
		Description: "Scope: admin, unrestricted. Writes wait while the database and repositories are copied.",
		Responses:   map[string]*huma.Response{"200": text("application/x-tar", "The backup")},
	}, stream[struct{}](s.handleBackup))
	register(api, huma.Operation{
		OperationID: "setReadOnly", Method: http.MethodPost, Path: "/api/admin/readonly", Tags: tags,
		Summary:     "Turn read-only mode on or off",
		Description: "Scope: admin, unrestricted. The mode lives in memory, so a restart turns it off.",
	}, s.handleReadOnly)
	register(api, huma.Operation{
		OperationID: "getStatus", Method: http.MethodGet, Path: "/api/admin/status", Tags: tags,
		Summary: "Read-only mode and the last backup", Description: "Scope: admin, unrestricted.",
	}, s.handleStatus)
	register(api, huma.Operation{
		OperationID: "checkStorage", Method: http.MethodGet, Path: "/api/admin/storage-check", Tags: tags,
		Summary:     "Stored files the database refers to but storage lacks",
		Description: "Scope: admin, unrestricted.",
	}, s.handleStorageCheck)
	register(api, huma.Operation{
		OperationID: "collectRegistry", Method: http.MethodPost, Path: "/api/admin/registry-gc", Tags: tags,
		Summary:     "Delete registry layers and manifests no tag refers to",
		Description: "Scope: admin, unrestricted. Lyra is read-only while it runs.",
	}, s.handleRegistryGC)
}
