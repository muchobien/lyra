package cli

import (
	"fmt"
	"io"
	"os"

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

func addTextFlags(cmd *cobra.Command, name, usage string) {
	cmd.Flags().String(name, "", usage)
	cmd.Flags().String(name+"-file", "", "read the "+name+" from this file, or - for stdin")
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

func readInput(cmd *cobra.Command, file string) ([]byte, error) {
	if file == "-" {
		return io.ReadAll(cmd.InOrStdin())
	}
	return os.ReadFile(file)
}
