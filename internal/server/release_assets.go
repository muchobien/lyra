package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"slices"
	"strings"
	"unicode"

	"github.com/distribution/distribution/v3/registry/storage/driver"

	"github.com/muchobien/lyra/internal/store"
)

const (
	maxAssetName     = 255
	assetNameProblem = "asset name must be 1 to 255 characters, " +
		"with no slashes, control characters or surrounding spaces"
)

type assetWrite struct {
	store.Asset
	EventID int64 `json:"event_id"`
}

type storageFile struct {
	ctx     context.Context
	storage driver.StorageDriver
	path    string
	size    int64
	offset  int64
	reader  io.ReadCloser
}

func (file *storageFile) Read(buffer []byte) (int, error) {
	if file.offset >= file.size {
		return 0, io.EOF
	}
	if file.reader == nil {
		reader, err := file.storage.Reader(file.ctx, file.path, file.offset)
		if err != nil {
			return 0, err
		}
		file.reader = reader
	}
	read, err := file.reader.Read(buffer)
	file.offset += int64(read)
	return read, err
}

func (file *storageFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekCurrent:
		offset += file.offset
	case io.SeekEnd:
		offset += file.size
	}
	if offset < 0 {
		return 0, errors.New("negative offset")
	}
	if offset != file.offset {
		file.Close()
	}
	file.offset = offset
	return offset, nil
}

func (file *storageFile) Close() error {
	if file.reader == nil {
		return nil
	}
	err := file.reader.Close()
	file.reader = nil
	return err
}

func validAssetName(name string) bool {
	return name != "" && len(name) <= maxAssetName && name != "." && name != ".." &&
		strings.TrimSpace(name) == name &&
		!strings.ContainsAny(name, `/\`) &&
		!strings.ContainsFunc(name, unicode.IsControl)
}

func assetPath(repo string, asset store.Asset) string {
	return releasesPath(repo) + "/" + asset.Object
}

func (s *Server) deleteObject(ctx context.Context, repo string, asset store.Asset) {
	err := s.storage.Delete(ctx, assetPath(repo, asset))
	if _, absent := errors.AsType[driver.PathNotFoundError](err); err != nil && !absent {
		slog.Error("delete release asset", "repo", repo, "asset", asset.Name, "object", asset.Object, "error", err)
	}
}

func (s *Server) storeObject(ctx context.Context, repo string, asset *store.Asset, content io.Reader) error {
	writer, err := s.storage.Writer(ctx, assetPath(repo, *asset), false)
	if err != nil {
		return err
	}
	hash := sha256.New()
	if asset.Size, err = io.Copy(io.MultiWriter(writer, hash), content); err != nil {
		writer.Cancel(ctx)
		writer.Close()
		return badRequest("upload: " + err.Error())
	}
	if err := writer.Commit(ctx); err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	asset.SHA256 = hex.EncodeToString(hash.Sum(nil))
	return nil
}

func (s *Server) handleUploadAsset(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, release, err := s.lookupRelease(r, scopeWrite)
	if err != nil {
		return err
	}
	name := r.URL.Query().Get("name")
	if !validAssetName(name) {
		return badRequest(assetNameProblem)
	}
	contentType := r.Header.Get("Content-Type")
	if _, _, err := mime.ParseMediaType(contentType); err != nil {
		contentType = "application/octet-stream"
	}
	asset := store.Asset{Name: name, ContentType: contentType, Object: randomHex(16)}
	if err := s.storeObject(ctx, repo.Name, &asset, r.Body); err != nil {
		return err
	}
	clobber := r.URL.Query().Get("clobber") == "true"
	created, replaced, eventID, err := s.store.AddAsset(ctx, identityFrom(r).actor, release, asset, clobber)
	if errors.Is(err, store.ErrExists) {
		s.deleteObject(ctx, repo.Name, asset)
		return conflict("asset " + name + " already exists; pass clobber=true to replace it")
	}
	if err != nil {
		s.deleteObject(ctx, repo.Name, asset)
		return missing(err, "release not found")
	}
	if replaced != nil {
		s.deleteObject(ctx, repo.Name, *replaced)
	}
	created.DownloadURL = downloadURL(r, repo.Name, release.Tag, created.Name)
	return writeJSON(w, http.StatusCreated, assetWrite{Asset: created, EventID: eventID})
}

func (s *Server) handleDeleteAsset(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, release, err := s.lookupRelease(r, scopeWrite)
	if err != nil {
		return err
	}
	name := pathParam(r, "name")
	asset, eventID, err := s.store.DeleteAsset(ctx, identityFrom(r).actor, release, name)
	if err != nil {
		return missing(err, "asset "+name+" not found")
	}
	s.deleteObject(ctx, repo.Name, asset)
	return writeJSON(w, http.StatusOK, deletedWithEvent{Deleted: name, EventID: eventID})
}

func (s *Server) serveAsset(w http.ResponseWriter, r *http.Request, release store.Release, name string) error {
	index := slices.IndexFunc(release.Assets, func(asset store.Asset) bool { return asset.Name == name })
	if index < 0 {
		return notFound("asset " + name + " not found")
	}
	asset := release.Assets[index]
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": asset.Name})
	w.Header().Set("ETag", `"`+asset.SHA256+`"`)
	w.Header().Set("Content-Type", asset.ContentType)
	w.Header().Set("Content-Disposition", disposition)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	file := &storageFile{
		ctx:     r.Context(),
		storage: s.storage,
		path:    assetPath(release.Repo, asset),
		size:    asset.Size,
	}
	defer file.Close()
	http.ServeContent(w, r, asset.Name, asset.CreatedAt, file)
	return nil
}

func (s *Server) handleDownloadAsset(w http.ResponseWriter, r *http.Request) error {
	_, release, err := s.lookupRelease(r, scopeRead)
	if err != nil {
		return err
	}
	return s.serveAsset(w, r, release, pathParam(r, "name"))
}

func (s *Server) handleStableDownload(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	rest := pathParam(r, "*")
	slash := strings.LastIndex(rest, "/")
	if slash < 0 {
		return notFound("asset not found")
	}
	tag, name := rest[:slash], rest[slash+1:]
	release, err := s.releaseByTag(r, repo.Name, tag)
	if err != nil {
		return missing(err, "no release for tag "+tag)
	}
	return s.serveAsset(w, r, release, name)
}

func (s *Server) handleLatestDownload(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	release, err := s.store.LatestRelease(r.Context(), repo.Name)
	if err != nil {
		return missing(err, "no published release")
	}
	return s.serveAsset(w, r, release, pathParam(r, "name"))
}
