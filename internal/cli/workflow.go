package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

func (app *app) workflowCommand() *cobra.Command {
	workflow := &cobra.Command{Use: "workflow", Short: "Start Actions workflows"}
	workflow.AddCommand(app.workflowRunCommand())
	return workflow
}

func (app *app) workflowRunCommand() *cobra.Command {
	var ref string
	var inputs []string
	run := &cobra.Command{
		Use:   "run <repo> <workflow>",
		Short: "Start a workflow that has a workflow_dispatch trigger",
		Long: `Start a workflow that has a workflow_dispatch trigger, from the workflow file at --ref
(default: the default branch). <workflow> is the file name, such as deploy.yml, or its
path. Inputs are checked against the workflow's: unknown ones are refused, missing ones
take their default, and required ones without a default must be given. Prints the run.`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			values := map[string]string{}
			for _, input := range inputs {
				name, value, found := strings.Cut(input, "=")
				if !found || name == "" {
					return UsageError{fmt.Errorf("--input %q: want name=value", input)}
				}
				values[name] = value
			}
			body := lyraapi.DispatchRequest{Workflow: args[1], Ref: &ref, Inputs: &values}
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.DispatchWorkflow(cmd.Context(), owner, repo, body))
		},
	}
	run.Flags().StringVar(&ref, "ref", "", "branch or tag to run the workflow from (default: the default branch)")
	run.Flags().StringArrayVar(&inputs, "input", nil, "input as name=value (repeatable)")
	return run
}
