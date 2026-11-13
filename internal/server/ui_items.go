package server

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

var itemPaths = map[string]string{"issue": "issues", "pr": "pulls"}

var lineKinds = map[gitdiff.LineOp]string{gitdiff.OpContext: "context", gitdiff.OpAdd: "add", gitdiff.OpDelete: "delete"}

type itemRow struct {
	Item store.Item
	Pull *store.Pull
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
	for _, value := range strings.Split(text, ",") {
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

func (s *Server) renderStoreError(w http.ResponseWriter, r *http.Request, err error, missing string) {
	var invalid store.InvalidError
	switch {
	case errors.Is(err, store.ErrNotFound):
		s.renderError(w, r, http.StatusNotFound, missing)
	case errors.As(err, &invalid):
		s.renderError(w, r, http.StatusBadRequest, invalid.Error())
	default:
		s.renderError(w, r, http.StatusInternalServerError, err.Error())
	}
}

func (s *Server) uiItem(w http.ResponseWriter, r *http.Request, kind string, write bool) (store.Repo, store.Item, bool) {
	repo, ok := s.uiLookup(w, r)
	if !ok {
		return repo, store.Item{}, false
	}
	if write && !identityFrom(r).can("write", repo.Name) {
		s.renderError(w, r, http.StatusForbidden, "This token cannot write to "+repo.Name+".")
		return repo, store.Item{}, false
	}
	missing := strings.ToUpper(nouns[kind][:1]) + nouns[kind][1:] + " not found."
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, missing)
		return repo, store.Item{}, false
	}
	item, err := s.store.Item(kind, repo.Name, number)
	if err != nil {
		s.renderStoreError(w, r, err, missing)
		return repo, item, false
	}
	return repo, item, true
}

func (s *Server) uiItems(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, ok := s.uiLookup(w, r)
		if !ok {
			return
		}
		state := r.URL.Query().Get("state")
		if state != "closed" {
			state = "open"
		}
		rows := []itemRow{}
		if kind == "issue" {
			items, err := s.store.Items(kind, repo.Name, state)
			if err != nil {
				s.renderError(w, r, http.StatusInternalServerError, err.Error())
				return
			}
			for _, item := range items {
				rows = append(rows, itemRow{Item: item})
			}
		} else {
			pulls, err := s.store.Pulls(repo.Name, "all")
			if err != nil {
				s.renderError(w, r, http.StatusInternalServerError, err.Error())
				return
			}
			for _, pull := range pulls {
				if (pull.State == "open") == (state == "open") {
					rows = append(rows, itemRow{Item: pull.Item, Pull: &pull})
				}
			}
		}
		view := struct {
			Kind     string
			Path     string
			Noun     string
			Rows     []itemRow
			State    string
			CanWrite bool
		}{kind, itemPaths[kind], nouns[kind], rows, state, identityFrom(r).can("write", repo.Name)}
		title := "Issues · "
		if kind == "pr" {
			title = "Pull requests · "
		}
		s.render(w, r, http.StatusOK, "items", page{Title: title + repo.Name, Repo: &repo, Ref: s.git.DefaultBranch(s.repoDir(repo.Name)), CloneURL: cloneURL(r, repo.Name), Data: view})
	}
}

