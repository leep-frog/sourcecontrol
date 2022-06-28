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
	createSSHAgent = "ps -p $SSH_AGENT_PID > /dev/null || eval `ssh-agent` && ssh-add"
)

func BranchCompletor() command.Completor[string] {
	return command.BashCompletor[string](`git branch | grep -v "\*"`)
}

var (
	nvFlag     = command.BoolFlag("no-verify", 'n', "Whether or not to run pre-commit checks")
	pushFlag   = command.BoolFlag("push", 'p', "Whether or not to push afterwards")
	messageArg = command.ListArg[string]("MESSAGE", "Commit message", 1, command.UnboundedList)
	branchArg  = command.Arg[string](
		"BRANCH",
		"Branch",
		BranchCompletor(),
	)
	mainFlag        = command.BoolFlag("main", 'm', "Whether to diff against main branch or just local diffs")
	addCompletor    = PrefixCompletor[[]string](".[^ ]")
	filesArg        = command.ListArg[string]("FILES", "Files to add", 0, command.UnboundedList, addCompletor)
	statusCompletor = PrefixCompletor[[]string]("..")
	statusFilesArg  = command.ListArg[string]("FILES", "Files to add", 0, command.UnboundedList, statusCompletor)
	repoName        = command.NewBashCommand[string]("REPO", []string{"git rev-parse --show-toplevel | xargs basename"})
	defRepoArg      = command.Arg[string]("DEFAULT_BRANCH", "Default branch for this git repo")
	forceDelete     = command.BoolFlag("force-delete", 'f', "force delete the branch")
	globalConfig    = command.BoolFlag("global", 'g', "Whether or not to change the global setting")
	newBranchFlag   = command.BoolFlag("new-branch", 'n', "Whether or not to checkout a new branch")
	uaArgs          = command.ListArg[string](
		"FILE", "Files to un-add",
		1, command.UnboundedList,
		// PrefixCompletor[[]string]("[^ ]."),
		command.BashCompletorWithOpts[[]string](&command.Completion{Distinct: true}, "git diff --cached --name-only --relative"),
	)
	diffArgs = command.ListArg[string](
		"FILE", "Files to diff",
		0, command.UnboundedList,
		command.BashCompletorWithOpts[[]string](&command.Completion{Distinct: true}, "git diff --name-only --relative"),
	)
	ucArgs = command.ListArg[string](
		"FILE", "Files to un-change",
		1, command.UnboundedList,
		PrefixCompletor[[]string](".[^ ]"),
	)
	gitLogArg = command.OptionalArg[int]("N", "Number of git logs to display", command.Positive[int](), command.Default(1))
)

func CLI() *git {
	return &git{}
}

func GitAliasers() sourcerer.Option {
	return sourcerer.Aliasers(map[string][]string{
		"gp":   {"g", "p"},
		"gpl":  {"g", "pl"},
		"gs":   {"g", "s"},
		"guco": {"g", "uco"},
		"gb":   {"g", "b"},
		"gc":   {"g", "c"},
		"gcnv": {"g", "c", "-n"},
		"cm":   {"g", "m"},
		"gcb":  {"g", "cb"},
		"gmm":  {"g", "mm"},
		"mm":   {"g", "mm"},
		"gcp":  {"g", "cp"},
		"gd":   {"g", "d"},
		"gdm":  {"g", "d", "-m"},
		"ga":   {"g", "a"},
		"guc":  {"g", "uc"},
		"gua":  {"g", "ua"},
		"ch":   {"g", "ch"},
		"gsh":  {"g", "sh"},
		"sq":   {"g", "q"},
		"gbd":  {"g", "bd"},
		"glg":  {"g", "lg"},
	})
}

type git struct {
	Caches        map[string][][]string
	MainBranches  map[string]string
	DefaultBranch string
	changed       bool
}

func (g *git) Changed() bool {
	return g.changed
}
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

