package cli

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"
)

var escaped = json.JoinOptions(
	jsontext.EscapeForHTML(true),
	jsontext.EscapeForJS(true),
	jsontext.AllowInvalidUTF8(true),
)

func (app *app) emit(cmd *cobra.Command, output []byte) error {
	if len(app.fields) > 0 {
		projected, err := project(output, app.fields)
		if err != nil {
			return err
		}
		output = projected
	}
	if app.jq == "" {
		_, err := cmd.OutOrStdout().Write(output)
		return err
	}
	return filter(cmd.OutOrStdout(), output, app.jq)
}

func (app *app) emitIndented(cmd *cobra.Command, value any) error {
	encoded, err := json.Marshal(value, escaped, jsontext.WithIndent("  "))
	if err != nil {
		return err
	}
	return app.emit(cmd, append(encoded, '\n'))
}

func project(output []byte, fields []string) ([]byte, error) {
	var value any
	if err := json.Unmarshal(output, &value); err != nil {
		return nil, UsageError{errors.New("--json: this command doesn't print JSON")}
	}
	switch typed := value.(type) {
	case map[string]any:
		picked, err := pick(typed, fields)
		if err != nil {
			return nil, err
		}
		value = picked
	case []any:
		for index, element := range typed {
			object, isObject := element.(map[string]any)
			if !isObject {
				continue
			}
			picked, err := pick(object, fields)
			if err != nil {
				return nil, err
			}
			typed[index] = picked
		}
	}
	encoded, err := json.Marshal(value, escaped, json.Deterministic(true), jsontext.WithIndent("  "))
	return append(encoded, '\n'), err
}

func pick(object map[string]any, fields []string) (map[string]any, error) {
	picked := map[string]any{}
	for _, field := range fields {
		value, found := object[field]
		if !found {
			available := strings.Join(slices.Sorted(maps.Keys(object)), ", ")
			return nil, UsageError{fmt.Errorf("--json: unknown field %s; available fields: %s", field, available)}
		}
		picked[field] = value
	}
	return picked, nil
}

func filter(out io.Writer, output []byte, expression string) error {
	query, err := gojq.Parse(expression)
	if err != nil {
		return UsageError{fmt.Errorf("--jq: %w", err)}
	}
	var input any
	if err := json.Unmarshal(output, &input); err != nil {
		return err
	}
	results := query.Run(input)
	for {
		value, ok := results.Next()
		if !ok {
			return nil
		}
		if err, isError := value.(error); isError {
			return fmt.Errorf("--jq: %w", err)
		}
		if text, isString := value.(string); isString {
			fmt.Fprintln(out, text)
			continue
		}
		encoded, err := gojq.Marshal(value)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(encoded))
	}
}
