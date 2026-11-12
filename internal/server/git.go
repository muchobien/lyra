package server

import (
	"errors"
	"io"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/muchobien/lyra/internal/store"
)

func (s *Server) handleGit(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	name := strings.TrimSuffix(r.PathValue("repo"), ".git")
	if _, err := s.store.Repo(name); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "repository not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	scope := "read"
	if strings.HasSuffix(r.URL.Path, "/git-receive-pack") || r.URL.Query().Get("service") == "git-receive-pack" {
		scope = "write"
	}
	if !id.can(scope, name) {
		http.Error(w, "token lacks "+scope+" access to "+name, http.StatusForbidden)
		return
	}
	gitPath, err := exec.LookPath("git")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rest := r.URL.Path[strings.Index(r.URL.Path[1:], "/")+1:]
	request := r.Clone(r.Context())
	request.URL.Path = "/" + name + ".git" + rest
	handler := &cgi.Handler{
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
			"REMOTE_USER=" + id.actor.User,
			"LYRA_DB=" + s.dbPath,
			"LYRA_REPO=" + name,
			"LYRA_USER=" + id.actor.User,
			"LYRA_VIA=" + id.actor.Via,
			"LYRA_AGENT=" + id.actor.Agent,
			"LYRA_SESSION=" + id.actor.Session,
		},
	}
	if slices.Contains(r.TransferEncoding, "chunked") {
		spool, err := os.CreateTemp(filepath.Join(s.config.DataDir, "tmp"), "request-*")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer os.Remove(spool.Name())
		defer spool.Close()
		size, err := io.Copy(spool, r.Body)
		if err == nil {
			_, err = spool.Seek(0, io.SeekStart)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		request.Body = spool
		request.ContentLength = size
		request.TransferEncoding = nil
	}
	handler.ServeHTTP(w, request)
	if scope == "write" {
		s.store.Notify()
	}
}
