package store

import (
	"context"
	"strings"
	"time"

	"github.com/muchobien/lyra/internal/store/queries"
)

type Rule struct {
	ID              int64     `json:"id"`
	Repo            string    `json:"repo"`
	Pattern         string    `json:"pattern"`
	RequireSigned   bool      `json:"require_signed"`
	RequireApproval bool      `json:"require_approval"`
	RequireChecks   bool      `json:"require_checks"`
	AllowForcePush  bool      `json:"allow_force_push"`
	AllowDeletion   bool      `json:"allow_deletion"`
	MergeMethods    []string  `json:"merge_methods"`
	CreatedAt       time.Time `json:"created_at"`
}

type ruleSetEvent struct {
	AllowDeletion   bool     `json:"allow_deletion"`
	AllowForcePush  bool     `json:"allow_force_push"`
	MergeMethods    []string `json:"merge_methods"`
	Pattern         string   `json:"pattern"`
	RequireApproval bool     `json:"require_approval"`
	RequireChecks   bool     `json:"require_checks"`
	RequireSigned   bool     `json:"require_signed"`
}

type ruleDeletedEvent struct {
	Pattern string `json:"pattern"`
}

func ruleFromRow(row queries.BranchRule) Rule {
	return Rule{
		ID:              row.ID,
		Repo:            row.Repo,
		Pattern:         row.Pattern,
		RequireSigned:   row.RequireSigned,
		RequireApproval: row.RequireApproval,
		RequireChecks:   row.RequireChecks,
		AllowForcePush:  row.AllowForcePush,
		AllowDeletion:   row.AllowDeletion,
		MergeMethods:    strings.Split(row.MergeMethods, ","),
		CreatedAt:       row.CreatedAt,
	}
}

func (s *Store) SetRule(ctx context.Context, repo string, actor Actor, rule Rule) (Rule, int64, error) {
	var saved Rule
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		row, err := q.SetRule(ctx, queries.SetRuleParams{
			Repo:            repo,
			Pattern:         rule.Pattern,
			RequireSigned:   rule.RequireSigned,
			RequireApproval: rule.RequireApproval,
			RequireChecks:   rule.RequireChecks,
			AllowForcePush:  rule.AllowForcePush,
			AllowDeletion:   rule.AllowDeletion,
			MergeMethods:    strings.Join(rule.MergeMethods, ","),
			CreatedAt:       now(),
		})
		if err != nil {
			return err
		}
		saved = ruleFromRow(row)
		eventID, err = addEvent(ctx, q, "rule.set", repo, 0, actor, ruleSetEvent{
			AllowDeletion:   rule.AllowDeletion,
			AllowForcePush:  rule.AllowForcePush,
			MergeMethods:    rule.MergeMethods,
			Pattern:         rule.Pattern,
			RequireApproval: rule.RequireApproval,
			RequireChecks:   rule.RequireChecks,
			RequireSigned:   rule.RequireSigned,
		})
		return err
	})
	return saved, eventID, err
}

func (s *Store) Rules(ctx context.Context, repo string) ([]Rule, error) {
	rows, err := s.queries.ListRules(ctx, repo)
	if err != nil {
		return nil, err
	}
	rules := []Rule{}
	for _, row := range rows {
		rules = append(rules, ruleFromRow(row))
	}
	return rules, nil
}

func (s *Store) DeleteRule(ctx context.Context, repo string, actor Actor, pattern string) (int64, error) {
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		if err := deleted(q.DeleteRule(ctx, queries.DeleteRuleParams{Repo: repo, Pattern: pattern})); err != nil {
			return err
		}
		var err error
		eventID, err = addEvent(ctx, q, "rule.delete", repo, 0, actor, ruleDeletedEvent{Pattern: pattern})
		return err
	})
	return eventID, err
}
