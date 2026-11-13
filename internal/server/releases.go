package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/distribution/distribution/v3/registry/storage/driver"
	"github.com/go-chi/chi/v5"

	"github.com/muchobien/lyra/internal/store"
)

func releasesPath(repo string) string {
	return "/lyra/releases/" + repo
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
	segments := strings.Split(tag, "/")
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	return baseURL(r) + "/" + repo + "/releases/download/" + strings.Join(segments, "/") + "/" + url.PathEscape(name)
}

func (s *Server) releaseLinks(r *http.Request, release store.Release) store.Release {
	for index := range release.Assets {
		release.Assets[index].DownloadURL = downloadURL(r, release.Repo, release.Tag, release.Assets[index].Name)
	}
	images, err := s.store.Images(release.Repo)
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

func checkAssetName(name string) string {
	if name == "" || len(name) > 255 || name == "." || name == ".." || strings.TrimSpace(name) != name ||
		strings.ContainsAny(name, `/\`) || strings.ContainsFunc(name, unicode.IsControl) {
		return "asset name must be 1 to 255 characters, with no slashes, control characters or surrounding spaces"
	}
	return ""
}

func writeReleaseError(w http.ResponseWriter, err error, tag string) {
	var conflict conflictError
	switch {
	case errors.As(err, &conflict):
		writeError(w, http.StatusConflict, conflict.Error())
	case errors.Is(err, store.ErrExists):
		writeError(w, http.StatusConflict, "a release for tag "+tag+" already exists")
	default:
		writeStoreError(w, err, "release not found")
	}
}

func (s *Server) visibleRelease(r *http.Request, repo string, release store.Release, err error) (store.Release, error) {
	if err == nil && release.Draft && !identityFrom(r).can("write", repo) {
		return release, store.ErrNotFound
	}
	return release, err
}

func (s *Server) lookupRelease(w http.ResponseWriter, r *http.Request, scope string) (store.Repo, store.Release, bool) {
	repo, ok := s.lookupRepo(w, r, scope)
	if !ok {
		return repo, store.Release{}, false
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "release not found")
		return repo, store.Release{}, false
	}
	release, err := s.store.Release(repo.Name, id)
	release, err = s.visibleRelease(r, repo.Name, release, err)
	if err != nil {
		writeStoreError(w, err, "release not found")
		return repo, release, false
	}
	return repo, release, true
}

func (s *Server) tagRelease(dir string, release *store.Release, explicitTarget, publish bool) (string, error) {
	ref := "refs/tags/" + release.Tag
	targetSHA, targetErr := s.git.ResolveCommit(dir, release.Target)
	if tagSHA, err := s.git.ResolveCommit(dir, ref); err == nil {
		if explicitTarget && targetErr == nil && targetSHA != tagSHA {
			return "", conflictError("tag " + release.Tag + " already points at " + tagSHA + ", not " + release.Target)
		}
		if explicitTarget && targetErr != nil {
			return "", store.InvalidError("target " + release.Target + " not found")
		}
		release.Commit = tagSHA
		return "", nil
	}
	if targetErr != nil {
		return "", store.InvalidError("target " + release.Target + " not found")
	}
	release.Commit = ""
	if !publish {
		return "", nil
	}
	if err := s.git.UpdateRef(dir, ref, targetSHA, strings.Repeat("0", len(targetSHA))); err != nil {
		return "", conflictError("tag " + release.Tag + " was created while publishing; try again")
	}
	release.Commit = targetSHA
	return targetSHA, nil
}

func (s *Server) untag(dir, tag, sha string) {
	if err := s.git.UpdateRef(dir, "refs/tags/"+tag, "", sha); err != nil {
		slog.Error("remove tag after a failed release", "tag", tag, "error", err)
	}
}

func (s *Server) handleListReleases(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	releases, err := s.store.Releases(repo.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	drafts := identityFrom(r).can("write", repo.Name)
	visible := []store.Release{}
	for _, release := range releases {
		if drafts || !release.Draft {
			visible = append(visible, s.releaseLinks(r, release))
		}
	}
	writeJSON(w, http.StatusOK, visible)
}

func (s *Server) handleLatestRelease(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	release, err := s.store.LatestRelease(repo.Name)
	if err != nil {
		writeStoreError(w, err, "no published release")
		return
	}
	writeJSON(w, http.StatusOK, s.releaseLinks(r, release))
}

func (s *Server) handleReleaseByTag(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	tag := pathParam(r, "*")
	release, err := s.store.ReleaseByTag(repo.Name, tag)
	release, err = s.visibleRelease(r, repo.Name, release, err)
	if err != nil {
		writeStoreError(w, err, "no release for tag "+tag)
		return
	}
	writeJSON(w, http.StatusOK, s.releaseLinks(r, release))
}

func (s *Server) handleGetRelease(w http.ResponseWriter, r *http.Request) {
	_, release, ok := s.lookupRelease(w, r, "read")
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.releaseLinks(r, release))
}

func (s *Server) handleCreateRelease(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "write")
	if !ok {
		return
	}
	var body struct {
		Tag           string `json:"tag"`
		Target        string `json:"target"`
		Title         string `json:"title"`
		Notes         string `json:"notes"`
		Draft         bool   `json:"draft"`
		Prerelease    bool   `json:"prerelease"`
		GenerateNotes bool   `json:"generate_notes"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if !s.git.ValidRef("refs/tags/" + body.Tag) {
		writeError(w, http.StatusBadRequest, "invalid tag name "+strconv.Quote(body.Tag))
		return
	}
	if body.Title == "" {
		body.Title = body.Tag
	}
	if problem := checkText(&body.Title, &body.Notes); problem != "" {
		writeError(w, http.StatusBadRequest, problem)
		return
	}
	if _, err := s.store.ReleaseByTag(repo.Name, body.Tag); err == nil {
		writeReleaseError(w, store.ErrExists, body.Tag)
		return
	}
	dir := s.repoDir(repo.Name)
	release := store.Release{
		Repo: repo.Name, Tag: body.Tag, Title: body.Title, Notes: body.Notes, Draft: body.Draft, Prerelease: body.Prerelease, Target: body.Target,
	}
	if release.Target == "" {
		release.Target = s.git.DefaultBranch(dir)
	}
	if body.GenerateNotes {
		commit, err := s.git.ResolveCommit(dir, "refs/tags/"+release.Tag)
		if err != nil {
			commit, err = s.git.ResolveCommit(dir, release.Target)
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "target "+release.Target+" not found")
			return
		}
		generated, err := s.generateNotes(repo.Name, release.Tag, commit, release.Prerelease)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "release notes: "+err.Error())
			return
		}
		release.Notes = strings.TrimSpace(release.Notes + "\n\n" + generated.Notes)
		if problem := checkText(nil, &release.Notes); problem != "" {
			writeError(w, http.StatusBadRequest, problem)
			return
		}
	}
	createdTag, err := s.tagRelease(dir, &release, body.Target != "", !body.Draft)
	if err != nil {
		writeReleaseError(w, err, body.Tag)
		return
	}
	if !release.Draft {
		published := time.Now().UTC()
		release.PublishedAt = &published
	}
	created, eventID, err := s.store.CreateRelease(identityFrom(r).actor, release)
	if err != nil {
		if createdTag != "" {
			s.untag(dir, body.Tag, createdTag)
		}
		writeReleaseError(w, err, body.Tag)
		return
	}
	writeJSON(w, http.StatusCreated, struct {
		store.Release
		EventID int64 `json:"event_id"`
	}{s.releaseLinks(r, created), eventID})
}

