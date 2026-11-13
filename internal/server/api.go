package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/distribution/distribution/v3/registry/storage/driver"
	"github.com/hiddeco/sshsig"
	"golang.org/x/crypto/ssh"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

const repoNameProblem = "repository names are owner/name; each part is 1 to 100 letters, digits, ., _ or -, " +
	"starting with a letter or digit, and the name can't end in .git"

type repoView struct {
	store.Repo
	DefaultBranch string      `json:"default_branch"`
	Branches      []repos.Ref `json:"branches"`
	CloneURL      string      `json:"clone_url"`
}

type meView struct {
	Actor store.Actor `json:"actor"`
	Repos []string    `json:"repos"`
	Scope string      `json:"scope"`
}

type createRepoRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type repoCreated struct {
	Description string `json:"description"`
}

type deletedName struct {
	Deleted string `json:"deleted"`
}

type noData struct{}

func cloneURL(r *http.Request, name string) string {
	return baseURL(r) + "/" + name + ".git"
}

func (s *Server) repoView(r *http.Request, repo store.Repo) (repoView, error) {
	dir := s.repoDir(repo.Name)
	branches, err := s.git.Branches(r.Context(), dir)
	if err != nil {
		return repoView{}, err
	}
	return repoView{
		Repo:          repo,
		DefaultBranch: s.git.DefaultBranch(r.Context(), dir),
		Branches:      branches,
		CloneURL:      cloneURL(r, repo.Name),
	}, nil
}

func (s *Server) lookupRepo(r *http.Request, scope string) (store.Repo, error) {
	name := r.PathValue("owner") + "/" + r.PathValue("repo")
	if !identityFrom(r).can(scope, name) {
		return store.Repo{}, forbidden("token lacks " + scope + " access to " + name)
	}
	repo, err := s.store.Repo(r.Context(), name)
	return repo, missing(err, "repository not found")
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) error {
	id := identityFrom(r)
	return writeJSON(w, http.StatusOK, meView{Actor: id.actor, Repos: id.repos, Scope: id.scope})
}

func (s *Server) handleListRepos(w http.ResponseWriter, r *http.Request) error {
	id := identityFrom(r)
	all, err := s.store.Repos(r.Context())
	if err != nil {
		return err
	}
	views := []repoView{}
	for _, repo := range all {
		if !id.can(scopeRead, repo.Name) {
			continue
		}
		view, err := s.repoView(r, repo)
		if err != nil {
			return err
		}
		views = append(views, view)
	}
	return writeJSON(w, http.StatusOK, views)
}

func validRepoName(full string) bool {
	owner, name, found := strings.Cut(full, "/")
	return found && repoNamePattern.MatchString(owner) && repoNamePattern.MatchString(name) &&
		!strings.HasSuffix(name, ".git")
}

func (s *Server) handleCreateRepo(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id := identityFrom(r)
	body, err := decodeBody[createRepoRequest](w, r)
	if err != nil {
		return err
	}
	if !validRepoName(body.Name) {
		return badRequest(repoNameProblem)
	}
	owner, _, _ := strings.Cut(body.Name, "/")
	if _, err := s.store.Owner(ctx, owner); err != nil {
		return missing(err, "owner "+owner+" doesn't exist; create it with lyra owner create "+owner)
	}
	if !id.can(scopeAdmin, body.Name) {
		return forbidden("token lacks admin access to " + body.Name)
	}
	repo, err := s.store.CreateRepo(ctx, body.Name, body.Description, id.actor.User)
	if store.IsUniqueViolation(err) {
		return conflict("repository already exists")
	}
	if err != nil {
		return err
	}
	if err := s.git.Init(ctx, s.repoDir(repo.Name)); err != nil {
		if cleanup := s.store.DeleteRepo(ctx, repo.Name); cleanup != nil {
			slog.Error("remove repository after a failed init", "repo", repo.Name, "error", cleanup)
		}
		return err
	}
	created := repoCreated{Description: repo.Description}
	if _, err := s.store.AddEvent(ctx, "repo.create", repo.Name, id.actor, created); err != nil {
		return err
	}
	view, err := s.repoView(r, repo)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusCreated, view)
}

