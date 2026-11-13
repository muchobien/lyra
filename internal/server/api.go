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
	Name        string `json:"name" doc:"owner/name"`
	Description string `json:"description,omitzero"`
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

func (s *Server) handleMe(r *http.Request, _ *struct{}) (*body[meView], error) {
	id := identityFrom(r)
	return reply(meView{Actor: id.actor, Repos: id.repos, Scope: id.scope})
}

func (s *Server) handleListRepos(r *http.Request, _ *struct{}) (*body[[]repoView], error) {
	id := identityFrom(r)
	all, err := s.store.Repos(r.Context())
	if err != nil {
		return nil, err
	}
	views := []repoView{}
	for _, repo := range all {
		if !id.can(scopeRead, repo.Name) {
			continue
		}
		view, err := s.repoView(r, repo)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return reply(views)
}

func validRepoName(full string) bool {
	owner, name, found := strings.Cut(full, "/")
	return found && repoNamePattern.MatchString(owner) && repoNamePattern.MatchString(name) &&
		!strings.HasSuffix(name, ".git")
}

func (s *Server) handleCreateRepo(r *http.Request, input *bodyInput[createRepoRequest]) (*body[repoView], error) {
	ctx := r.Context()
	id := identityFrom(r)
	request := input.Body
	if !validRepoName(request.Name) {
		return nil, badRequest(repoNameProblem)
	}
	owner, _, _ := strings.Cut(request.Name, "/")
	if _, err := s.store.Owner(ctx, owner); err != nil {
		return nil, missing(err, "owner "+owner+" doesn't exist; create it with lyra owner create "+owner)
	}
	if !id.can(scopeAdmin, request.Name) {
		return nil, forbidden("token lacks admin access to " + request.Name)
	}
	repo, err := s.store.CreateRepo(ctx, request.Name, request.Description, id.actor.User)
	if store.IsUniqueViolation(err) {
		return nil, conflict("repository already exists")
	}
	if err != nil {
		return nil, err
	}
	if err := s.git.Init(ctx, s.repoDir(repo.Name)); err != nil {
		if cleanup := s.store.DeleteRepo(ctx, repo.Name); cleanup != nil {
			slog.Error("remove repository after a failed init", "repo", repo.Name, "error", cleanup)
		}
		return nil, err
	}
	created := repoCreated{Description: repo.Description}
	if _, err := s.store.AddEvent(ctx, "repo.create", repo.Name, id.actor, created); err != nil {
		return nil, err
	}
	view, err := s.repoView(r, repo)
	if err != nil {
		return nil, err
	}
	return reply(view)
}

func (s *Server) handleGetRepo(r *http.Request, _ *RepoPath) (*body[repoView], error) {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return nil, err
	}
	view, err := s.repoView(r, repo)
	if err != nil {
		return nil, err
	}
	return reply(view)
}

func (s *Server) handleDeleteRepo(r *http.Request, _ *RepoPath) (*body[deletedName], error) {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeAdmin)
	if err != nil {
		return nil, err
	}
	if err := s.store.DeleteRepo(ctx, repo.Name); err != nil {
		return nil, err
	}
	if err := os.RemoveAll(s.repoDir(repo.Name)); err != nil {
		return nil, err
	}
	for _, prefix := range []string{releasesPath(repo.Name), registryRepositories + repo.Name} {
		err := s.storage.Delete(ctx, prefix)
		if _, absent := errors.AsType[driver.PathNotFoundError](err); err != nil && !absent {
			return nil, err
		}
	}
	if _, err := s.store.AddEvent(ctx, "repo.delete", repo.Name, identityFrom(r).actor, noData{}); err != nil {
		return nil, err
	}
	return reply(deletedName{Deleted: repo.Name})
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

type listCommitsInput struct {
	RepoPath
	Ref   string `query:"ref" doc:"Defaults to the default branch."`
	Limit int    `query:"limit" doc:"1 to 500; defaults to 30."`
}

func (s *Server) handleListCommits(r *http.Request, input *listCommitsInput) (*body[[]repos.Commit], error) {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return nil, err
	}
	ref := input.Ref
	if ref == "" {
		ref = s.git.DefaultBranch(ctx, s.repoDir(repo.Name))
	}
	commits, err := s.commits(ctx, repo.Name, ref, queryLimit(input.Limit, 30, 500))
	if errors.Is(err, repos.ErrNotFound) {
		return nil, notFound("unknown ref " + ref)
	}
	if err != nil {
		return nil, err
	}
	return reply(commits)
}

type commitInput struct {
	RepoPath
	Rev string `path:"rev"`
}

func (s *Server) handleGetCommit(r *http.Request, input *commitInput) (*body[repos.Commit], error) {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return nil, err
	}
	commit, err := s.git.Commit(ctx, s.repoDir(repo.Name), input.Rev)
	if errors.Is(err, repos.ErrNotFound) {
		return nil, notFound("unknown revision")
	}
	if err != nil {
		return nil, err
	}
	if err := s.verifySignatures(ctx, &commit.Signature); err != nil {
		return nil, err
	}
	return reply(commit)
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

func (s *Server) handleListTags(r *http.Request, _ *RepoPath) (*body[[]repos.Tag], error) {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return nil, err
	}
	tags, err := s.tags(r.Context(), repo.Name)
	if err != nil {
		return nil, err
	}
	return reply(tags)
}