func (s *Server) handleEditRelease(w http.ResponseWriter, r *http.Request) {
	repo, release, ok := s.lookupRelease(w, r, "write")
	if !ok {
		return
	}
	var body struct {
		Title      *string `json:"title"`
		Notes      *string `json:"notes"`
		Draft      *bool   `json:"draft"`
		Prerelease *bool   `json:"prerelease"`
		Target     *string `json:"target"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if !release.Draft && body.Draft != nil && *body.Draft {
		writeError(w, http.StatusBadRequest, "a published release can't go back to draft")
		return
	}
	if !release.Draft && body.Target != nil {
		writeError(w, http.StatusBadRequest, "the target of a published release can't change")
		return
	}
	if problem := checkText(body.Title, body.Notes); problem != "" {
		writeError(w, http.StatusBadRequest, problem)
		return
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
	publish := release.Draft && body.Draft != nil && !*body.Draft
	dir := s.repoDir(repo.Name)
	createdTag := ""
	if release.Draft {
		var err error
		if createdTag, err = s.tagRelease(dir, &release, body.Target != nil, publish); err != nil {
			writeReleaseError(w, err, release.Tag)
			return
		}
	}
	action := "release.updated"
	if publish {
		action = "release.published"
		published := time.Now().UTC()
		release.Draft = false
		release.PublishedAt = &published
	}
	updated, eventID, err := s.store.UpdateRelease(identityFrom(r).actor, release, action)
	if err != nil {
		if createdTag != "" {
			s.untag(dir, release.Tag, createdTag)
		}
		writeReleaseError(w, err, release.Tag)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		store.Release
		EventID int64 `json:"event_id"`
	}{s.releaseLinks(r, updated), eventID})
}

func (s *Server) deleteObject(r *http.Request, repo string, asset store.Asset) {
	err := s.storage.Delete(r.Context(), releasesPath(repo)+"/"+asset.Object)
	var missing driver.PathNotFoundError
	if err != nil && !errors.As(err, &missing) {
		slog.Error("delete release asset", "repo", repo, "asset", asset.Name, "object", asset.Object, "error", err)
	}
}

func (s *Server) handleDeleteRelease(w http.ResponseWriter, r *http.Request) {
	repo, release, ok := s.lookupRelease(w, r, "write")
	if !ok {
		return
	}
	eventID, err := s.store.DeleteRelease(identityFrom(r).actor, release)
	if err != nil {
		writeReleaseError(w, err, release.Tag)
		return
	}
	for _, asset := range release.Assets {
		s.deleteObject(r, repo.Name, asset)
	}
	tagDeleted := false
	if r.URL.Query().Get("cleanup_tag") == "true" {
		dir := s.repoDir(repo.Name)
		if sha, err := s.git.ResolveCommit(dir, "refs/tags/"+release.Tag); err == nil {
			if err := s.git.UpdateRef(dir, "refs/tags/"+release.Tag, "", ""); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			tagDeleted = true
			slog.Info("deleted release tag", "repo", repo.Name, "tag", release.Tag, "sha", sha)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": release.Tag, "tag_deleted": tagDeleted, "event_id": eventID})
}

func (s *Server) handleUploadAsset(w http.ResponseWriter, r *http.Request) {
	repo, release, ok := s.lookupRelease(w, r, "write")
	if !ok {
		return
	}
	name := r.URL.Query().Get("name")
	if problem := checkAssetName(name); problem != "" {
		writeError(w, http.StatusBadRequest, problem)
		return
	}
	contentType := r.Header.Get("Content-Type")
	if _, _, err := mime.ParseMediaType(contentType); err != nil {
		contentType = "application/octet-stream"
	}
	random := make([]byte, 16)
	rand.Read(random)
	asset := store.Asset{Name: name, ContentType: contentType, Object: hex.EncodeToString(random)}
	ctx := r.Context()
	writer, err := s.storage.Writer(ctx, releasesPath(repo.Name)+"/"+asset.Object, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	hash := sha256.New()
	if asset.Size, err = io.Copy(io.MultiWriter(writer, hash), r.Body); err != nil {
		writer.Cancel(ctx)
		writer.Close()
		writeError(w, http.StatusBadRequest, "upload: "+err.Error())
		return
	}
	if err := writer.Commit(ctx); err != nil {
		writer.Close()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := writer.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	asset.SHA256 = hex.EncodeToString(hash.Sum(nil))
	created, replaced, eventID, err := s.store.AddAsset(identityFrom(r).actor, release, asset, r.URL.Query().Get("clobber") == "true")
	if err != nil {
		s.deleteObject(r, repo.Name, asset)
		if errors.Is(err, store.ErrExists) {
			writeError(w, http.StatusConflict, "asset "+name+" already exists; pass clobber=true to replace it")
			return
		}
		writeStoreError(w, err, "release not found")
		return
	}
	if replaced != nil {
		s.deleteObject(r, repo.Name, *replaced)
	}
	created.DownloadURL = downloadURL(r, repo.Name, release.Tag, created.Name)
	writeJSON(w, http.StatusCreated, struct {
		store.Asset
		EventID int64 `json:"event_id"`
	}{created, eventID})
}

func (s *Server) handleDeleteAsset(w http.ResponseWriter, r *http.Request) {
	repo, release, ok := s.lookupRelease(w, r, "write")
	if !ok {
		return
	}
	name := pathParam(r, "name")
	asset, eventID, err := s.store.DeleteAsset(identityFrom(r).actor, release, name)
	if err != nil {
		writeStoreError(w, err, "asset "+name+" not found")
		return
	}
	s.deleteObject(r, repo.Name, asset)
	writeJSON(w, http.StatusOK, map[string]any{"deleted": name, "event_id": eventID})
}

func (s *Server) serveAsset(w http.ResponseWriter, r *http.Request, release store.Release, name string) {
	index := slices.IndexFunc(release.Assets, func(asset store.Asset) bool { return asset.Name == name })
	if index < 0 {
		writeError(w, http.StatusNotFound, "asset "+name+" not found")
		return
	}
	asset := release.Assets[index]
	etag := `"` + asset.SHA256 + `"`
	w.Header().Set("ETag", etag)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", asset.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(asset.Size, 10))
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": asset.Name}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method == http.MethodHead {
		return
	}
	reader, err := s.storage.Reader(r.Context(), releasesPath(release.Repo)+"/"+asset.Object, 0)
	if err != nil {
		w.Header().Del("Content-Length")
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer reader.Close()
	if _, err := io.Copy(w, reader); err != nil {
		slog.Error("download release asset", "repo", release.Repo, "asset", asset.Name, "error", err)
	}
}

func (s *Server) handleDownloadAsset(w http.ResponseWriter, r *http.Request) {
	_, release, ok := s.lookupRelease(w, r, "read")
	if !ok {
		return
	}
	s.serveAsset(w, r, release, pathParam(r, "name"))
}

func (s *Server) handleStableDownload(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	rest := pathParam(r, "*")
	slash := strings.LastIndex(rest, "/")
	if slash < 0 {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}
	tag, name := rest[:slash], rest[slash+1:]
	release, err := s.store.ReleaseByTag(repo.Name, tag)
	release, err = s.visibleRelease(r, repo.Name, release, err)
	if err != nil {
		writeStoreError(w, err, "no release for tag "+tag)
		return
	}
	s.serveAsset(w, r, release, name)
}

func (s *Server) handleLatestDownload(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	release, err := s.store.LatestRelease(repo.Name)
	if err != nil {
		writeStoreError(w, err, "no published release")
		return
	}
	s.serveAsset(w, r, release, pathParam(r, "name"))
}

type notePull struct {
	Number int64  `json:"number"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

type releaseNotes struct {
	Notes       string     `json:"notes"`
	PreviousTag string     `json:"previous_tag"`
	Pulls       []notePull `json:"pulls"`
}

func (s *Server) generateNotes(repo, tag, commit string, prerelease bool) (releaseNotes, error) {
	dir := s.repoDir(repo)
	releases, err := s.store.Releases(repo)
	if err != nil {
		return releaseNotes{}, err
	}
	notes := releaseNotes{Pulls: []notePull{}}
	exclude := []string{}
	for _, release := range releases {
		if release.Draft || release.Tag == tag || release.Commit == "" || (release.Prerelease && !prerelease) {
			continue
		}
		if ancestor, err := s.git.IsAncestor(dir, release.Commit, commit); err == nil && ancestor {
			notes.PreviousTag = release.Tag
			exclude = append(exclude, release.Commit)
			break
		}
	}
	commits, err := s.git.RevList(dir, commit, exclude...)
	if err != nil {
		return notes, err
	}
	reachable := map[string]bool{}
	for _, sha := range commits {
		reachable[sha] = true
	}
	pulls, err := s.store.Pulls(repo, "merged")
	if err != nil {
		return notes, err
	}
	slices.Reverse(pulls)
	for _, pull := range pulls {
		if !reachable[pull.MergeSHA] {
			continue
		}
		author := pull.Author.Agent
		if author == "" {
			author = pull.Author.User
		}
		notes.Pulls = append(notes.Pulls, notePull{Number: pull.Number, Title: pull.Title, Author: author})
	}
	since := ""
	if notes.PreviousTag != "" {
		since = " since " + notes.PreviousTag
	}
	if len(notes.Pulls) == 0 {
		notes.Notes = "No pull requests merged" + since + "."
		return notes, nil
	}
	var text strings.Builder
	text.WriteString("## Pull requests merged" + since + "\n\n")
	for _, pull := range notes.Pulls {
		text.WriteString(fmt.Sprintf("- %s (#%d) by %s\n", pull.Title, pull.Number, pull.Author))
	}
	notes.Notes = strings.TrimSuffix(text.String(), "\n")
	return notes, nil
}

func (s *Server) handleReleaseNotes(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	query := r.URL.Query()
	dir := s.repoDir(repo.Name)
	tag := query.Get("tag")
	commit, err := s.git.ResolveCommit(dir, "refs/tags/"+tag)
	if tag == "" || err != nil {
		target := query.Get("target")
		if target == "" {
			target = s.git.DefaultBranch(dir)
		}
		if commit, err = s.git.ResolveCommit(dir, target); err != nil {
			writeError(w, http.StatusBadRequest, "target "+target+" not found")
			return
		}
	}
	notes, err := s.generateNotes(repo.Name, tag, commit, query.Get("prerelease") == "true")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, notes)
}

func (s *Server) uiReleases(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.uiLookup(w, r)
	if !ok {
		return
	}
	tag := pathParam(r, "*")
	var releases []store.Release
	var err error
	if tag == "" {
		releases, err = s.store.Releases(repo.Name)
	} else {
		var release store.Release
		release, err = s.store.ReleaseByTag(repo.Name, tag)
		release, err = s.visibleRelease(r, repo.Name, release, err)
		releases = []store.Release{release}
	}
	if err != nil {
		s.renderStoreError(w, r, err, "No release for tag "+tag+".")
		return
	}
	drafts := identityFrom(r).can("write", repo.Name)
	view := struct {
		Releases []store.Release
		Latest   string
		Single   bool
	}{Releases: []store.Release{}, Single: tag != ""}
	if latest, err := s.store.LatestRelease(repo.Name); err == nil {
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
	s.render(w, r, http.StatusOK, "releases", page{Title: title, Repo: &repo, Ref: s.git.DefaultBranch(s.repoDir(repo.Name)), CloneURL: cloneURL(r, repo.Name), Data: view})
}
