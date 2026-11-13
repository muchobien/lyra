package server

import (
	"archive/tar"
	"context"
	"encoding/json"
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

const readOnlyMessage = "Lyra is read-only for maintenance; try again later"

func isWrite(r *http.Request) bool {
	switch {
	case strings.HasSuffix(r.URL.Path, "/git-upload-pack"):
		return false
	case r.Method == http.MethodGet && r.URL.Query().Get("service") == "git-receive-pack":
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
			} else {
				writeError(w, http.StatusServiceUnavailable, readOnlyMessage)
			}
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

type countingWriter struct {
	writer io.Writer
	count  int64
}

func (counter *countingWriter) Write(data []byte) (int, error) {
	written, err := counter.writer.Write(data)
	counter.count += int64(written)
	return written, err
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
		info, err := entry.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(filepath.Join(prefix, relative))
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
	})
}

func (s *Server) stage(staging string) (time.Duration, error) {
	start := time.Now()
	s.gate.Lock()
	defer s.gate.Unlock()
	if err := os.MkdirAll(staging, 0o750); err != nil {
		return time.Since(start), err
	}
	if err := s.store.Snapshot(filepath.Join(staging, "lyra.db")); err != nil {
		return time.Since(start), fmt.Errorf("snapshot: %w", err)
	}
	if err := linkTree(filepath.Join(s.config.DataDir, "repos"), filepath.Join(staging, "repos")); err != nil {
		return time.Since(start), fmt.Errorf("copy repos: %w", err)
	}
	return time.Since(start), nil
}

func (s *Server) manifest(staging string) (backup.Manifest, error) {
	manifest := backup.Manifest{Format: backup.Format, CreatedAt: time.Now().UTC(), Repos: []backup.Repo{}}
	snapshot, err := store.Open(filepath.Join(staging, "lyra.db"))
	if err != nil {
		return manifest, err
	}
	defer snapshot.Close()
	if manifest.Schema, err = snapshot.SchemaVersion(); err != nil {
		return manifest, err
	}
	if manifest.Objects, err = snapshot.StoredObjects(); err != nil {
		return manifest, err
	}
	stored, err := snapshot.Repos()
	if err != nil {
		return manifest, err
	}
	for _, repo := range stored {
		refs, err := s.git.Refs(filepath.Join(staging, "repos", filepath.FromSlash(repo.Name)+".git"))
		if err != nil {
			return manifest, fmt.Errorf("%s: %w", repo.Name, err)
		}
		manifest.Repos = append(manifest.Repos, backup.Repo{Name: repo.Name, Refs: refs})
	}
	return manifest, nil
}

func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	if !identityFrom(r).can("admin", "") {
		writeError(w, http.StatusForbidden, "backups need an unrestricted admin token")
		return
	}
	staging := filepath.Join(s.config.DataDir, "tmp", "backup-"+randomHex(8))
	defer os.RemoveAll(staging)
	pause, err := s.stage(staging)
	slog.Info("backup staged", "pause", pause, "error", err)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	manifest, err := s.manifest(staging)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/x-tar")
	w.Header().Set("Content-Disposition", `attachment; filename="lyra-`+manifest.CreatedAt.Format("20060102-150405")+`.tar"`)
	w.Header().Set("Lyra-Backup-Pause-Ms", fmt.Sprint(pause.Milliseconds()))
	counter := &countingWriter{writer: w}
	archive := tar.NewWriter(counter)
	err = addTree(archive, filepath.Join(staging, "lyra.db"), "lyra.db")
	if err == nil {
		err = addTree(archive, filepath.Join(staging, "repos"), "repos")
	}
	if err == nil {
		var encoded []byte
		if encoded, err = json.MarshalIndent(manifest, "", "  "); err == nil {
			if err = archive.WriteHeader(&tar.Header{Name: "manifest.json", Mode: 0o644, Size: int64(len(encoded)), ModTime: manifest.CreatedAt, Typeflag: tar.TypeReg}); err == nil {
				_, err = archive.Write(encoded)
			}
		}
	}
	if err == nil {
		err = archive.Close()
	}
	if err != nil {
		slog.Error("backup", "error", err)
		return
	}
	if _, err := s.store.AddEvent("backup.created", "", identityFrom(r).actor, map[string]any{
		"bytes": counter.count, "pause_ms": pause.Milliseconds(), "repos": len(manifest.Repos), "objects": len(manifest.Objects),
	}); err != nil {
		slog.Error("record backup", "error", err)
	}
}

func (s *Server) handleReadOnly(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	if !id.can("admin", "") {
		writeError(w, http.StatusForbidden, "read-only mode needs an unrestricted admin token")
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if s.readOnly.Swap(body.Enabled) != body.Enabled {
		action := "readonly.off"
		if body.Enabled {
			action = "readonly.on"
		}
		if _, err := s.store.AddEvent(action, "", id.actor, map[string]any{}); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	s.handleStatus(w, r)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if !identityFrom(r).can("admin", "") {
		writeError(w, http.StatusForbidden, "status needs an unrestricted admin token")
		return
	}
	status := map[string]any{"read_only": s.readOnly.Load(), "last_backup": nil}
	last, err := s.store.LatestEvent("backup.created")
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err == nil {
		details := map[string]any{}
		json.Unmarshal(last.Data, &details)
		details["at"] = last.At
		status["last_backup"] = details
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleStorageCheck(w http.ResponseWriter, r *http.Request) {
	if !identityFrom(r).can("admin", "") {
		writeError(w, http.StatusForbidden, "the storage check needs an unrestricted admin token")
		return
	}
	objects, err := s.store.StoredObjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	missing := []store.StoredObject{}
	for _, object := range objects {
		info, err := s.storage.Stat(r.Context(), object.Path)
		var notFound driver.PathNotFoundError
		switch {
		case errors.As(err, &notFound):
			missing = append(missing, object)
		case err != nil:
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		case object.Size > 0 && info.Size() != object.Size:
			missing = append(missing, object)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"checked": len(objects), "missing": missing})
}

func (s *Server) registryUsage(ctx context.Context) (int, int64, error) {
	count, size := 0, int64(0)
	err := s.storage.Walk(ctx, "/docker/registry/v2/blobs", func(info driver.FileInfo) error {
		if !info.IsDir() && path.Base(info.Path()) == "data" {
			count++
			size += info.Size()
		}
		return nil
	})
	var missing driver.PathNotFoundError
	if errors.As(err, &missing) {
		return 0, 0, nil
	}
	return count, size, err
}

func (s *Server) handleRegistryGC(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	if !id.can("admin", "") {
		writeError(w, http.StatusForbidden, "registry garbage collection needs an unrestricted admin token")
		return
	}
	ctx := r.Context()
	wasReadOnly := s.readOnly.Swap(true)
	defer s.readOnly.Store(wasReadOnly)
	s.gate.Lock()
	s.gate.Unlock()
	start := time.Now()
	blobsBefore, bytesBefore, err := s.registryUsage(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	namespace, err := storage.NewRegistry(ctx, s.storage, storage.EnableDelete)
	if err == nil {
		err = storage.MarkAndSweep(ctx, s.storage, namespace, storage.GCOpts{RemoveUntagged: true, Quiet: true})
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "garbage collection: "+err.Error())
		return
	}
	blobsAfter, bytesAfter, err := s.registryUsage(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	result := map[string]any{
		"blobs_before": blobsBefore, "blobs_after": blobsAfter, "bytes_freed": bytesBefore - bytesAfter, "ms": time.Since(start).Milliseconds(),
	}
	if _, err := s.store.AddEvent("registry.gc", "", id.actor, result); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
