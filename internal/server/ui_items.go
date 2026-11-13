package server

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/muchobien/lyra/internal/store"
)

const (
	lineAdd    = "add"
	lineDelete = "delete"
)

var itemPaths = map[string]string{kindIssue: "issues", kindPull: "pulls"}

var lineKinds = map[gitdiff.LineOp]string{
	gitdiff.OpContext: "context",
	gitdiff.OpAdd:     lineAdd,
	gitdiff.OpDelete:  lineDelete,
}

type diffLine struct {
	Kind     string          `json:"kind"`
	Old      int64           `json:"old"`
	New      int64           `json:"new"`
	Text     string          `json:"text"`
	HTML     string          `json:"html"`
	Comments []store.Comment `json:"comments"`
}

type diffFile struct {
	Path    string     `json:"path"`
	OldPath string     `json:"old_path"`
	Binary  bool       `json:"binary"`
	Lines   []diffLine `json:"lines"`
}

func splitList(text string) []string {
	values := []string{}
	for value := range strings.SplitSeq(text, ",") {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func difference(values, without []string) []string {
	remaining := []string{}
	for _, value := range values {
		if !slices.Contains(without, value) {
			remaining = append(remaining, value)
		}
	}
	return remaining
}

func itemURL(repo, kind string, number int64) string {
	return fmt.Sprintf("/%s/%s/%d", repo, itemPaths[kind], number)
}

func (s *Server) uiItem(r *http.Request, kind string, write bool) (store.Repo, store.Item, error) {
	repo, err := s.uiLookup(r)
	if err != nil {
		return repo, store.Item{}, err
	}
	if write && !identityFrom(r).can(scopeWrite, repo.Name) {
		return repo, store.Item{}, forbidden("This token cannot write to " + repo.Name + ".")
	}
	absent := strings.ToUpper(nouns[kind][:1]) + nouns[kind][1:] + " not found."
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil {
		return repo, store.Item{}, notFound(absent)
	}
	item, err := s.store.Item(r.Context(), kind, repo.Name, number)
	return repo, item, missing(err, absent)
}

func (s *Server) uiCreateIssue(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.uiLookup(r)
	if err != nil {
		return err
	}
	id := identityFrom(r)
	if !id.can(scopeWrite, repo.Name) {
		return forbidden("This token cannot write to " + repo.Name + ".")
	}
	title, body := r.FormValue("title"), r.FormValue("body")
	if err := checkText(&title, &body); err != nil {
		return err
	}
	item, _, err := s.store.CreateItem(r.Context(), kindIssue, repo.Name, id.actor, title, body, nil, nil)
	if err != nil {
		return missing(err, "Repository not found.")
	}
	http.Redirect(w, r, itemURL(repo.Name, kindIssue, item.Number), http.StatusSeeOther)
	return nil
}

func commentsOn(comments []store.Comment, file *gitdiff.File, line diffLine) []store.Comment {
	matching := []store.Comment{}
	for _, comment := range comments {
		onOld := comment.Side == sideOld && line.Old == comment.Line &&
			comment.Path == file.OldName && line.Kind != lineAdd
		onNew := comment.Side == sideNew && line.New == comment.Line &&
			comment.Path == file.NewName && line.Kind != lineDelete
		if !comment.Outdated && (onOld || onNew) {
			matching = append(matching, comment)
		}
	}
	return matching
}

func fragmentLines(file *gitdiff.File, fragment *gitdiff.TextFragment, comments []store.Comment) []diffLine {
	header := fmt.Sprintf(
		"@@ -%d,%d +%d,%d @@ %s",
		fragment.OldPosition, fragment.OldLines, fragment.NewPosition, fragment.NewLines, fragment.Comment,
	)
	lines := []diffLine{{Kind: "hunk", Text: header}}
	oldNumber, newNumber := fragment.OldPosition, fragment.NewPosition
	for _, line := range fragment.Lines {
		shown := diffLine{Kind: lineKinds[line.Op], Text: strings.TrimSuffix(line.Line, "\n")}
		if line.Old() {
			shown.Old = oldNumber
			oldNumber++
		}
		if line.New() {
			shown.New = newNumber
			newNumber++
		}
		if matching := commentsOn(comments, file, shown); len(matching) > 0 {
			shown.Comments = matching
		}
		lines = append(lines, shown)
	}
	return lines
}

func diffView(files []*gitdiff.File, comments []store.Comment) []diffFile {
	view := []diffFile{}
	for _, file := range files {
		shown := diffFile{Path: file.NewName, Binary: file.IsBinary}
		if file.IsDelete {
			shown.Path = file.OldName
		}
		if file.IsRename {
			shown.OldPath = file.OldName
		}
		for _, fragment := range file.TextFragments {
			shown.Lines = append(shown.Lines, fragmentLines(file, fragment, comments)...)
		}
		view = append(view, shown)
	}
	return view
}

func (s *Server) uiCommentItem(kind string) handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		repo, item, err := s.uiItem(r, kind, true)
		if err != nil {
			return err
		}
		body := r.FormValue("body")
		if strings.TrimSpace(body) == "" {
			return badRequest("Comment body is empty.")
		}
		if err := checkText(nil, &body); err != nil {
			return err
		}
		lineNumber, _ := strconv.ParseInt(r.FormValue("line"), 10, 64)
		anchor := store.LineRef{Path: r.FormValue("path"), Line: lineNumber, Side: r.FormValue("side")}
		line, err := s.commentLine(ctx, repo.Name, kind, item.Number, anchor)
		if err != nil {
			return missing(err, "Not found.")
		}
		actor := identityFrom(r).actor
		comment, _, err := s.store.AddComment(ctx, kind, repo.Name, item.Number, actor, body, line)
		if err != nil {
			return missing(err, "Not found.")
		}
		target := itemURL(repo.Name, kind, item.Number)
		if line.Path != "" {
			target += fmt.Sprintf("?tab=files#comment-%d", comment.ID)
		}
		http.Redirect(w, r, target, http.StatusSeeOther)
		return nil
	}
}

