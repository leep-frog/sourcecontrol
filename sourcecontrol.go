package sourcecontrol

import (
	"fmt"
	"strings"

	"github.com/leep-frog/command"
)

type Git struct{}

func (*Git) Load(jsn string) error { return nil }
func (*Git) Changed() bool         { return false }
func (*Git) Setup() []string       { return nil }
func (*Git) Name() string {
	return "g"
}

func (g *Git) filesWithPrefix(prefixCode string) ([]string, error) {
	return command.BashCommand[[]string]("opts", []string{
		fmt.Sprintf(`results="$(git status --porcelain | grep "%s" | cut -c 4-)";`, prefixCode),
		`relative_results="";`,
		`toplevel="$(git rev-parse --show-toplevel)";`,
		`for git_path in $results;`,
		`do`,
		`    full_path="$toplevel/$git_path";`,
		`    path="$(realpath --relative-to="." "$full_path")";`,
		`    relative_results="$relative_results $path";`,
		`done;`,
	}).Run(nil)
}

//func (g *Git) autocompleteStatus

func (g *Git) Node() *command.Node {
	//porcelain
	ac := &command.Completor[[]string]{
		SuggestionFetcher: command.SimpleFetcher(func(ts []string, d *command.Data) (*command.Completion, error) {
			results, err := g.filesWithPrefix(".[^ ]")
			if err != nil {
				return nil, err
			}
			return &command.Completion{
				Suggestions: results,
			}, nil
		}),
	}
	return command.BranchNode(map[string]*command.Node{
		"s": command.SerialNodes(
			command.Description("Status"),
			command.SimpleExecutableNode("git status"),
		),
		"b": command.SerialNodes(
			command.Description("Branch"),
			command.SimpleExecutableNode("git branch"),
		),
		"c": command.SerialNodes(
			command.Description("Commit"),
			command.ListArg[string]("MESSAGE", "Commit message", 1, command.UnboundedList),
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				return []string{fmt.Sprintf("git commit %q", strings.Join(d.StringList("MESSAGE"), " "))}, nil
			}),
		),
		"p": command.SerialNodes(
			command.Description("Push"),
			command.SimpleExecutableNode("git push"),
		),
		"l": command.SerialNodes(
			command.Description("Pull"),
			command.SimpleExecutableNode("git pull"),
		),
		"pp": command.SerialNodes(
			command.Description("Pull and push"),
			command.SimpleExecutableNode("git pull && git push"),
		),
		"a": command.SerialNodes(
			command.Description("Add"),
			command.ListArg[string]("FILES", "Files to add", 0, command.UnboundedList, ac),
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				fs := d.StringList("FILES")
				if len(fs) == 0 {
					return []string{"git add ."}, nil
				}
				return []string{fmt.Sprintf("git add %s", strings.Join(fs, " "))}, nil
			}),
		),
	}, nil)
}

//func (g *Git) status()