func (s *Server) handleGetRepo(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	view, err := s.repoView(r, repo)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleDeleteRepo(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeAdmin)
	if err != nil {
		return err
	}
	if err := s.store.DeleteRepo(ctx, repo.Name); err != nil {
		return err
	}
	if err := os.RemoveAll(s.repoDir(repo.Name)); err != nil {
		return err
	}
	for _, prefix := range []string{releasesPath(repo.Name), registryRepositories + repo.Name} {
		err := s.storage.Delete(ctx, prefix)
		if _, absent := errors.AsType[driver.PathNotFoundError](err); err != nil && !absent {
			return err
		}
	}
	if _, err := s.store.AddEvent(ctx, "repo.delete", repo.Name, identityFrom(r).actor, noData{}); err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, deletedName{Deleted: repo.Name})
}

func (s *Server) verifySignatures(ctx context.Context, signatures ...*repos.Signature) error {
	keys, err := s.store.Keys(ctx, "")
	if err != nil {
		return err
	}
	return verifyWithKeys(keys, signatures...)
}

func verifyWithKey(key store.Key, parsed *sshsig.Signature, signature *repos.Signature) error {
	publicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key.PublicKey))
	if err != nil {
		return err
	}
	payload := bytes.NewReader(signature.Payload)
	if sshsig.Verify(payload, parsed, publicKey, parsed.HashAlgorithm, "git") == nil {
		signature.Status = repos.Verified
		signature.KeyID = key.ID
		signature.KeyUser = key.User
		signature.KeyAgent = key.Agent
	}
	return nil
}

func verifyWithKeys(keys []store.Key, signatures ...*repos.Signature) error {
	for _, signature := range signatures {
		if signature.Armored == nil {
			continue
		}
		parsed, err := sshsig.Unarmor(signature.Armored)
		if err != nil {
			continue
		}
		signature.Fingerprint = ssh.FingerprintSHA256(parsed.PublicKey)
		for _, key := range keys {
			if !key.Signing || key.Fingerprint != signature.Fingerprint {
				continue
			}
			if err := verifyWithKey(key, parsed, signature); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Server) commits(ctx context.Context, repo, ref string, limit int, exclude ...string) ([]repos.Commit, error) {
	commits, err := s.git.Log(ctx, s.repoDir(repo), ref, limit, exclude...)
	if err != nil {
		return nil, err
	}
	signatures := []*repos.Signature{}
	for index := range commits {
		signatures = append(signatures, &commits[index].Signature)
	}
	return commits, s.verifySignatures(ctx, signatures...)
}

func (s *Server) handleListCommits(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		ref = s.git.DefaultBranch(ctx, s.repoDir(repo.Name))
	}
	commits, err := s.commits(ctx, repo.Name, ref, queryLimit(r, 30, 500))
	if errors.Is(err, repos.ErrNotFound) {
		return notFound("unknown ref " + ref)
	}
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, commits)
}

func (s *Server) handleGetCommit(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	commit, err := s.git.Commit(ctx, s.repoDir(repo.Name), r.PathValue("rev"))
	if errors.Is(err, repos.ErrNotFound) {
		return notFound("unknown revision")
	}
	if err != nil {
		return err
	}
	if err := s.verifySignatures(ctx, &commit.Signature); err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, commit)
}

func (s *Server) tags(ctx context.Context, repo string) ([]repos.Tag, error) {
	tags, err := s.git.Tags(ctx, s.repoDir(repo))
	if err != nil {
		return nil, err
	}
	signatures := []*repos.Signature{}
	for index := range tags {
		signatures = append(signatures, &tags[index].Signature)
	}
	return tags, s.verifySignatures(ctx, signatures...)
}

func (s *Server) handleListTags(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	tags, err := s.tags(r.Context(), repo.Name)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, tags)
}
