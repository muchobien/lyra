package server

import (
	"net/http"

	"code.forgejo.org/forgejo/actions-proto/ping/v1/pingv1connect"
	"code.forgejo.org/forgejo/actions-proto/runner/v1/runnerv1connect"
	"github.com/go-chi/chi/v5"

	"github.com/muchobien/lyra/internal/webui"
)

func headAsGet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			chi.RouteContext(r.Context()).RouteMethod = http.MethodGet
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Handler() http.Handler {
	router := chi.NewRouter()
	router.Use(
		http.NewCrossOriginProtection().Handler,
		headAsGet,
		s.instrument,
		s.authenticate,
		s.writeGate,
	)
	router.Handle("/v2/*", http.HandlerFunc(s.handleRegistry))
	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok\n"))
	})
	router.Route("/api", s.apiRoutes)
	router.Handle("/static/assets/*", http.StripPrefix("/static/", webui.Assets()))
	router.Get("/", s.page(s.uiApplication))
	router.Get("/{owner}", s.page(s.uiApplication))
	router.Route("/{owner}/{repo}", s.uiRoutes)
	return router
}

func (s *Server) apiRoutes(router chi.Router) {
	router.Get("/me", api(s.handleMe))
	router.Get("/code.css", api(s.handleCodeCSS))
	router.Post("/markdown", api(s.handleMarkdown))
	router.Route("/repos", func(repos chi.Router) {
		repos.Get("/", api(s.handleListRepos))
		repos.Post("/", api(s.handleCreateRepo))
		repos.Route("/{owner}/{repo}", s.repoRoutes)
	})
	router.Route("/tokens", func(tokens chi.Router) {
		tokens.Get("/", api(s.handleListTokens))
		tokens.Post("/", api(s.handleCreateToken))
		tokens.Delete("/{id}", api(s.handleDeleteToken))
	})
	router.Route("/keys", func(keys chi.Router) {
		keys.Get("/", api(s.handleListKeys))
		keys.Post("/", api(s.handleCreateKey))
		keys.Delete("/{id}", api(s.handleDeleteKey))
	})
	router.Get("/events", api(s.handleListEvents))
	router.Get("/events/stream", api(s.handleEventStream))
	router.Get("/signing-key", api(s.handleSigningKey))
	runnerPath, runnerHandler := runnerv1connect.NewRunnerServiceHandler(runnerService{server: s})
	pingPath, pingHandler := pingv1connect.NewPingServiceHandler(pingService{})
	router.Handle("/actions"+runnerPath+"*", http.StripPrefix("/api/actions", runnerHandler))
	router.Handle("/actions"+pingPath+"*", http.StripPrefix("/api/actions", pingHandler))
	router.Route("/admin", func(admin chi.Router) {
		admin.Post("/backup", api(s.handleBackup))
		admin.Post("/readonly", api(s.handleReadOnly))
		admin.Get("/status", api(s.handleStatus))
		admin.Get("/storage-check", api(s.handleStorageCheck))
		admin.Post("/registry-gc", api(s.handleRegistryGC))
	})
	router.Route("/owners", func(owners chi.Router) {
		owners.Get("/", api(s.handleListOwners))
		owners.Post("/", api(s.handleCreateOwner))
		owners.Get("/{owner}", api(s.handleGetOwner))
		owners.Delete("/{owner}", api(s.handleDeleteOwner))
	})
	router.Route("/runners", func(runners chi.Router) {
		runners.Get("/", api(s.handleListRunners))
		runners.Post("/", api(s.handleCreateRunner))
		runners.Delete("/{id}", api(s.handleDeleteRunner))
		runners.Get("/tokens", api(s.handleListRunnerTokens))
		runners.Post("/tokens", api(s.handleCreateRunnerToken))
		runners.Delete("/tokens/{id}", api(s.handleDeleteRunnerToken))
	})
}

