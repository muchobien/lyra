package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/muchobien/lyra/internal/store"
)

const (
	releaseUpdated   = "release.updated"
	releasePublished = "release.published"
)

type createReleaseRequest struct {
	Tag           string `json:"tag"`
	Target        string `json:"target"`
	Title         string `json:"title"`
	Notes         string `json:"notes"`
	Draft         bool   `json:"draft"`
	Prerelease    bool   `json:"prerelease"`
	GenerateNotes bool   `json:"generate_notes"`
}

type editReleaseRequest struct {
	Title      *string `json:"title"`
	Notes      *string `json:"notes"`
	Draft      *bool   `json:"draft"`
	Prerelease *bool   `json:"prerelease"`
	Target     *string `json:"target"`
}

type releaseWrite struct {
	store.Release
	EventID int64    `json:"event_id"`
	Actions []string `json:"actions,omitempty"`
}

type releaseDeleted struct {
	Deleted    string `json:"deleted"`
	EventID    int64  `json:"event_id"`
	TagDeleted bool   `json:"tag_deleted"`
}

type releasesView struct {
	Releases []store.Release
	Latest   string
	Single   bool
}

func releasesPath(repo string) string {
	return "/lyra/releases/" + repo
}

func tagRef(tag string) string {
	return "refs/tags/" + tag
}

func pathParam(r *http.Request, key string) string {
	value := chi.URLParam(r, key)
	if r.URL.RawPath == "" {
		return value
	}
	unescaped, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	return unescaped
}

func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func downloadURL(r *http.Request, repo, tag, name string) string {
	return baseURL(r) + "/" + repo + "/releases/download/" + escapePath(tag) + "/" + url.PathEscape(name)
}

func (s *Server) releaseLinks(r *http.Request, release store.Release) store.Release {
	for index := range release.Assets {
		release.Assets[index].DownloadURL = downloadURL(r, release.Repo, release.Tag, release.Assets[index].Name)
	}
	images, err := s.store.Images(r.Context(), release.Repo)
	if err != nil {
		slog.Error("images for release", "repo", release.Repo, "tag", release.Tag, "error", err)
	}
	release.Images = []store.Image{}
	for _, image := range images {
		if image.Tag == release.Tag || "v"+image.Tag == release.Tag {
			image.Reference = s.imageReference(r, image)
			release.Images = append(release.Images, image)
		}
	}
	return release
}

func releaseError(err error, tag string) error {
	if errors.Is(err, store.ErrExists) {
		return conflict("a release for tag " + tag + " already exists")
	}
	return missing(err, "release not found")
}

func (s *Server) visibleRelease(r *http.Request, repo string, release store.Release, err error) (store.Release, error) {
	if err == nil && release.Draft && !identityFrom(r).can(scopeWrite, repo) {
		return release, store.ErrNotFound
	}
	return release, err
}

func (s *Server) lookupRelease(r *http.Request, scope string) (store.Repo, store.Release, error) {
	repo, err := s.lookupRepo(r, scope)
	if err != nil {
		return repo, store.Release{}, err
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return repo, store.Release{}, notFound("release not found")
	}
	release, err := s.store.Release(r.Context(), repo.Name, id)
	release, err = s.visibleRelease(r, repo.Name, release, err)
	return repo, release, missing(err, "release not found")
}

func (s *Server) releaseByTag(r *http.Request, repo, tag string) (store.Release, error) {
	release, err := s.store.ReleaseByTag(r.Context(), repo, tag)
	return s.visibleRelease(r, repo, release, err)
}

func (s *Server) tagRelease(
	ctx context.Context,
	dir string,
	release *store.Release,
	explicitTarget, publish bool,
) (string, error) {
	ref := tagRef(release.Tag)
	absent := store.InvalidError("target " + release.Target + " not found")
	targetSHA, targetErr := s.git.ResolveCommit(ctx, dir, release.Target)
	if tagSHA, err := s.git.ResolveCommit(ctx, dir, ref); err == nil {
		if explicitTarget && targetErr != nil {
			return "", absent
		}
		if explicitTarget && targetSHA != tagSHA {
			return "", conflict("tag " + release.Tag + " already points at " + tagSHA + ", not " + release.Target)
		}
		release.Commit = tagSHA
		return "", nil
	}
	if targetErr != nil {
		return "", absent
	}
	release.Commit = ""
	if !publish {
		return "", nil
	}
	if err := s.git.UpdateRef(ctx, dir, ref, targetSHA, strings.Repeat("0", len(targetSHA))); err != nil {
		return "", conflict("tag " + release.Tag + " was created while publishing; try again")
	}
	release.Commit = targetSHA
	return targetSHA, nil
}

