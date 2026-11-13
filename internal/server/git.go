package server

import (
	"io"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

const receivePack = "git-receive-pack"

func (s *Server) gitBackend(r *http.Request, name string) (*cgi.Handler, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, err
	}
	actor := identityFrom(r).actor
	return &cgi.Handler{
		Path:       gitPath,
		Args:       []string{"http-backend"},
		InheritEnv: []string{"PATH", "TMPDIR"},
		Env: []string{
			"GIT_PROJECT_ROOT=" + filepath.Join(s.config.DataDir, "repos"),
			"GIT_HTTP_EXPORT_ALL=1",
			"GIT_CONFIG_NOSYSTEM=1",
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_COUNT=1",
			"GIT_CONFIG_KEY_0=core.hooksPath",
			"GIT_CONFIG_VALUE_0=" + filepath.Join(s.config.DataDir, "hooks"),
			"GIT_PROTOCOL=" + r.Header.Get("Git-Protocol"),
			"REMOTE_USER=" + actor.User,
			"LYRA_DB=" + s.dbPath,
			"LYRA_REPO=" + name,
			"LYRA_USER=" + actor.User,
			"LYRA_VIA=" + actor.Via,
			"LYRA_AGENT=" + actor.Agent,
			"LYRA_SESSION=" + actor.Session,
		},
	}, nil
}

func (s *Server) spoolBody(request *http.Request) (func(), error) {
	spool, err := os.CreateTemp(filepath.Join(s.config.DataDir, "tmp"), "request-*")
	if err != nil {
		return nil, err
	}
	cleanup := func() {
		spool.Close()
		os.Remove(spool.Name())
	}
	size, err := io.Copy(spool, request.Body)
	if err == nil {
		_, err = spool.Seek(0, io.SeekStart)
	}
	if err != nil {
		cleanup()
		return nil, badRequest(err.Error())
	}
	request.Body = spool
	request.ContentLength = size
	request.TransferEncoding = nil
	return cleanup, nil
}

func (s *Server) serveGit(w http.ResponseWriter, r *http.Request) error {
	id := identityFrom(r)
	name := r.PathValue("owner") + "/" + strings.TrimSuffix(r.PathValue("repo"), ".git")
	if _, err := s.store.Repo(r.Context(), name); err != nil {
		return missing(err, "repository not found")
	}
	scope := scopeRead
	if strings.HasSuffix(r.URL.Path, "/"+receivePack) || r.URL.Query().Get("service") == receivePack {
		scope = scopeWrite
	}
	if !id.can(scope, name) {
		return forbidden("token lacks " + scope + " access to " + name)
	}
	backend, err := s.gitBackend(r, name)
	if err != nil {
		return err
	}
	rest := strings.TrimPrefix(r.URL.Path, "/"+r.PathValue("owner")+"/"+r.PathValue("repo"))
	request := r.Clone(r.Context())
	request.URL.Path = "/" + name + ".git" + rest
	if slices.Contains(r.TransferEncoding, "chunked") {
		cleanup, err := s.spoolBody(request)
		if err != nil {
			return err
		}
		defer cleanup()
	}
	backend.ServeHTTP(w, request)
	if scope == scopeWrite {
		s.store.Notify()
	}
	return nil
}

func (s *Server) handleGit(w http.ResponseWriter, r *http.Request) {
	if err := s.serveGit(w, r); err != nil {
		status, message := errorStatus(err)
		http.Error(w, message, status)
	}
}