func itemEditFromForm(r *http.Request, item store.Item) store.ItemEdit {
	id := identityFrom(r)
	edit := store.ItemEdit{}
	if r.PostForm.Has("state") {
		state := r.PostForm.Get("state")
		edit.State = &state
	}
	if r.PostForm.Has("labels") {
		wanted := splitList(r.PostForm.Get("labels"))
		edit.AddLabels = difference(wanted, item.Labels)
		edit.RemoveLabels = difference(item.Labels, wanted)
	}
	if r.PostForm.Has("assignees") {
		wanted := id.resolve(splitList(r.PostForm.Get("assignees")))
		edit.AddAssignees = difference(wanted, item.Assignees)
		edit.RemoveAssignees = difference(item.Assignees, wanted)
	}
	return edit
}

func (s *Server) uiEditItem(kind string) handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		repo, item, err := s.uiItem(r, kind, true)
		if err != nil {
			return err
		}
		if err := r.ParseForm(); err != nil {
			return badRequest(err.Error())
		}
		edit := itemEditFromForm(r, item)
		if kind == kindPull {
			if _, err := s.checkReopen(ctx, repo.Name, item.Number, edit.State); err != nil {
				return err
			}
		}
		actor := identityFrom(r).actor
		if _, _, err := s.store.EditItem(ctx, kind, repo.Name, item.Number, actor, edit); err != nil {
			return missing(err, "Not found.")
		}
		if kind == kindPull {
			s.queueReopened(ctx, repo.Name, item.Number, item.State == stateOpen, actor)
		}
		http.Redirect(w, r, itemURL(repo.Name, kind, item.Number), http.StatusSeeOther)
		return nil
	}
}
