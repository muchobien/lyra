package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/muchobien/lyra/internal/store"
)

var reviewStates = map[string]string{"approve": "approved", "request_changes": "changes_requested", "comment": "commented"}

func (s *Server) pullDiff(repo string, pull store.Pull) ([]*gitdiff.File, error) {
	base, head := s.pullRange(repo, pull)
	diff, err := s.git.Diff(s.repoDir(repo), base, head)
	if err != nil {
		return nil, err
	}
	files, _, err := gitdiff.Parse(strings.NewReader(diff))
	return files, err
}

func anchored(files []*gitdiff.File, line store.LineRef) bool {
	for _, file := range files {
		if (line.Side == "new" && file.NewName != line.Path) || (line.Side == "old" && file.OldName != line.Path) {
			continue
		}
		for _, fragment := range file.TextFragments {
			oldNumber, newNumber := fragment.OldPosition, fragment.NewPosition
			for _, diffLine := range fragment.Lines {
				if line.Side == "old" && diffLine.Old() && oldNumber == line.Line {
					return true
				}
				if line.Side == "new" && diffLine.New() && newNumber == line.Line {
					return true
				}
				if diffLine.Old() {
					oldNumber++
				}
				if diffLine.New() {
					newNumber++
				}
			}
		}
	}
	return false
}

func (s *Server) checkLines(repo string, pull store.Pull, lines []*store.LineRef) error {
	if len(lines) == 0 {
		return nil
	}
	files, err := s.pullDiff(repo, pull)
	if err != nil {
		return err
	}
	for _, line := range lines {
		if line.Side == "" {
			line.Side = "new"
		}
		if line.Side != "old" && line.Side != "new" {
			return store.InvalidError("side must be old or new")
		}
		if !anchored(files, *line) {
			return store.InvalidError(fmt.Sprintf("line %d (%s side) of %s is not in the diff of #%d", line.Line, line.Side, line.Path, pull.Number))
		}
		line.CommitSHA = pull.HeadSHA
	}
	return nil
}

func (s *Server) commentLine(repo, kind string, number int64, path string, line int64, side string) (store.LineRef, error) {
	ref := store.LineRef{Path: path, Line: line, Side: side}
	if path == "" && line == 0 && side == "" {
		return ref, nil
	}
	if kind != "pr" {
		return ref, store.InvalidError("only pull request comments can point at a line")
	}
	if path == "" || line <= 0 {
		return ref, store.InvalidError("a line comment needs a path and a line number")
	}
	pull, err := s.store.Pull(repo, number)
	if err != nil {
		return ref, err
	}
	err = s.checkLines(repo, pull, []*store.LineRef{&ref})
	return ref, err
}

func (s *Server) markOutdated(repo string, pull store.Pull, comments []store.Comment) {
	for index := range comments {
		comment := &comments[index]
		if comment.Path == "" || comment.Side != "new" || comment.CommitSHA == pull.HeadSHA {
			continue
		}
		changed, err := s.git.Changed(s.repoDir(repo), comment.CommitSHA, pull.HeadSHA, comment.Path)
		comment.Outdated = err != nil || changed
	}
}

func isAuthor(pull store.Pull, actor store.Actor) bool {
	return pull.Author.User == actor.User && pull.Author.Agent == actor.Agent
}

func approvalProblems(pull store.Pull, reviews []store.Review, head string) []string {
	latest := map[string]store.Review{}
	order := []string{}
	for _, review := range reviews {
		if review.State == "commented" || isAuthor(pull, review.Author) {
			continue
		}
		reviewer := review.Author.User + " " + review.Author.Agent
		if _, seen := latest[reviewer]; !seen {
			order = append(order, reviewer)
		}
		latest[reviewer] = review
	}
	problems := []string{}
	approved := false
	for _, reviewer := range order {
		review := latest[reviewer]
		name := review.Author.User
		if review.Author.Agent != "" {
			name = review.Author.Agent + " for " + review.Author.User
		}
		if review.State == "changes_requested" {
			problems = append(problems, name+" requested changes")
		}
		approved = approved || (review.State == "approved" && review.CommitSHA == head)
	}
	if !approved {
		problems = append(problems, fmt.Sprintf("needs an approving review of %s by someone other than the author", head[:7]))
	}
	return problems
}

func (s *Server) handleReviewPull(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "write")
	if !ok {
		return
	}
	number, ok := itemNumber(w, r)
	if !ok {
		return
	}
	var body struct {
		State    string              `json:"state"`
		Body     string              `json:"body"`
		Comments []store.LineComment `json:"comments"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	review, eventID, err := s.reviewPull(r, repo.Name, number, body.State, body.Body, body.Comments)
	if err != nil {
		writeStoreError(w, err, "pull request not found")
		return
	}
	writeJSON(w, http.StatusCreated, struct {
		store.Review
		EventID int64 `json:"event_id"`
	}{review, eventID})
}

func (s *Server) reviewPull(r *http.Request, repo string, number int64, state, body string, comments []store.LineComment) (store.Review, int64, error) {
	id := identityFrom(r)
	stored, ok := reviewStates[state]
	if !ok {
		return store.Review{}, 0, store.InvalidError("state must be approve, request_changes or comment")
	}
	if problem := checkText(nil, &body); problem != "" {
		return store.Review{}, 0, store.InvalidError(problem)
	}
	if state == "request_changes" && strings.TrimSpace(body) == "" {
		return store.Review{}, 0, store.InvalidError("requesting changes needs a body")
	}
	if state == "comment" && strings.TrimSpace(body) == "" && len(comments) == 0 {
		return store.Review{}, 0, store.InvalidError("a comment review needs a body or line comments")
	}
	pull, err := s.store.Pull(repo, number)
	if err != nil {
		return store.Review{}, 0, err
	}
	if state != "comment" && isAuthor(pull, id.actor) {
		return store.Review{}, 0, store.InvalidError("the author of a pull request can't approve it or request changes")
	}
	lines := []*store.LineRef{}
	for index := range comments {
		if strings.TrimSpace(comments[index].Body) == "" {
			return store.Review{}, 0, store.InvalidError("line comments need a body")
		}
		lines = append(lines, &comments[index].LineRef)
	}
	if err := s.checkLines(repo, pull, lines); err != nil {
		return store.Review{}, 0, err
	}
	return s.store.CreateReview(repo, number, id.actor, stored, body, pull.HeadSHA, comments)
}

func (s *Server) uiReviewPull(w http.ResponseWriter, r *http.Request) {
	repo, item, ok := s.uiItem(w, r, "pr", true)
	if !ok {
		return
	}
	if _, _, err := s.reviewPull(r, repo.Name, item.Number, r.FormValue("state"), r.FormValue("body"), nil); err != nil {
		s.renderStoreError(w, r, err, "Pull request not found.")
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/%s/pulls/%d", repo.Name, item.Number), http.StatusSeeOther)
}