func (s *Server) untag(ctx context.Context, dir, tag, sha string) {
	if sha == "" {
		return
	}
	if err := s.git.UpdateRef(ctx, dir, tagRef(tag), "", sha); err != nil {
		slog.Error("remove tag after a failed release", "tag", tag, "error", err)
	}
}

func (s *Server) handleListReleases(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	releases, err := s.store.Releases(r.Context(), repo.Name)
	if err != nil {
		return err
	}
	drafts := identityFrom(r).can(scopeWrite, repo.Name)
	visible := []store.Release{}
	for _, release := range releases {
		if drafts || !release.Draft {
			visible = append(visible, s.releaseLinks(r, release))
		}
	}
	return writeJSON(w, http.StatusOK, visible)
}

func (s *Server) handleLatestRelease(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	release, err := s.store.LatestRelease(r.Context(), repo.Name)
	if err != nil {
		return missing(err, "no published release")
	}
	return writeJSON(w, http.StatusOK, s.releaseLinks(r, release))
}

func (s *Server) handleReleaseByTag(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	tag := pathParam(r, "*")
	release, err := s.releaseByTag(r, repo.Name, tag)
	if err != nil {
		return missing(err, "no release for tag "+tag)
	}
	return writeJSON(w, http.StatusOK, s.releaseLinks(r, release))
}

func (s *Server) handleGetRelease(w http.ResponseWriter, r *http.Request) error {
	_, release, err := s.lookupRelease(r, scopeRead)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, s.releaseLinks(r, release))
}

func (s *Server) appendGeneratedNotes(ctx context.Context, dir string, release *store.Release) error {
	commit, err := s.git.ResolveCommit(ctx, dir, tagRef(release.Tag))
	if err != nil {
		commit, err = s.git.ResolveCommit(ctx, dir, release.Target)
	}
	if err != nil {
		return badRequest("target " + release.Target + " not found")
	}
	generated, err := s.generateNotes(ctx, release.Repo, release.Tag, commit, release.Prerelease)
	if err != nil {
		return errors.New("release notes: " + err.Error())
	}
	release.Notes = strings.TrimSpace(release.Notes + "\n\n" + generated.Notes)
	return checkText(nil, &release.Notes)
}

func (s *Server) newRelease(ctx context.Context, repo string, body createReleaseRequest) (store.Release, error) {
	release := store.Release{
		Repo:       repo,
		Tag:        body.Tag,
		Title:      body.Title,
		Notes:      body.Notes,
		Draft:      body.Draft,
		Prerelease: body.Prerelease,
		Target:     body.Target,
	}
	if !s.git.ValidRef(ctx, tagRef(body.Tag)) {
		return release, badRequest("invalid tag name " + strconv.Quote(body.Tag))
	}
	if release.Title == "" {
		release.Title = body.Tag
	}
	if err := checkText(&release.Title, &release.Notes); err != nil {
		return release, err
	}
	if _, err := s.store.ReleaseByTag(ctx, repo, body.Tag); err == nil {
		return release, releaseError(store.ErrExists, body.Tag)
	}
	dir := s.repoDir(repo)
	if release.Target == "" {
		release.Target = s.git.DefaultBranch(ctx, dir)
	}
	if body.GenerateNotes {
		if err := s.appendGeneratedNotes(ctx, dir, &release); err != nil {
			return release, err
		}
	}
	return release, nil
}

func (s *Server) handleCreateRelease(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeWrite)
	if err != nil {
		return err
	}
	body, err := decodeBody[createReleaseRequest](w, r)
	if err != nil {
		return err
	}
	release, err := s.newRelease(ctx, repo.Name, body)
	if err != nil {
		return err
	}
	dir := s.repoDir(repo.Name)
	createdTag, err := s.tagRelease(ctx, dir, &release, body.Target != "", !body.Draft)
	if err != nil {
		return releaseError(err, body.Tag)
	}
	if !release.Draft {
		published := time.Now().UTC()
		release.PublishedAt = &published
	}
	created, eventID, err := s.store.CreateRelease(ctx, identityFrom(r).actor, release)
	if err != nil {
		s.untag(ctx, dir, body.Tag, createdTag)
		return releaseError(err, body.Tag)
	}
	return writeJSON(w, http.StatusCreated, releaseWrite{
		Release: s.releaseLinks(r, created),
		EventID: eventID,
		Actions: s.queueRelease(r, created),
	})
}