func (g *git) defualtBranch(d *command.Data) string {
	if g.MainBranches == nil {
		if len(g.DefaultBranch) == 0 {
			return "main"
		}
		return g.DefaultBranch
	}
	if m, ok := g.MainBranches[repoName.Get(d)]; ok {
		return m
	}
	if len(g.DefaultBranch) == 0 {
		return "main"
	}
	return g.DefaultBranch
}

func (g *git) setDefualtBranch(o command.Output, d *command.Data, v string) {
	if g.MainBranches == nil {
		g.MainBranches = map[string]string{}
	}
	rn := repoName.Get(d)
	g.MainBranches[rn] = v
	o.Stdoutf("Setting default branch for %s to %s", rn, v)
}

func (g *git) unsetDefualtBranch(o command.Output, d *command.Data) {
	if g.MainBranches == nil {
		return
	}
	rn := repoName.Get(d)
	delete(g.MainBranches, rn)
	o.Stdoutf("Deleting default branch for %s", rn)
}

func (g *git) MarkChanged() {
	g.changed = true
}

func PrefixCompletorScript(prefixCode string) []string {
	return []string{
		fmt.Sprintf(`results="$(git status --porcelain | grep "%s" | cut -c 4-)";`, prefixCode),
		`toplevel="$(git rev-parse --show-toplevel)";`,
		`for git_path in $results;`,
		`do`,
		`    realpath --relative-to="." "$toplevel/$git_path";`,
		`done;`,
	}
}

func PrefixCompletor[T any](prefixCode string) command.Completor[T] {
	return command.CompletorFromFunc(func(T, *command.Data) (*command.Completion, error) {
		results, err := command.NewBashCommand[[]string]("opts", PrefixCompletorScript(prefixCode)).Run(nil)
		if err != nil {
			return nil, err
		}
		return &command.Completion{
			Distinct:        true,
			Suggestions:     results,
			CaseInsensitive: true,
		}, nil
	})
}

