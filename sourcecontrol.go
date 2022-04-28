package sourcecontrol

import (
	"fmt"
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
)

// TODO: test this package

func GitCLI() sourcerer.CLI {
	return &git{}
}

type git struct{}

func (*git) Changed() bool   { return false }
func (*git) Setup() []string { return nil }
func (*git) Name() string {
	return "g"
}

func filesWithPrefix(prefixCode string) ([]string, error) {
	return command.BashCommand[[]string]("opts", []string{
		fmt.Sprintf(`results="$(git status --porcelain | grep "%s" | cut -c 4-)";`, prefixCode),
		`toplevel="$(git rev-parse --show-toplevel)";`,
		`for git_path in $results;`,
		`do`,
		`    realpath --relative-to="." "$toplevel/$git_path";`,
		`done;`,
	}).Run(nil)
}

func prefixCompletor[T any](prefixCode string) *command.Completor[T] {
	return &command.Completor[T]{
		Distinct: true,
		SuggestionFetcher: command.SimpleFetcher(func(T, *command.Data) (*command.Completion, error) {
			results, err := filesWithPrefix(prefixCode)
			if err != nil {
				return nil, err
			}
			return &command.Completion{
				Suggestions: results,
			}, nil
		}),
	}
}

func (g *git) Node() *command.Node {
	addCompletor := prefixCompletor[[]string](".[^ ]")

	nvFlag := command.BoolFlag("no-verify", 'n', "Whether or not to run pre-commit checks")

	branchArg := command.Arg[string](
		"BRANCH",
		"Branch",
		command.BashCompletor[string](`git branch | grep -v "\*"`),
	)

	diffArgs := command.ListArg[string](
		"FILE", "Files to diff",
		0, command.UnboundedList,
		&command.Completor[[]string]{
			SuggestionFetcher: command.BashFetcher[[]string]("git diff --name-only --relative"),
			Distinct:          true,
		},
	)

	uaArgs := command.ListArg[string](
		"FILE", "Files to un-add",
		1, command.UnboundedList,
		// prefixCompletor[[]string]("[^ ]."),
		&command.Completor[[]string]{
			SuggestionFetcher: command.BashFetcher[[]string]("git diff --cached --name-only --relative"),
			Distinct:          true,
		},
	)

	ucArgs := command.ListArg[string](
		"FILE", "Files to un-change",
		1, command.UnboundedList,
		prefixCompletor[[]string](".[^ ]"),
	)

	return command.BranchNode(map[string]*command.Node{
		// Simple commands
		"b": command.SerialNodes(
			command.Description("Branch"),
			command.SimpleExecutableNode("git branch"),
		),
		"l": command.SerialNodes(
			command.Description("Pull"),
			command.SimpleExecutableNode("git pull"),
		),
		"m": command.SerialNodes(
			command.Description("Checkout main"),
			command.SimpleExecutableNode("git checkout main"),
		),
		"mm": command.SerialNodes(
			command.Description("Merge main"),
			command.SimpleExecutableNode("git merge main"),
		),
		"p": command.SerialNodes(
			command.Description("Push"),
			command.SimpleExecutableNode("git push"),
		),
		"pp": command.SerialNodes(
			command.Description("Pull and push"),
			command.SimpleExecutableNode("git pull && git push"),
		),

		"s": command.SerialNodes(
			command.Description("Status"),
			command.SimpleExecutableNode("git status"),
		),
		"uco": command.SerialNodes(
			command.Description("Undo commit"),
			command.SimpleExecutableNode("git reset HEAD~"),
		),

		// Complex commands
		// Commit
		"c": command.SerialNodes(
			command.Description("Commit"),
			command.NewFlagNode(
				nvFlag,
			),
			command.ListArg[string]("MESSAGE", "Commit message", 1, command.UnboundedList),
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				r := []string{
					fmt.Sprintf("git commit -m %q", strings.Join(d.StringList("MESSAGE"), " ")),
				}
				if d.Bool(nvFlag.Name()) {
					r = append(r, " --no-verify")
				}
				return r, nil
			}),
		),

		// Commit & push
		"cp": command.SerialNodes(
			command.Description("Commit and push"),
			command.NewFlagNode(
				nvFlag,
			),
			command.ListArg[string]("MESSAGE", "Commit message", 1, command.UnboundedList),
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				r := []string{
					fmt.Sprintf("git commit -m %q", strings.Join(d.StringList("MESSAGE"), " ")),
				}
				if d.Bool(nvFlag.Name()) {
					r = append(r, " --no-verify")
				}
				return append(r, "&& git push"), nil
			}),
		),

		// Checkout new branch
		"cb": command.SerialNodes(
			command.Description("Checkout new branch"),
			branchArg,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				return []string{
					fmt.Sprintf("git checkout -b %s", d.String(branchArg.Name())),
				}, nil
			}),
		),

		// Change/Checkout branch
		"ch": command.SerialNodes(
			command.Description("Checkout existing branch"),
			branchArg,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				return []string{
					fmt.Sprintf("git checkout %s", d.String(branchArg.Name())),
				}, nil
			}),
		),

		// Diff
		"d": command.SerialNodes(
			command.Description("Diff"),
			command.NewFlagNode(
				command.BoolFlag("main", 'm', "Whether to diff against main branch or just local diffs"),
			),
			diffArgs,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				branch := "--"
				if d.Bool("main") {
					branch = "main"
				}
				return []string{
					fmt.Sprintf("git diff %s %s", branch, strings.Join(d.StringList(diffArgs.Name()), " ")),
				}, nil
			}),
		),

		// Undo change
		"uc": command.SerialNodes(
			command.Description("Undo change"),
			ucArgs,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				return []string{
					fmt.Sprintf("git checkout -- %s", strings.Join(d.StringList(ucArgs.Name()), " ")),
				}, nil
			}),
		),

		// Undo add
		"ua": command.SerialNodes(
			command.Description("Undo add"),
			uaArgs,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				return []string{
					fmt.Sprintf("git reset %s", strings.Join(d.StringList(uaArgs.Name()), " ")),
				}, nil
			}),
		),

		// Add
		"a": command.SerialNodes(
			command.Description("Add"),
			command.ListArg[string]("FILES", "Files to add", 0, command.UnboundedList, addCompletor),
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				fs := d.StringList("FILES")
				if len(fs) == 0 {
					return []string{"git add ."}, nil
				}
				return []string{fmt.Sprintf("git add %s", strings.Join(fs, " "))}, nil
			}),
		),
	}, nil, command.BranchAliases(map[string][]string{
		"l": []string{"pl"},
	}))
}
