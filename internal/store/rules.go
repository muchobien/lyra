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
	MergeMethods    []string  `json:"merge_methods"`
	CreatedAt       time.Time `json:"created_at"`
}

func ruleFromRow(row queries.BranchRule) Rule {
	return Rule{
		ID: row.ID, Repo: row.Repo, Pattern: row.Pattern, RequireSigned: row.RequireSigned, RequireApproval: row.RequireApproval,
		MergeMethods: strings.Split(row.MergeMethods, ","), CreatedAt: row.CreatedAt,
	}
}

func (s *Store) SetRule(repo string, actor Actor, rule Rule) (Rule, int64, error) {
	var saved Rule
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		row, err := q.SetRule(context.Background(), queries.SetRuleParams{
			Repo: repo, Pattern: rule.Pattern, RequireSigned: rule.RequireSigned, RequireApproval: rule.RequireApproval,
			MergeMethods: strings.Join(rule.MergeMethods, ","), CreatedAt: now(),
		})
		if err != nil {
			return err
		}
		saved = ruleFromRow(row)
		eventID, err = addEvent(q, "rule.set", repo, 0, actor, map[string]any{
			"pattern": rule.Pattern, "require_signed": rule.RequireSigned, "require_approval": rule.RequireApproval, "merge_methods": rule.MergeMethods,
		})
		return err
	})
	return saved, eventID, err
}

func (s *Store) Rules(repo string) ([]Rule, error) {
	rows, err := s.queries.ListRules(context.Background(), repo)
	if err != nil {
		return nil, err
	}
	rules := []Rule{}
	for _, row := range rows {
		rules = append(rules, ruleFromRow(row))
	}
	return rules, nil
}

func (s *Store) DeleteRule(repo string, actor Actor, pattern string) (int64, error) {
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		if err := deleted(q.DeleteRule(context.Background(), queries.DeleteRuleParams{Repo: repo, Pattern: pattern})); err != nil {
			return err
		}
		var err error
		eventID, err = addEvent(q, "rule.delete", repo, 0, actor, map[string]any{"pattern": pattern})
		return err
	})
	return eventID, err
}
