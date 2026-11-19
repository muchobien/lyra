package server

import (
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/muchobien/lyra/internal/store"
)

var attachmentTypes = []string{
	"image/png",
	"image/jpeg",
	"image/gif",
	"image/webp",
	"image/avif",
	"video/mp4",
	"video/webm",
	"video/quicktime",
}

type uploadAttachmentInput struct {
	RepoPath
	Name string `query:"name" required:"true" doc:"File name, shown in the attachment's URL."`
}

func attachmentsPath(repo string) string {
	return "/lyra/attachments/" + repo
}

func attachmentPath(attachment store.Attachment) string {
	return attachmentsPath(attachment.Repo) + "/" + attachment.Object
}

func attachmentURL(r *http.Request, attachment store.Attachment) string {
	id := strconv.FormatInt(attachment.ID, 10)
	return baseURL(r) + "/" + attachment.Repo + "/attachments/" + id + "/" + url.PathEscape(attachment.Name)
}

func (s *Server) handleUploadAttachment(w http.ResponseWriter, r *http.Request, input *uploadAttachmentInput) error {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeWrite, input.fullName())
	if err != nil {
		return err
	}
	if !validAssetName(input.Name) {
		return badRequest("attachment name must be 1 to 255 characters, " +
			"with no slashes, control characters or surrounding spaces")
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !slices.Contains(attachmentTypes, contentType) {
		return badRequest("attachments are PNG, JPEG, GIF, WebP or AVIF images, or MP4, WebM or QuickTime videos")
	}
	attachment := store.Attachment{Repo: repo.Name, Name: input.Name, ContentType: contentType, Object: randomHex(16)}
	path := attachmentPath(attachment)
	if attachment.Size, attachment.SHA256, err = s.storeObject(ctx, path, r.Body); err != nil {
		return err
	}
	created, err := s.store.AddAttachment(ctx, identityFrom(r).actor, attachment)
	if err != nil {
		if err := s.storage.Delete(ctx, path); err != nil {
			slog.Error("delete attachment", "repo", repo.Name, "object", attachment.Object, "error", err)
		}
		return err
	}
	created.URL = attachmentURL(r, created)
	return writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleDownloadAttachment(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeRead, routeRepo(r))
	if err != nil {
		return err
	}
	id, err := strconv.ParseInt(pathParam(r, "id"), 10, 64)
	if err != nil {
		return notFound("attachment not found")
	}
	attachment, err := s.store.Attachment(r.Context(), repo.Name, id)
	if err != nil {
		return missing(err, "attachment not found")
	}
	disposition := mime.FormatMediaType("inline", map[string]string{"filename": attachment.Name})
	w.Header().Set("ETag", `"`+attachment.SHA256+`"`)
	w.Header().Set("Content-Type", attachment.ContentType)
	w.Header().Set("Content-Disposition", disposition)
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	file := &storageFile{
		ctx:     r.Context(),
		storage: s.storage,
		path:    attachmentPath(attachment),
		size:    attachment.Size,
	}
	defer file.Close()
	http.ServeContent(w, r, attachment.Name, attachment.CreatedAt, file)
	return nil
}
