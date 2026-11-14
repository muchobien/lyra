package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/muchobien/lyra/internal/store"
)

const (
	sideOld = "old"
	sideNew = "new"

	reviewApprove        = "approve"
	reviewRequestChanges = "request_changes"
	reviewComment        = "comment"

	reviewApproved         = "approved"
	reviewChangesRequested = "changes_requested"
	reviewCommented        = "commented"
)

var reviewStates = map[string]string{
	reviewApprove:        reviewApproved,
	reviewRequestChanges: reviewChangesRequested,
	reviewComment:        reviewCommented,
}

type reviewRequest struct {
	State    string              `json:"state" enum:"approve,request_changes,comment"`
	Body     string              `json:"body,omitzero"`
	HeadSHA  string              `json:"head_sha,omitzero" doc:"Reject a review of a changed head."`
	Comments []store.LineComment `json:"comments,omitzero" doc:"Line comments on lines in the diff."`
}

type reviewInput struct {
	ItemPath
	Body reviewRequest
}

type reviewWrite struct {
	store.Review
	EventID int64 `json:"event_id"`
}

func (s *Server) pullDiff(ctx context.Context, repo string, pull store.Pull) ([]*gitdiff.File, error) {
	base, head := s.pullRange(ctx, repo, pull)
	diff, err := s.git.Diff(ctx, s.repoDir(repo), base, head)
	if err != nil {
		return nil, err
	}
	files, _, err := gitdiff.Parse(strings.NewReader(diff))
	return files, err
}

func fragmentHas(fragment *gitdiff.TextFragment, line store.LineRef) bool {
	oldNumber, newNumber := fragment.OldPosition, fragment.NewPosition
	for _, diffLine := range fragment.Lines {
		if line.Side == sideOld && diffLine.Old() && oldNumber == line.Line {
			return true
		}
		if line.Side == sideNew && diffLine.New() && newNumber == line.Line {
			return true
		}
		if diffLine.Old() {
			oldNumber++
		}
		if diffLine.New() {
			newNumber++
		}
	}
	return false
}

func anchored(files []*gitdiff.File, line store.LineRef) bool {
	for _, file := range files {
		if line.Side == sideNew && file.NewName != line.Path {
			continue
		}
		if line.Side == sideOld && file.OldName != line.Path {
			continue
		}
		for _, fragment := range file.TextFragments {
			if fragmentHas(fragment, line) {
				return true
			}
		}
	}
	return false
}

func (s *Server) checkLines(ctx context.Context, repo string, pull store.Pull, lines []*store.LineRef) error {
	if len(lines) == 0 {
		return nil
	}
	files, err := s.pullDiff(ctx, repo, pull)
	if err != nil {
		return err
	}
	for _, line := range lines {
		if line.Side == "" {
			line.Side = sideNew
		}
		if line.Side != sideOld && line.Side != sideNew {
			return store.InvalidError("side must be old or new")
		}
		if !anchored(files, *line) {
			return store.InvalidError(fmt.Sprintf(
				"line %d (%s side) of %s is not in the diff of #%d",
				line.Line, line.Side, line.Path, pull.Number,
			))
		}
		line.CommitSHA = pull.HeadSHA
	}
	return nil
}

func (s *Server) commentLine(
	ctx context.Context,
	repo, kind string,
	number int64,
	ref store.LineRef,
) (store.LineRef, error) {
	if ref.Path == "" && ref.Line == 0 && ref.Side == "" {
		return ref, nil
	}
	if kind != kindPull {
		return ref, store.InvalidError("only pull request comments can point at a line")
	}
	if ref.Path == "" || ref.Line <= 0 {
		return ref, store.InvalidError("a line comment needs a path and a line number")
	}
	pull, err := s.store.Pull(ctx, repo, number)
	if err != nil {
		return ref, err
	}
	if ref.CommitSHA != "" && ref.CommitSHA != pull.HeadSHA {
		return ref, conflict("the pull request changed; refresh the diff before commenting on this line")
	}
	err = s.checkLines(ctx, repo, pull, []*store.LineRef{&ref})
	return ref, err
}

