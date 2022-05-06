package sourcecontrol

import (
	"fmt"
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
)

// TODO: test this package

const (
	commitCacheKey = "COMMIT_CACHE_KEY"
)

var (
	nvFlag     = command.BoolFlag("no-verify", 'n', "Whether or not to run pre-commit checks")
	pushFlag   = command.BoolFlag("push", 'p', "Whether or not to push afterwards")
	messageArg = command.ListArg[string]("MESSAGE", "Commit message", 1, command.UnboundedList)
	branchArg  = command.Arg[string](
		"BRANCH",
		"Branch",
		command.BashCompletor[string](`git branch | grep -v "\*"`),
	)
	mainFlag     = command.BoolFlag("main", 'm', "Whether to diff against main branch or just local diffs")
	addCompletor = prefixCompletor[[]string](".[^ ]")
	filesArg     = command.ListArg[string]("FILES", "Files to add", 0, command.UnboundedList, addCompletor)
)

func GitCLI() sourcerer.CLI {
	return &git{}
}

func GitAliasers() sourcerer.Option {
	return sourcerer.Aliasers(map[string][]string{
		"gp":   []string{"g", "p"},
		"gpl":  []string{"g", "pl"},
		"gs":   []string{"g", "s"},
		"guco": []string{"g", "uco"},
		"gb":   []string{"g", "b"},
		"gc":   []string{"g", "c"},
		"gcnv": []string{"g", "c", "-n"},
		"cm":   []string{"g", "m"},
		"gcb":  []string{"g", "cb"},
		"gmm":  []string{"g", "mm"},
		"mm":   []string{"g", "mm"},
		"gcp":  []string{"g", "cp"},
		"gd":   []string{"g", "d"},
		"gdm":  []string{"g", "d", "-m"},
		"ga":   []string{"g", "a"},
		"guc":  []string{"g", "uc"},
		"gua":  []string{"g", "ua"},
		"ch":   []string{"g", "ch"},
	})
}

type git struct {
	Caches  map[string][][]string
	changed bool
}

func (*git) Changed() bool   { return false }
func (*git) Setup() []string { return nil }
func (*git) Name() string {
	return "g"
}

func (g *git) Cache() map[string][][]string {
	if g.Caches == nil {
		g.Caches = map[string][][]string{}
	}
	return g.Caches
}

func (g *git) MarkChanged() {
	g.changed = true
}

func filesWithPrefix(prefixCode string) ([]string, error) {
	return command.NewBashCommand[[]string]("opts", []string{
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
		Fetcher: command.SimpleFetcher(func(T, *command.Data) (*command.Completion, error) {
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
	diffArgs := command.ListArg[string](
		"FILE", "Files to diff",
		0, command.UnboundedList,
		&command.Completor[[]string]{
			Fetcher:  command.BashFetcher[[]string]("git diff --name-only --relative"),
			Distinct: true,
		},
	)

	uaArgs := command.ListArg[string](
		"FILE", "Files to un-add",
		1, command.UnboundedList,
		// prefixCompletor[[]string]("[^ ]."),
		&command.Completor[[]string]{
			Fetcher:  command.BashFetcher[[]string]("git diff --cached --name-only --relative"),
			Distinct: true,
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
		"c": command.CacheNode(commitCacheKey, g, command.SerialNodes(
			command.Description("Commit"),
			command.NewFlagNode(
				nvFlag,
				pushFlag,
			),
			messageArg,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				r := []string{
					fmt.Sprintf("git commit -m %q", strings.Join(messageArg.Get(d), " ")),
				}
				if nvFlag.Get(d) {
					r = append(r, " --no-verify")
				}
				if pushFlag.Get(d) {
					r = append(r, "&& git push")
				}
				r = append(r, "&& echo Success!")
				//if d.Bool(pushFlag.Name())
				return []string{strings.Join(r, " ")}, nil
			})),
		),

		// Commit & push
		"cp": command.SerialNodes(
			command.Description("Commit and push"),
			command.NewFlagNode(
				nvFlag,
			),
			messageArg,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				r := []string{
					fmt.Sprintf("git commit -m %q", strings.Join(messageArg.Get(d), " ")),
				}
				if nvFlag.Get(d) {
					r = append(r, "--no-verify")
				}
				return []string{strings.Join(append(r, "&& git push"), " ")}, nil
			}),
		),

		// Checkout new branch
		"cb": command.SerialNodes(
			command.Description("Checkout new branch"),
			branchArg,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				return []string{
					fmt.Sprintf("git checkout -b %s", branchArg.Get(d)),
				}, nil
			}),
		),

		// Change/Checkout branch
		"ch": command.SerialNodes(
			command.Description("Checkout existing branch"),
			branchArg,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				return []string{
					fmt.Sprintf("git checkout %s", branchArg.Get(d)),
				}, nil
			}),
		),

		// Diff
		"d": command.SerialNodes(
			command.Description("Diff"),
			command.NewFlagNode(mainFlag),
			diffArgs,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				branch := "--"
				if mainFlag.Get(d) {
					branch = "main"
				}
				return []string{
					fmt.Sprintf("git diff %s %s", branch, strings.Join(diffArgs.Get(d), " ")),
				}, nil
			}),
		),

		// Undo change
		"uc": command.SerialNodes(
			command.Description("Undo change"),
			ucArgs,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				return []string{
					fmt.Sprintf("git checkout -- %s", strings.Join(ucArgs.Get(d), " ")),
				}, nil
			}),
		),

		// Undo add
		"ua": command.SerialNodes(
			command.Description("Undo add"),
			uaArgs,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				return []string{
					fmt.Sprintf("git reset %s", strings.Join(ucArgs.Get(d), " ")),
				}, nil
			}),
		),

		// Add
		"a": command.SerialNodes(
			command.Description("Add"),
			filesArg,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				fs := filesArg.Get(d)
				if len(fs) == 0 {
					return []string{"git add ."}, nil
				}
				return []string{fmt.Sprintf("git add %s", strings.Join(fs, " "))}, nil
			}),
		),
	}, nil, command.BranchSynonyms(map[string][]string{
		"l": []string{"pl"},
	}))
}
