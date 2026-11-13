package cli

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"

	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"
)

var escaped = json.JoinOptions(
	jsontext.EscapeForHTML(true),
	jsontext.EscapeForJS(true),
	jsontext.AllowInvalidUTF8(true),
)

func (app *app) emit(cmd *cobra.Command, output []byte) error {
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
