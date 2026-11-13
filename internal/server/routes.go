package server

import (
	"net/http"

	"code.forgejo.org/forgejo/actions-proto/ping/v1/pingv1connect"
	"code.forgejo.org/forgejo/actions-proto/runner/v1/runnerv1connect"
	"github.com/go-chi/chi/v5"
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
	s.registerAPI(newAPI(router))
	runnerPath, runnerHandler := runnerv1connect.NewRunnerServiceHandler(runnerService{server: s})
	pingPath, pingHandler := pingv1connect.NewPingServiceHandler(pingService{})
	router.Handle("/api/actions"+runnerPath+"*", http.StripPrefix("/api/actions", runnerHandler))
	router.Handle("/api/actions"+pingPath+"*", http.StripPrefix("/api/actions", pingHandler))
	router.Get("/api/repos/{owner}/{repo}/releases/tags/*", api(s.handleReleaseByTag))
	router.Handle("/api/*", api(func(http.ResponseWriter, *http.Request) error {
		return notFound("no such API endpoint")
	}))
	router.Get("/", s.page(s.uiRepoList))
	router.Get("/{owner}", s.page(s.uiOwner))
	router.Route("/{owner}/{repo}", s.uiRoutes)
	return router
}

func (s *Server) uiRoutes(repo chi.Router) {
	repo.Get("/info/refs", s.handleGit)
	repo.Post("/git-upload-pack", s.handleGit)
	repo.Post("/git-receive-pack", s.handleGit)

	repo.Get("/", s.page(s.uiRepo))
	repo.Get("/tree/{ref}", s.page(s.uiTree))
	repo.Get("/tree/{ref}/*", s.page(s.uiTree))
	repo.Get("/blob/{ref}/*", s.page(s.uiBlob))
	repo.Get("/commits/{ref}", s.page(s.uiCommits))
	repo.Get("/commit/{sha}", s.page(s.uiCommit))
	repo.Get("/tags", s.page(s.uiTags))
	repo.Get("/releases", s.page(s.uiReleases))
	repo.Get("/runs", s.page(s.uiRuns))
	repo.Get("/runs/{number}", s.page(s.uiRun))
	repo.Post("/runs/{number}/{action}", s.page(s.uiRunAction))
	repo.Get("/images", s.page(s.uiImages))
	repo.Get("/releases/tag/*", s.page(s.uiReleases))
	repo.Get("/releases/download/*", api(s.handleStableDownload))
	repo.Get("/releases/latest/download/{name}", api(s.handleLatestDownload))
	repo.Post("/issues", s.page(s.uiCreateIssue))
	repo.Post("/pulls/{number}/merge", s.page(s.uiMergePull))
	repo.Post("/pulls/{number}/reviews", s.page(s.uiReviewPull))
	for kind, path := range itemPaths {
		repo.Get("/"+path, s.page(s.uiItems(kind)))
		repo.Get("/"+path+"/{number}", s.page(s.uiItemPage(kind)))
		repo.Post("/"+path+"/{number}/comments", s.page(s.uiCommentItem(kind)))
		repo.Post("/"+path+"/{number}/edit", s.page(s.uiEditItem(kind)))
	}
}
