package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/muchobien/lyra/internal/store"
)

type notePull struct {
	Number int64  `json:"number"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

type releaseNotes struct {
	Notes       string     `json:"notes"`
	PreviousTag string     `json:"previous_tag"`
	Pulls       []notePull `json:"pulls"`
}

func (s *Server) previousRelease(
	ctx context.Context,
	repo, tag, commit string,
	prerelease bool,
) (store.Release, bool, error) {
	releases, err := s.store.Releases(ctx, repo)
	if err != nil {
		return store.Release{}, false, err
	}
	for _, release := range releases {
		if release.Draft || release.Tag == tag || release.Commit == "" || (release.Prerelease && !prerelease) {
			continue
		}
		ancestor, err := s.git.IsAncestor(ctx, s.repoDir(repo), release.Commit, commit)
		if err == nil && ancestor {
			return release, true, nil
		}
	}
	return store.Release{}, false, nil
}

func (s *Server) mergedPulls(ctx context.Context, repo string, commits []string) ([]notePull, error) {
	pulls, err := s.store.Pulls(ctx, repo, stateMerged)
	if err != nil {
		return nil, err
	}
	merged := []notePull{}
	for _, pull := range slices.Backward(pulls) {
		if !slices.Contains(commits, pull.MergeSHA) {
			continue
		}
		author := pull.Author.Agent
		if author == "" {
			author = pull.Author.User
		}
		merged = append(merged, notePull{Number: pull.Number, Title: pull.Title, Author: author})
	}
	return merged, nil
}

func notesText(pulls []notePull, previousTag string) string {
	since := ""
	if previousTag != "" {
		since = " since " + previousTag
	}
	if len(pulls) == 0 {
		return "No pull requests merged" + since + "."
	}
	var text strings.Builder
	text.WriteString("## Pull requests merged" + since + "\n\n")
	for _, pull := range pulls {
		fmt.Fprintf(&text, "- %s (#%d) by %s\n", pull.Title, pull.Number, pull.Author)
	}
	return strings.TrimSuffix(text.String(), "\n")
}

func (s *Server) generateNotes(ctx context.Context, repo, tag, commit string, prerelease bool) (releaseNotes, error) {
	notes := releaseNotes{Pulls: []notePull{}}
	previous, found, err := s.previousRelease(ctx, repo, tag, commit, prerelease)
	if err != nil {
		return releaseNotes{}, err
	}
	exclude := []string{}
	if found {
		notes.PreviousTag = previous.Tag
		exclude = append(exclude, previous.Commit)
	}
	commits, err := s.git.RevList(ctx, s.repoDir(repo), commit, exclude...)
	if err != nil {
		return notes, err
	}
	if notes.Pulls, err = s.mergedPulls(ctx, repo, commits); err != nil {
		return notes, err
	}
	notes.Notes = notesText(notes.Pulls, notes.PreviousTag)
	return notes, nil
}

type releaseNotesInput struct {
	RepoPath
	Tag        string `query:"tag"`
	Target     string `query:"target" doc:"Used when the tag doesn't exist; defaults to the default branch."`
	Prerelease bool   `query:"prerelease"`
}

func (s *Server) handleReleaseNotes(r *http.Request, input *releaseNotesInput) (*body[releaseNotes], error) {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return nil, err
	}
	dir := s.repoDir(repo.Name)
	commit, err := s.git.ResolveCommit(ctx, dir, tagRef(input.Tag))
	if input.Tag == "" || err != nil {
		target := input.Target
		if target == "" {
			target = s.git.DefaultBranch(ctx, dir)
		}
		if commit, err = s.git.ResolveCommit(ctx, dir, target); err != nil {
			return nil, badRequest("target " + target + " not found")
		}
	}
	notes, err := s.generateNotes(ctx, repo.Name, input.Tag, commit, input.Prerelease)
	if err != nil {
		return nil, err
	}
	return reply(notes)
}
