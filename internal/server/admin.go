package server

import (
	"archive/tar"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/distribution/distribution/v3/registry/storage"
	"github.com/distribution/distribution/v3/registry/storage/driver"

	"github.com/muchobien/lyra/internal/backup"
	"github.com/muchobien/lyra/internal/store"
)

const (
	readOnlyMessage = "Lyra is read-only for maintenance; try again later"
	registryBlobs   = "/docker/registry/v2/blobs"
)

type backupCreated struct {
	Bytes   int64 `json:"bytes"`
	Objects int   `json:"objects"`
	PauseMS int64 `json:"pause_ms"`
	Repos   int   `json:"repos"`
}

type lastBackup struct {
	At time.Time `json:"at"`
	backupCreated
}

type statusView struct {
	LastBackup *lastBackup `json:"last_backup"`
	ReadOnly   bool        `json:"read_only"`
}

type readOnlyRequest struct {
	Enabled bool `json:"enabled"`
}

type storageCheck struct {
	Checked int                  `json:"checked"`
	Missing []store.StoredObject `json:"missing"`
}

type registryCollected struct {
	BlobsAfter  int   `json:"blobs_after"`
	BlobsBefore int   `json:"blobs_before"`
	BytesFreed  int64 `json:"bytes_freed"`
	MS          int64 `json:"ms"`
}

type countingWriter struct {
	writer io.Writer
	count  int64
}

func (counter *countingWriter) Write(data []byte) (int, error) {
	written, err := counter.writer.Write(data)
	counter.count += int64(written)
	return written, err
}

func isWrite(r *http.Request) bool {
	switch {
	case r.URL.Path == "/api/markdown":
		return false
	case strings.HasSuffix(r.URL.Path, "/git-upload-pack"):
		return false
	case r.Method == http.MethodGet && r.URL.Query().Get("service") == receivePack:
		return true
	}
	return r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions
}

func (s *Server) writeGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isWrite(r) || strings.HasPrefix(r.URL.Path, "/api/admin/") {
			next.ServeHTTP(w, r)
			return
		}
		if s.readOnly.Load() {
			if strings.HasPrefix(r.URL.Path, "/v2/") {
				registryError(w, http.StatusServiceUnavailable, "UNAVAILABLE", readOnlyMessage)
				return
			}
			writeError(w, http.StatusServiceUnavailable, readOnlyMessage)
			return
		}
		s.gate.RLock()
		defer s.gate.RUnlock()
		next.ServeHTTP(w, r)
	})
}

func linkTree(source, target string) error {
	return filepath.WalkDir(source, func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, file)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		switch {
		case entry.IsDir():
			return os.MkdirAll(destination, 0o750)
		case entry.Type().IsRegular():
			return os.Link(file, destination)
		}
		return nil
	})
}

func addFile(archive *tar.Writer, file, name string, entry fs.DirEntry) error {
	info, err := entry.Info()
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = name
	if entry.IsDir() {
		header.Name += "/"
	}
	if err := archive.WriteHeader(header); err != nil {
		return err
	}
	if !entry.Type().IsRegular() {
		return nil
	}
	source, err := os.Open(file)
	if err != nil {
		return err
	}
	defer source.Close()
	_, err = io.Copy(archive, source)
	return err
}

func addTree(archive *tar.Writer, root, prefix string) error {
	return filepath.WalkDir(root, func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, file)
		if err != nil {
			return err
		}
		return addFile(archive, file, filepath.ToSlash(filepath.Join(prefix, relative)), entry)
	})
}

func (s *Server) stage(ctx context.Context, staging string) (time.Duration, error) {
	start := time.Now()
	s.gate.Lock()
	defer s.gate.Unlock()
	if err := os.MkdirAll(staging, 0o750); err != nil {
		return time.Since(start), err
	}
	if err := s.store.Snapshot(ctx, filepath.Join(staging, "lyra.db")); err != nil {
		return time.Since(start), fmt.Errorf("snapshot: %w", err)
	}
	if err := linkTree(filepath.Join(s.config.DataDir, "repos"), filepath.Join(staging, "repos")); err != nil {
		return time.Since(start), fmt.Errorf("copy repos: %w", err)
	}
	return time.Since(start), nil
}

func (s *Server) manifest(ctx context.Context, staging string) (backup.Manifest, error) {
	manifest := backup.Manifest{Format: backup.Format, CreatedAt: time.Now().UTC(), Repos: []backup.Repo{}}
	snapshot, err := store.Open(filepath.Join(staging, "lyra.db"))
	if err != nil {
		return manifest, err
	}
	defer snapshot.Close()
	if manifest.Schema, err = snapshot.SchemaVersion(ctx); err != nil {
		return manifest, err
	}
	if manifest.Objects, err = snapshot.StoredObjects(ctx); err != nil {
		return manifest, err
	}
	stored, err := snapshot.Repos(ctx)
	if err != nil {
		return manifest, err
	}
	for _, repo := range stored {
		refs, err := s.git.Refs(ctx, filepath.Join(staging, "repos", filepath.FromSlash(repo.Name)+".git"))
		if err != nil {
			return manifest, fmt.Errorf("%s: %w", repo.Name, err)
		}
		manifest.Repos = append(manifest.Repos, backup.Repo{Name: repo.Name, Refs: refs})
	}
	return manifest, nil
}