func (g *git) Node() *command.Node {
	return command.BranchNode(map[string]*command.Node{
		// Configs
		"cfg": command.SerialNodes(
			command.Description("Config settings"),
			command.BranchNode(map[string]*command.Node{
				"main": command.BranchNode(map[string]*command.Node{
					"show": command.SerialNodes(
						command.ExecutorNode(func(o command.Output, d *command.Data) {
							o.Stdoutf("Global main: %s", g.DefaultBranch)
							for k, v := range g.MainBranches {
								o.Stdoutf("%s: %s", k, v)
							}
						}),
					),
					"set": command.SerialNodes(
						command.NewFlagNode(globalConfig),
						repoName,
						defRepoArg,
						command.ExecutorNode(func(o command.Output, d *command.Data) {
							if globalConfig.Get(d) {
								g.DefaultBranch = defRepoArg.Get(d)
							} else {
								g.setDefualtBranch(o, d, defRepoArg.Get(d))
							}
							g.changed = true
						}),
					),
					"unset": command.SerialNodes(
						repoName,
						command.ExecutorNode(func(o command.Output, d *command.Data) {
							if globalConfig.Get(d) {
								g.DefaultBranch = ""
							} else {
								g.unsetDefualtBranch(o, d)
							}
							g.changed = true
						}),
					),
				}, nil),
			}, nil),
		),

		// Simple commands
		"b": command.SerialNodes(
			command.Description("Branch"),
			command.SimpleExecutableNode("git branch"),
		),
		"l": command.SerialNodes(
			command.Description("Pull"),
			command.SimpleExecutableNode(
				createSSHAgent,
				"git pull",
			),
		),
		"p": command.SerialNodes(
			command.Description("Push"),
			command.SimpleExecutableNode(
				createSSHAgent,
				"git push",
			),
		),
		"pp": command.SerialNodes(
			command.Description("Pull and push"),
			command.SimpleExecutableNode(
				createSSHAgent,
				"git pull && git push",
			),
		),
		"sh": command.SerialNodes(
			command.Description("Create ssh-agent"),
			command.SimpleExecutableNode(createSSHAgent),
		),
		"uco": command.SerialNodes(
			command.Description("Undo commit"),
			command.SimpleExecutableNode("git reset HEAD~"),
		),

		// Complex commands
		// Git log
		"lg": command.SerialNodes(
			command.Description("Git log"),
			gitLogArg,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				return []string{
					fmt.Sprintf("git log -n %d", gitLogArg.Get(d)),
				}, nil
			}),
		),
		// Checkout main
		"m": command.SerialNodes(
			command.Description("Checkout main"),
			repoName,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				return []string{
					fmt.Sprintf("git checkout %s", g.defualtBranch(d)),
				}, nil
			}),
		),
		// Merge main
		"mm": command.SerialNodes(
			command.Description("Merge main"),
			repoName,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				return []string{
					fmt.Sprintf("git merge %s", g.defualtBranch(d)),
				}, nil
			}),
		),
		// Commit
		"c": command.SerialNodes(
			command.Description("Commit"),
			command.NewFlagNode(
				nvFlag,
				pushFlag,
			),
			messageArg,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				var nv string
				if nvFlag.Get(d) {
					nv = "--no-verify"
				}
				r := []string{
					fmt.Sprintf("git commit -m %s %q", nv, strings.Join(messageArg.Get(d), " ")),
				}
				if pushFlag.Get(d) {
					r = append(r,
						createSSHAgent,
						"git push",
					)
				}
				r = append(r, "echo Success!")
				return r, nil
			}),
		),

		// Commit & push
		"cp": command.SerialNodes(
			command.Description("Commit and push"),
			command.NewFlagNode(
				nvFlag,
			),
			messageArg,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				var nv string
				if nvFlag.Get(d) {
					nv = "--no-verify"
				}
				r := fmt.Sprintf("git commit -m %s %q", nv, strings.Join(messageArg.Get(d), " "))
				return []string{
					r,
					createSSHAgent,
					"git push",
					"echo Success!",
				}, nil
			}),
		),

		// Squash
		"q": command.CacheNode(commitCacheKey, g, command.SerialNodes(
			command.Description("Squash local commits"),
			command.NewFlagNode(
				nvFlag,
				pushFlag,
			),
			messageArg,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				r := []string{
					fmt.Sprintf("git reset --soft HEAD~3 && git commit -m %q", strings.Join(messageArg.Get(d), " ")),
				}
				if nvFlag.Get(d) {
					r = append(r, " --no-verify")
				}
				if pushFlag.Get(d) {
					r = append(r, "&& git push")
				}
				r = append(r, "&& echo Success!")
				return []string{strings.Join(r, " ")}, nil
			})),
		),

		// Checkout branch
		"ch": command.SerialNodes(
			command.Description("Checkout new branch"),
			command.NewFlagNode(
				newBranchFlag,
			),
			branchArg,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				flag := ""
				if newBranchFlag.Get(d) {
					flag = "-b "
				}
				return []string{
					fmt.Sprintf("git checkout %s%s", flag, branchArg.Get(d)),
				}, nil
			}),
		),

		// Delete branch
		"bd": command.SerialNodes(
			command.Description("Delete branch"),
			command.NewFlagNode(forceDelete),
			branchArg,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				flag := "-d"
				if forceDelete.Get(d) {
					flag = "-D"
				}
				return []string{
					fmt.Sprintf("git branch %s %s", flag, branchArg.Get(d)),
				}, nil
			}),
		),

		// Diff
		"d": command.SerialNodes(
			command.Description("Diff"),
			command.NewFlagNode(mainFlag),
			diffArgs,
			repoName,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				branch := "--"
				if mainFlag.Get(d) {
					branch = g.defualtBranch(d)
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

		// Status
		"s": command.SerialNodes(
			command.Description("Status"),
			statusFilesArg,
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				return []string{fmt.Sprintf("git status %s", strings.Join(statusFilesArg.Get(d), " "))}, nil
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
		"l": {"pl"},
	}))
}