func (s *Server) uiCreateIssue(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.uiLookup(w, r)
	if !ok {
		return
	}
	id := identityFrom(r)
	if !id.can("write", repo.Name) {
		s.renderError(w, r, http.StatusForbidden, "This token cannot write to "+repo.Name+".")
		return
	}
	title, body := r.FormValue("title"), r.FormValue("body")
	if problem := checkText(&title, &body); problem != "" {
		s.renderError(w, r, http.StatusBadRequest, problem)
		return
	}
	item, _, err := s.store.CreateItem("issue", repo.Name, id.actor, title, body, nil, nil)
	if err != nil {
		s.renderStoreError(w, r, err, "Repository not found.")
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/%s/issues/%d", repo.Name, item.Number), http.StatusSeeOther)
}

func (s *Server) uiItemPage(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, item, ok := s.uiItem(w, r, kind, false)
		if !ok {
			return
		}
		view := itemPage{
			Kind: kind, Path: itemPaths[kind], Noun: nouns[kind], Item: item, CanWrite: identityFrom(r).can("write", repo.Name),
			Labels: strings.Join(item.Labels, ", "), Assignees: strings.Join(item.Assignees, ", "), Tab: r.URL.Query().Get("tab"),
		}
		if view.Tab != "commits" && view.Tab != "files" {
			view.Tab = ""
		}
		comments, err := s.store.Comments(item)
		if err != nil {
			s.renderError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		reviews := []store.Review{}
		if kind == "pr" {
			pull, err := s.store.Pull(repo.Name, item.Number)
			if err == nil {
				reviews, err = s.store.Reviews(item)
			}
			if err != nil {
				s.renderError(w, r, http.StatusInternalServerError, err.Error())
				return
			}
			view.Pull = &pull
			view.IsAuthor = isAuthor(pull, identityFrom(r).actor)
			view.Merge, view.MergeError = s.mergeStatus(repo.Name, pull)
			checks, err := s.store.Checks(repo.Name, pull.HeadSHA)
			if err != nil {
				s.renderError(w, r, http.StatusInternalServerError, err.Error())
				return
			}
			view.Checks = &checks
			s.markOutdated(repo.Name, pull, comments)
			for index := range reviews {
				s.markOutdated(repo.Name, pull, reviews[index].Comments)
			}
			base, head := s.pullRange(repo.Name, pull)
			switch view.Tab {
			case "commits":
				view.Commits, err = s.commits(repo.Name, head, 250, base)
			case "files":
				view.Files, err = s.git.DiffFiles(s.repoDir(repo.Name), base, head)
				if err != nil {
					break
				}
				files, err := s.pullDiff(repo.Name, pull)
				if err != nil {
					s.renderError(w, r, http.StatusInternalServerError, err.Error())
					return
				}
				for _, file := range files {
					shown := diffFile{Path: file.NewName, Binary: file.IsBinary}
					if file.IsDelete {
						shown.Path = file.OldName
					}
					if file.IsRename {
						shown.OldPath = file.OldName
					}
					for _, fragment := range file.TextFragments {
						shown.Lines = append(shown.Lines, diffLine{Kind: "hunk", Text: fmt.Sprintf("@@ -%d,%d +%d,%d @@ %s", fragment.OldPosition, fragment.OldLines, fragment.NewPosition, fragment.NewLines, fragment.Comment)})
						oldNumber, newNumber := fragment.OldPosition, fragment.NewPosition
						for _, line := range fragment.Lines {
							shown.Lines = append(shown.Lines, diffLine{Kind: lineKinds[line.Op], Text: strings.TrimSuffix(line.Line, "\n")})
							current := &shown.Lines[len(shown.Lines)-1]
							if line.Old() {
								current.Old = oldNumber
								oldNumber++
							}
							if line.New() {
								current.New = newNumber
								newNumber++
							}
							for _, comment := range comments {
								onOld := comment.Side == "old" && current.Old == comment.Line && comment.Path == file.OldName && current.Kind != "add"
								onNew := comment.Side == "new" && current.New == comment.Line && comment.Path == file.NewName && current.Kind != "delete"
								if !comment.Outdated && (onOld || onNew) {
									current.Comments = append(current.Comments, comment)
								}
							}
						}
					}
					view.DiffFiles = append(view.DiffFiles, shown)
				}
			}
			if err != nil {
				s.renderError(w, r, http.StatusInternalServerError, err.Error())
				return
			}
		}
		for index := range comments {
			if comments[index].ReviewID == 0 {
				view.Timeline = append(view.Timeline, timelineEntry{At: comments[index].CreatedAt, Comment: &comments[index]})
			}
		}
		for index := range reviews {
			view.Timeline = append(view.Timeline, timelineEntry{At: reviews[index].CreatedAt, Review: &reviews[index]})
		}
		sort.SliceStable(view.Timeline, func(first, second int) bool { return view.Timeline[first].At.Before(view.Timeline[second].At) })
		s.render(w, r, http.StatusOK, "item", page{Title: fmt.Sprintf("%s · #%d · %s", item.Title, item.Number, repo.Name), Repo: &repo, Ref: s.git.DefaultBranch(s.repoDir(repo.Name)), CloneURL: cloneURL(r, repo.Name), Data: view})
	}
}

func (s *Server) uiCommentItem(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, item, ok := s.uiItem(w, r, kind, true)
		if !ok {
			return
		}
		body := r.FormValue("body")
		if strings.TrimSpace(body) == "" {
			s.renderError(w, r, http.StatusBadRequest, "Comment body is empty.")
			return
		}
		if problem := checkText(nil, &body); problem != "" {
			s.renderError(w, r, http.StatusBadRequest, problem)
			return
		}
		lineNumber, _ := strconv.ParseInt(r.FormValue("line"), 10, 64)
		line, err := s.commentLine(repo.Name, kind, item.Number, r.FormValue("path"), lineNumber, r.FormValue("side"))
		if err != nil {
			s.renderStoreError(w, r, err, "Not found.")
			return
		}
		comment, _, err := s.store.AddComment(kind, repo.Name, item.Number, identityFrom(r).actor, body, line)
		if err != nil {
			s.renderStoreError(w, r, err, "Not found.")
			return
		}
		target := fmt.Sprintf("/%s/%s/%d", repo.Name, itemPaths[kind], item.Number)
		if line.Path != "" {
			target += fmt.Sprintf("?tab=files#comment-%d", comment.ID)
		}
		http.Redirect(w, r, target, http.StatusSeeOther)
	}
}

func (s *Server) uiEditItem(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, item, ok := s.uiItem(w, r, kind, true)
		if !ok {
			return
		}
		if err := r.ParseForm(); err != nil {
			s.renderError(w, r, http.StatusBadRequest, err.Error())
			return
		}
		id := identityFrom(r)
		edit := store.ItemEdit{}
		if r.PostForm.Has("state") {
			state := r.PostForm.Get("state")
			edit.State = &state
		}
		if r.PostForm.Has("labels") {
			wanted := splitList(r.PostForm.Get("labels"))
			edit.AddLabels, edit.RemoveLabels = difference(wanted, item.Labels), difference(item.Labels, wanted)
		}
		if r.PostForm.Has("assignees") {
			wanted := id.resolve(splitList(r.PostForm.Get("assignees")))
			edit.AddAssignees, edit.RemoveAssignees = difference(wanted, item.Assignees), difference(item.Assignees, wanted)
		}
		if _, _, err := s.store.EditItem(kind, repo.Name, item.Number, id.actor, edit); err != nil {
			s.renderStoreError(w, r, err, "Not found.")
			return
		}
		if kind == "pr" {
			s.queueReopened(repo.Name, item.Number, item.State == "open", id.actor)
		}
		http.Redirect(w, r, fmt.Sprintf("/%s/%s/%d", repo.Name, itemPaths[kind], item.Number), http.StatusSeeOther)
	}
}