func (s *Server) repoRoutes(repo chi.Router) {
	repo.Get("/", api(s.handleGetRepo))
	repo.Get("/tree", api(s.handleTree))
	repo.Get("/blob", api(s.handleBlob))
	repo.Get("/files", api(s.handleFiles))
	repo.Delete("/", api(s.handleDeleteRepo))
	repo.Get("/commits", api(s.handleListCommits))
	repo.Get("/commits/{rev}", api(s.handleGetCommit))
	repo.Get("/commits/{rev}/view", api(s.handleCommitWorkspace))
	repo.Get("/tags", api(s.handleListTags))
	repo.Route("/issues", func(issues chi.Router) {
		issues.Get("/", api(s.handleListIssues))
		issues.Post("/", api(s.handleCreateIssue))
		issues.Get("/{number}", api(s.handleGetIssue))
		issues.Get("/{number}/view", api(s.handleItemWorkspace(kindIssue)))
		issues.Patch("/{number}", api(s.handleEditItem(kindIssue)))
		issues.Post("/{number}/comments", api(s.handleCommentItem(kindIssue)))
	})
	repo.Route("/pulls", func(pulls chi.Router) {
		pulls.Get("/", api(s.handleListPulls))
		pulls.Post("/", api(s.handleCreatePull))
		pulls.Get("/{number}", api(s.handleGetPull))
		pulls.Get("/{number}/view", api(s.handleItemWorkspace(kindPull)))
		pulls.Get("/{number}/patch", api(s.handlePullPatch))
		pulls.Patch("/{number}", api(s.handleEditItem(kindPull)))
		pulls.Post("/{number}/comments", api(s.handleCommentItem(kindPull)))
		pulls.Get("/{number}/commits", api(s.handlePullCommits))
		pulls.Get("/{number}/diff", api(s.handlePullDiff))
		pulls.Get("/{number}/files", api(s.handlePullFiles))
		pulls.Post("/{number}/merge", api(s.handleMergePull))
		pulls.Post("/{number}/reviews", api(s.handleReviewPull))
	})
	repo.Get("/images", api(s.handleListImages))
	repo.Post("/workflows/dispatch", api(s.handleDispatch))
	repo.Get("/workflows", api(s.handleWorkflows))
	repo.Route("/secrets", func(secrets chi.Router) {
		secrets.Get("/", api(s.handleListSecrets))
		secrets.Put("/{name}", api(s.handleSetSecret))
		secrets.Delete("/{name}", api(s.handleDeleteSecret))
	})
	repo.Route("/variables", func(variables chi.Router) {
		variables.Get("/", api(s.handleListVariables))
		variables.Put("/{name}", api(s.handleSetVariable))
		variables.Delete("/{name}", api(s.handleDeleteVariable))
	})
	repo.Route("/runs", func(runs chi.Router) {
		runs.Get("/", api(s.handleListRuns))
		runs.Get("/{number}", api(s.handleGetRun))
		runs.Get("/{number}/jobs/{job}/logs", api(s.handleRunLogs))
		runs.Post("/{number}/cancel", api(s.handleCancelRun))
		runs.Post("/{number}/rerun", api(s.handleRerunRun))
	})
	repo.Route("/releases", func(releases chi.Router) {
		releases.Get("/", api(s.handleListReleases))
		releases.Post("/", api(s.handleCreateRelease))
		releases.Get("/latest", api(s.handleLatestRelease))
		releases.Get("/notes", api(s.handleReleaseNotes))
		releases.Get("/tags/*", api(s.handleReleaseByTag))
		releases.Get("/{id}", api(s.handleGetRelease))
		releases.Patch("/{id}", api(s.handleEditRelease))
		releases.Delete("/{id}", api(s.handleDeleteRelease))
		releases.Post("/{id}/assets", api(s.handleUploadAsset))
		releases.Get("/{id}/assets/{name}", api(s.handleDownloadAsset))
		releases.Delete("/{id}/assets/{name}", api(s.handleDeleteAsset))
	})
	repo.Route("/rules", func(rules chi.Router) {
		rules.Get("/", api(s.handleListRules))
		rules.Post("/", api(s.handleSetRule))
		rules.Delete("/", api(s.handleDeleteRule))
	})
	repo.Route("/labels", func(labels chi.Router) {
		labels.Get("/", api(s.handleListLabels))
		labels.Post("/", api(s.handleCreateLabel))
		labels.Delete("/{label}", api(s.handleDeleteLabel))
	})
}

func (s *Server) uiRoutes(repo chi.Router) {
	repo.Get("/info/refs", s.handleGit)
	repo.Post("/git-upload-pack", s.handleGit)
	repo.Post("/git-receive-pack", s.handleGit)

	repo.Get("/", s.page(s.uiApplication))
	repo.Get("/tree/{ref}", s.page(s.uiSource))
	repo.Get("/tree/{ref}/*", s.page(s.uiSource))
	repo.Get("/blob/{ref}/*", s.page(s.uiSource))
	repo.Get("/commits/{ref}", s.page(s.uiApplication))
	repo.Get("/commit/{sha}", s.page(s.uiApplication))
	repo.Get("/tags", s.page(s.uiApplication))
	repo.Get("/releases", s.page(s.uiApplication))
	repo.Get("/runs", s.page(s.uiApplication))
	repo.Get("/runs/{number}", s.page(s.uiApplication))
	repo.Post("/runs/{number}/{action}", s.page(s.uiRunAction))
	repo.Get("/images", s.page(s.uiApplication))
	repo.Get("/releases/tag/*", s.page(s.uiApplication))
	repo.Get("/activity", s.page(s.uiApplication))
	repo.Get("/settings", s.page(s.uiApplication))
	repo.Get("/releases/download/*", api(s.handleStableDownload))
	repo.Get("/releases/latest/download/{name}", api(s.handleLatestDownload))
	repo.Post("/issues", s.page(s.uiCreateIssue))
	repo.Post("/pulls/{number}/merge", s.page(s.uiMergePull))
	repo.Post("/pulls/{number}/reviews", s.page(s.uiReviewPull))
	for kind, path := range itemPaths {
		repo.Get("/"+path, s.page(s.uiApplication))
		repo.Get("/"+path+"/{number}", s.page(s.uiApplication))
		repo.Post("/"+path+"/{number}/comments", s.page(s.uiCommentItem(kind)))
		repo.Post("/"+path+"/{number}/edit", s.page(s.uiEditItem(kind)))
	}
}
