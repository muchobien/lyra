package server

import (
	"fmt"
	"net/http"
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

func (s *Server) uiItem(r *http.Request, kind string) (store.Repo, store.Item, error) {
	repo, err := s.uiLookup(r)
	if err != nil {
		return repo, store.Item{}, err
	}
	absent := strings.ToUpper(nouns[kind][:1]) + nouns[kind][1:] + " not found."
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil {
		return repo, store.Item{}, notFound(absent)
	}
	item, err := s.store.Item(r.Context(), kind, repo.Name, number)
	return repo, item, missing(err, absent)
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
