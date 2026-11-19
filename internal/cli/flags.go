package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type UsageError struct {
	Err error
}

func (usage UsageError) Error() string {
	return usage.Err.Error()
}

func ExactArgs(count int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != count {
			path := cmd.CommandPath()
			message := "%s takes %d argument(s), got %d; see %s --help"
			return UsageError{fmt.Errorf(message, path, count, len(args), path)}
		}
		return nil
	}
}

func MinimumArgs(count int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < count {
			path := cmd.CommandPath()
			message := "%s takes at least %d argument(s), got %d; see %s --help"
			return UsageError{fmt.Errorf(message, path, count, len(args), path)}
		}
		return nil
	}
}

func MaximumArgs(count int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > count {
			path := cmd.CommandPath()
			message := "%s takes at most %d argument(s), got %d; see %s --help"
			return UsageError{fmt.Errorf(message, path, count, len(args), path)}
		}
		return nil
	}
}

type seconds time.Duration

func (value *seconds) String() string {
	if *value == 0 {
		return "0"
	}
	return time.Duration(*value).String()
}

func (value *seconds) Set(text string) error {
	if count, err := strconv.ParseFloat(text, 64); err == nil {
		*value = seconds(count * float64(time.Second))
		return nil
	}
	duration, err := time.ParseDuration(text)
	if err != nil {
		return fmt.Errorf("%q isn't a number of seconds or a duration such as 30m", text)
	}
	*value = seconds(duration)
	return nil
}

func (value *seconds) Type() string {
	return "duration"
}

func addTimeoutFlag(cmd *cobra.Command, timeout *time.Duration) {
	usage := "exit 8 after this long, in seconds or as 30s or 10m (default: no limit)"
	cmd.Flags().Var((*seconds)(timeout), "timeout", usage)
}

func addTextFlags(cmd *cobra.Command, name, usage string) {
	cmd.Flags().StringP(name, name[:1], "", usage)
	cmd.Flags().StringP(name+"-file", "F", "", "read the "+name+" from this file, or - for stdin")
}

func readText(cmd *cobra.Command, name string) (string, bool, error) {
	flags := cmd.Flags()
	if flags.Changed(name) && flags.Changed(name+"-file") {
		return "", false, UsageError{fmt.Errorf("pass --%s or --%s-file, not both", name, name)}
	}
	if flags.Changed(name + "-file") {
		file, _ := flags.GetString(name + "-file")
		data, err := readInput(cmd, file)
		return string(data), true, err
	}
	text, _ := flags.GetString(name)
	return text, flags.Changed(name), nil
}

func readValue(cmd *cobra.Command) (string, error) {
	if cmd.Flags().Changed("body") {
		return cmd.Flags().GetString("body")
	}
	value, err := io.ReadAll(cmd.InOrStdin())
	return strings.TrimSuffix(string(value), "\n"), err
}

func readInput(cmd *cobra.Command, file string) ([]byte, error) {
	if file == "-" {
		return io.ReadAll(cmd.InOrStdin())
	}
	return os.ReadFile(file)
}