func (s *Server) markOutdated(ctx context.Context, repo string, pull store.Pull, comments []store.Comment) {
	for index := range comments {
		comment := &comments[index]
		if comment.Path == "" || comment.Side != sideNew || comment.CommitSHA == pull.HeadSHA {
			continue
		}
		changed, err := s.git.Changed(ctx, s.repoDir(repo), comment.CommitSHA, pull.HeadSHA, comment.Path)
		comment.Outdated = err != nil || changed
	}
}

func isAuthor(pull store.Pull, actor store.Actor) bool {
	return pull.Author.User == actor.User && pull.Author.Agent == actor.Agent
}

func latestReviews(pull store.Pull, reviews []store.Review) []store.Review {
	latest := map[string]store.Review{}
	order := []string{}
	for _, review := range reviews {
		if review.State == reviewCommented || isAuthor(pull, review.Author) {
			continue
		}
		reviewer := review.Author.User + " " + review.Author.Agent
		if _, seen := latest[reviewer]; !seen {
			order = append(order, reviewer)
		}
		latest[reviewer] = review
	}
	ordered := []store.Review{}
	for _, reviewer := range order {
		ordered = append(ordered, latest[reviewer])
	}
	return ordered
}

func approvalProblems(pull store.Pull, reviews []store.Review, head string) []string {
	problems := []string{}
	approved := false
	for _, review := range latestReviews(pull, reviews) {
		name := review.Author.User
		if review.Author.Agent != "" {
			name = review.Author.Agent + " for " + review.Author.User
		}
		if review.State == reviewChangesRequested {
			problems = append(problems, name+" requested changes")
		}
		approved = approved || (review.State == reviewApproved && review.CommitSHA == head)
	}
	if !approved {
		problems = append(problems, fmt.Sprintf(
			"needs an approving review of %s by someone other than the author",
			head[:7],
		))
	}
	return problems
}

func (s *Server) handleReviewPull(r *http.Request, input *reviewInput) (*body[reviewWrite], error) {
	repo, err := s.lookupRepo(r, scopeWrite, input.fullName())
	if err != nil {
		return nil, err
	}
	review, eventID, err := s.reviewPull(r, repo.Name, input.Number, input.Body)
	if err != nil {
		return nil, missing(err, "pull request not found")
	}
	return reply(reviewWrite{Review: review, EventID: eventID})
}

func checkReview(request reviewRequest) error {
	if _, ok := reviewStates[request.State]; !ok {
		return store.InvalidError("state must be approve, request_changes or comment")
	}
	if err := checkText(nil, &request.Body); err != nil {
		return err
	}
	empty := strings.TrimSpace(request.Body) == ""
	if request.State == reviewRequestChanges && empty {
		return store.InvalidError("requesting changes needs a body")
	}
	if request.State == reviewComment && empty && len(request.Comments) == 0 {
		return store.InvalidError("a comment review needs a body or line comments")
	}
	return nil
}

func (s *Server) reviewPull(
	r *http.Request,
	repo string,
	number int64,
	request reviewRequest,
) (store.Review, int64, error) {
	ctx := r.Context()
	id := identityFrom(r)
	if err := checkReview(request); err != nil {
		return store.Review{}, 0, err
	}
	pull, err := s.store.Pull(ctx, repo, number)
	if err != nil {
		return store.Review{}, 0, err
	}
	if request.HeadSHA != "" && request.HeadSHA != pull.HeadSHA {
		return store.Review{}, 0, conflict("the pull request changed; refresh the diff before submitting this review")
	}
	if request.State != reviewComment && isAuthor(pull, id.actor) {
		return store.Review{}, 0, store.InvalidError("the author of a pull request can't approve it or request changes")
	}
	lines := []*store.LineRef{}
	for index := range request.Comments {
		if strings.TrimSpace(request.Comments[index].Body) == "" {
			return store.Review{}, 0, store.InvalidError("line comments need a body")
		}
		lines = append(lines, &request.Comments[index].LineRef)
	}
	if err := s.checkLines(ctx, repo, pull, lines); err != nil {
		return store.Review{}, 0, err
	}
	return s.store.CreateReview(
		ctx,
		repo,
		number,
		id.actor,
		reviewStates[request.State],
		request.Body,
		pull.HeadSHA,
		request.Comments,
	)
}