func writeBackup(writer io.Writer, staging string, manifest backup.Manifest) error {
	archive := tar.NewWriter(writer)
	if err := addTree(archive, filepath.Join(staging, "lyra.db"), "lyra.db"); err != nil {
		return err
	}
	if err := addTree(archive, filepath.Join(staging, "repos"), "repos"); err != nil {
		return err
	}
	encoded, err := json.Marshal(manifest, jsontext.WithIndent("  "), json.Deterministic(true))
	if err != nil {
		return err
	}
	header := &tar.Header{
		Name:     "manifest.json",
		Mode:     0o644,
		Size:     int64(len(encoded)),
		ModTime:  manifest.CreatedAt,
		Typeflag: tar.TypeReg,
	}
	if err := archive.WriteHeader(header); err != nil {
		return err
	}
	if _, err := archive.Write(encoded); err != nil {
		return err
	}
	return archive.Close()
}

func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	if !identityFrom(r).can(scopeAdmin, "") {
		return forbidden("backups need an unrestricted admin token")
	}
	staging := filepath.Join(s.config.DataDir, "tmp", "backup-"+randomHex(8))
	defer os.RemoveAll(staging)
	pause, err := s.stage(ctx, staging)
	slog.Info("backup staged", "pause", pause, "error", err)
	if err != nil {
		return err
	}
	manifest, err := s.manifest(ctx, staging)
	if err != nil {
		return err
	}
	filename := "lyra-" + manifest.CreatedAt.Format("20060102-150405") + ".tar"
	w.Header().Set("Content-Type", "application/x-tar")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Lyra-Backup-Pause-Ms", fmt.Sprint(pause.Milliseconds()))
	counter := &countingWriter{writer: w}
	if err := writeBackup(counter, staging, manifest); err != nil {
		slog.Error("backup", "error", err)
		return nil
	}
	created := backupCreated{
		Bytes:   counter.count,
		Objects: len(manifest.Objects),
		PauseMS: pause.Milliseconds(),
		Repos:   len(manifest.Repos),
	}
	if _, err := s.store.AddEvent(ctx, "backup.created", "", identityFrom(r).actor, created); err != nil {
		slog.Error("record backup", "error", err)
	}
	return nil
}

func (s *Server) handleReadOnly(w http.ResponseWriter, r *http.Request) error {
	id := identityFrom(r)
	if !id.can(scopeAdmin, "") {
		return forbidden("read-only mode needs an unrestricted admin token")
	}
	body, err := decodeBody[readOnlyRequest](w, r)
	if err != nil {
		return err
	}
	if s.readOnly.Swap(body.Enabled) != body.Enabled {
		action := "readonly.off"
		if body.Enabled {
			action = "readonly.on"
		}
		if _, err := s.store.AddEvent(r.Context(), action, "", id.actor, noData{}); err != nil {
			return err
		}
	}
	return s.handleStatus(w, r)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) error {
	if !identityFrom(r).can(scopeAdmin, "") {
		return forbidden("status needs an unrestricted admin token")
	}
	status := statusView{ReadOnly: s.readOnly.Load()}
	last, err := s.store.LatestEvent(r.Context(), "backup.created")
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	if err == nil {
		status.LastBackup = &lastBackup{At: last.At}
		if err := json.Unmarshal(last.Data, &status.LastBackup.backupCreated); err != nil {
			return err
		}
	}
	return writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleStorageCheck(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	if !identityFrom(r).can(scopeAdmin, "") {
		return forbidden("the storage check needs an unrestricted admin token")
	}
	objects, err := s.store.StoredObjects(ctx)
	if err != nil {
		return err
	}
	missing := []store.StoredObject{}
	for _, object := range objects {
		info, err := s.storage.Stat(ctx, object.Path)
		_, absent := errors.AsType[driver.PathNotFoundError](err)
		switch {
		case absent:
			missing = append(missing, object)
		case err != nil:
			return err
		case object.Size > 0 && info.Size() != object.Size:
			missing = append(missing, object)
		}
	}
	return writeJSON(w, http.StatusOK, storageCheck{Checked: len(objects), Missing: missing})
}

func (s *Server) registryUsage(ctx context.Context) (int, int64, error) {
	count, size := 0, int64(0)
	err := s.storage.Walk(ctx, registryBlobs, func(info driver.FileInfo) error {
		if !info.IsDir() && path.Base(info.Path()) == "data" {
			count++
			size += info.Size()
		}
		return nil
	})
	if _, absent := errors.AsType[driver.PathNotFoundError](err); absent {
		return 0, 0, nil
	}
	return count, size, err
}

func (s *Server) collectRegistry(ctx context.Context) error {
	namespace, err := storage.NewRegistry(ctx, s.storage, storage.EnableDelete)
	if err != nil {
		return fmt.Errorf("garbage collection: %w", err)
	}
	options := storage.GCOpts{RemoveUntagged: true, Quiet: true}
	if err := storage.MarkAndSweep(ctx, s.storage, namespace, options); err != nil {
		return fmt.Errorf("garbage collection: %w", err)
	}
	return nil
}

func (s *Server) handleRegistryGC(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id := identityFrom(r)
	if !id.can(scopeAdmin, "") {
		return forbidden("registry garbage collection needs an unrestricted admin token")
	}
	wasReadOnly := s.readOnly.Swap(true)
	defer s.readOnly.Store(wasReadOnly)
	s.gate.Lock()
	s.gate.Unlock()
	start := time.Now()
	blobsBefore, bytesBefore, err := s.registryUsage(ctx)
	if err != nil {
		return err
	}
	if err := s.collectRegistry(ctx); err != nil {
		return err
	}
	blobsAfter, bytesAfter, err := s.registryUsage(ctx)
	if err != nil {
		return err
	}
	result := registryCollected{
		BlobsAfter:  blobsAfter,
		BlobsBefore: blobsBefore,
		BytesFreed:  bytesBefore - bytesAfter,
		MS:          time.Since(start).Milliseconds(),
	}
	if _, err := s.store.AddEvent(ctx, "registry.gc", "", id.actor, result); err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, result)
}