func applyReleaseEdit(release *store.Release, body editReleaseRequest) error {
	if !release.Draft && body.Draft != nil && *body.Draft {
		return badRequest("a published release can't go back to draft")
	}
	if !release.Draft && body.Target != nil {
		return badRequest("the target of a published release can't change")
	}
	if err := checkText(body.Title, body.Notes); err != nil {
		return err
	}
	if body.Title != nil {
		release.Title = *body.Title
	}
	if body.Notes != nil {
		release.Notes = *body.Notes
	}
	if body.Prerelease != nil {
		release.Prerelease = *body.Prerelease
	}
	if body.Target != nil {
		release.Target = *body.Target
	}
	return nil
}

func (s *Server) handleEditRelease(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, release, err := s.lookupRelease(r, scopeWrite)
	if err != nil {
		return err
	}
	body, err := decodeBody[editReleaseRequest](w, r)
	if err != nil {
		return err
	}
	if err := applyReleaseEdit(&release, body); err != nil {
		return err
	}
	publish := release.Draft && body.Draft != nil && !*body.Draft
	dir := s.repoDir(repo.Name)
	createdTag := ""
	if release.Draft {
		if createdTag, err = s.tagRelease(ctx, dir, &release, body.Target != nil, publish); err != nil {
			return releaseError(err, release.Tag)
		}
	}
	action := releaseUpdated
	if publish {
		action = releasePublished
		published := time.Now().UTC()
		release.Draft = false
		release.PublishedAt = &published
	}
	updated, eventID, err := s.store.UpdateRelease(ctx, identityFrom(r).actor, release, action)
	if err != nil {
		s.untag(ctx, dir, release.Tag, createdTag)
		return releaseError(err, release.Tag)
	}
	actions := []string{}
	if publish {
		actions = s.queueRelease(r, updated)
	}
	return writeJSON(w, http.StatusOK, releaseWrite{
		Release: s.releaseLinks(r, updated),
		EventID: eventID,
		Actions: actions,
	})
}

func (s *Server) deleteTag(ctx context.Context, repo, tag string) (bool, error) {
	dir := s.repoDir(repo)
	sha, err := s.git.ResolveCommit(ctx, dir, tagRef(tag))
	if err != nil {
		return false, nil
	}
	if err := s.git.UpdateRef(ctx, dir, tagRef(tag), "", ""); err != nil {
		return false, err
	}
	slog.Info("deleted release tag", "repo", repo, "tag", tag, "sha", sha)
	return true, nil
}

func (s *Server) handleDeleteRelease(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, release, err := s.lookupRelease(r, scopeWrite)
	if err != nil {
		return err
	}
	eventID, err := s.store.DeleteRelease(ctx, identityFrom(r).actor, release)
	if err != nil {
		return releaseError(err, release.Tag)
	}
	for _, asset := range release.Assets {
		s.deleteObject(ctx, repo.Name, asset)
	}
	tagDeleted := false
	if r.URL.Query().Get("cleanup_tag") == "true" {
		if tagDeleted, err = s.deleteTag(ctx, repo.Name, release.Tag); err != nil {
			return err
		}
	}
	return writeJSON(w, http.StatusOK, releaseDeleted{Deleted: release.Tag, EventID: eventID, TagDeleted: tagDeleted})
}

func (s *Server) shownReleases(r *http.Request, repo, tag string) ([]store.Release, error) {
	if tag == "" {
		return s.store.Releases(r.Context(), repo)
	}
	release, err := s.releaseByTag(r, repo, tag)
	return []store.Release{release}, missing(err, "No release for tag "+tag+".")
}

func (s *Server) uiReleases(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.uiLookup(r)
	if err != nil {
		return err
	}
	tag := pathParam(r, "*")
	releases, err := s.shownReleases(r, repo.Name, tag)
	if err != nil {
		return err
	}
	drafts := identityFrom(r).can(scopeWrite, repo.Name)
	view := releasesView{Releases: []store.Release{}, Single: tag != ""}
	if latest, err := s.store.LatestRelease(ctx, repo.Name); err == nil {
		view.Latest = latest.Tag
	}
	for _, release := range releases {
		if drafts || !release.Draft {
			view.Releases = append(view.Releases, s.releaseLinks(r, release))
		}
	}
	title := "Releases · " + repo.Name
	if tag != "" {
		title = releases[0].Title + " · " + repo.Name
	}
	ref := s.git.DefaultBranch(ctx, s.repoDir(repo.Name))
	s.render(w, r, http.StatusOK, "releases", s.repoPage(r, &repo, title, ref, view))
	return nil
}
