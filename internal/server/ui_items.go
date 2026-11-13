package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

const (
	tabCommits = "commits"
	tabFiles   = "files"

	lineAdd    = "add"
	lineDelete = "delete"
)

var itemPaths = map[string]string{kindIssue: "issues", kindPull: "pulls"}

var itemTitles = map[string]string{kindIssue: "Issues · ", kindPull: "Pull requests · "}

var lineKinds = map[gitdiff.LineOp]string{
	gitdiff.OpContext: "context",
	gitdiff.OpAdd:     lineAdd,
	gitdiff.OpDelete:  lineDelete,
}

type itemRow struct {
	Item store.Item
	Pull *store.Pull
}

type itemsView struct {
	Kind     string
	Path     string
	Noun     string
	Rows     []itemRow
	State    string
	CanWrite bool
}

type timelineEntry struct {
	At      time.Time
	Comment *store.Comment
	Review  *store.Review
}

type diffLine struct {
	Kind     string
	Old      int64
	New      int64
	Text     string
	Comments []store.Comment
}

type diffFile struct {
	Path    string
	OldPath string
	Binary  bool
	Lines   []diffLine
}

type itemPage struct {
	Kind       string
	Path       string
	Noun       string
	Item       store.Item
	Pull       *store.Pull
	Timeline   []timelineEntry
	CanWrite   bool
	IsAuthor   bool
	Labels     string
	Assignees  string
	Tab        string
	Commits    []repos.Commit
	Files      []repos.FileChange
	DiffFiles  []diffFile
	Merge      *mergePlan
	MergeError string
	Checks     *store.Checks
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

func (s *Server) itemRows(ctx context.Context, kind, repo, state string) ([]itemRow, error) {
	rows := []itemRow{}
	if kind == kindIssue {
		items, err := s.store.Items(ctx, kind, repo, state)
		for _, item := range items {
			rows = append(rows, itemRow{Item: item})
		}
		return rows, err
	}
	pulls, err := s.store.Pulls(ctx, repo, stateAll)
	for _, pull := range pulls {
		if (pull.State == stateOpen) == (state == stateOpen) {
			rows = append(rows, itemRow{Item: pull.Item, Pull: &pull})
		}
	}
	return rows, err
}

func (s *Server) uiItems(kind string) handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		repo, err := s.uiLookup(r)
		if err != nil {
			return err
		}
		state := r.URL.Query().Get("state")
		if state != stateClosed {
			state = stateOpen
		}
		rows, err := s.itemRows(ctx, kind, repo.Name, state)
		if err != nil {
			return err
		}
		view := itemsView{
			Kind:     kind,
			Path:     itemPaths[kind],
			Noun:     nouns[kind],
			Rows:     rows,
			State:    state,
			CanWrite: identityFrom(r).can(scopeWrite, repo.Name),
		}
		ref := s.git.DefaultBranch(ctx, s.repoDir(repo.Name))
		s.render(w, r, http.StatusOK, "items", s.repoPage(r, &repo, itemTitles[kind]+repo.Name, ref, view))
		return nil
	}
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

func (s *Server) pullTab(
	ctx context.Context,
	repo string,
	pull store.Pull,
	comments []store.Comment,
	view *itemPage,
) error {
	base, head := s.pullRange(ctx, repo, pull)
	var err error
	switch view.Tab {
	case tabCommits:
		view.Commits, err = s.commits(ctx, repo, head, pullCommitLimit, base)
	case tabFiles:
		if view.Files, err = s.git.DiffFiles(ctx, s.repoDir(repo), base, head); err != nil {
			return err
		}
		var files []*gitdiff.File
		if files, err = s.pullDiff(ctx, repo, pull); err == nil {
			view.DiffFiles = diffView(files, comments)
		}
	}
	return err
}

func (s *Server) pullDetails(
	r *http.Request,
	repo string,
	item store.Item,
	comments []store.Comment,
	view *itemPage,
) ([]store.Review, error) {
	ctx := r.Context()
	pull, err := s.store.Pull(ctx, repo, item.Number)
	if err != nil {
		return nil, err
	}
	reviews, err := s.store.Reviews(ctx, item)
	if err != nil {
		return nil, err
	}
	view.Pull = &pull
	view.IsAuthor = isAuthor(pull, identityFrom(r).actor)
	view.Merge, view.MergeError = s.mergeStatus(ctx, repo, pull)
	checks, err := s.store.Checks(ctx, repo, pull.HeadSHA)
	if err != nil {
		return nil, err
	}
	view.Checks = &checks
	s.markOutdated(ctx, repo, pull, comments)
	for index := range reviews {
		s.markOutdated(ctx, repo, pull, reviews[index].Comments)
	}
	return reviews, s.pullTab(ctx, repo, pull, comments, view)
}

func timeline(comments []store.Comment, reviews []store.Review) []timelineEntry {
	entries := []timelineEntry{}
	for index := range comments {
		if comments[index].ReviewID == 0 {
			entries = append(entries, timelineEntry{At: comments[index].CreatedAt, Comment: &comments[index]})
		}
	}
	for index := range reviews {
		entries = append(entries, timelineEntry{At: reviews[index].CreatedAt, Review: &reviews[index]})
	}
	slices.SortStableFunc(entries, func(first, second timelineEntry) int {
		return first.At.Compare(second.At)
	})
	return entries
}

func (s *Server) uiItemPage(kind string) handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		repo, item, err := s.uiItem(r, kind, false)
		if err != nil {
			return err
		}
		view := itemPage{
			Kind:      kind,
			Path:      itemPaths[kind],
			Noun:      nouns[kind],
			Item:      item,
			CanWrite:  identityFrom(r).can(scopeWrite, repo.Name),
			Labels:    strings.Join(item.Labels, ", "),
			Assignees: strings.Join(item.Assignees, ", "),
			Tab:       r.URL.Query().Get("tab"),
		}
		if view.Tab != tabCommits && view.Tab != tabFiles {
			view.Tab = ""
		}
		comments, err := s.store.Comments(ctx, item)
		if err != nil {
			return err
		}
		reviews := []store.Review{}
		if kind == kindPull {
			if reviews, err = s.pullDetails(r, repo.Name, item, comments, &view); err != nil {
				return err
			}
		}
		view.Timeline = timeline(comments, reviews)
		title := fmt.Sprintf("%s · #%d · %s", item.Title, item.Number, repo.Name)
		ref := s.git.DefaultBranch(ctx, s.repoDir(repo.Name))
		s.render(w, r, http.StatusOK, "item", s.repoPage(r, &repo, title, ref, view))
		return nil
	}
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
