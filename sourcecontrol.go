package sourcecontrol

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
)

func joinByOS(cmds ...string) ([]string, error) {
	switch sourcerer.CurrentOS.Name() {
	case "linux":
		return []string{strings.Join(cmds, " && ")}, nil
	case "windows":
		var wr []string
		for _, c := range cmds {
			wr = append(wr, wCmd(c))
		}
		return wr, nil
	}
	return nil, fmt.Errorf("Unknown OS (%q)", sourcerer.CurrentOS.Name())
}

func executableJoinByOS(cmds ...string) command.Processor {
	return command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
		s, err := joinByOS(cmds...)
		return s, o.Err(err)
	})
}

const (
	commitCacheKey = "COMMIT_CACHE_KEY"
	// See https://github.com/leep-frog/ssh
	createSSHAgentCommand = "ssh-add"

	linuxOS   = "linux"
	windowsOS = "windows"
)

func wCmd(s string) string {
	return strings.Join([]string{
		s,
		// Windows escapes quotes with backtick, hence why '%q' alone isn't sufficient here
		fmt.Sprintf("if (!$?) { throw %q }", fmt.Sprintf("Command failed: %s", strings.ReplaceAll(s, `"`, "'"))),
	}, "\n")
}

var (
	sshNode = command.SerialNodes(
		command.FunctionWrap(),
		command.SimpleExecutableProcessor(createSSHAgentCommand),
	)
	nvFlag     = command.BoolValueFlag("no-verify", 'n', "Whether or not to run pre-commit checks", "--no-verify ")
	pushFlag   = command.BoolFlag("push", 'p', "Whether or not to push afterwards")
	messageArg = command.ListArg[string]("MESSAGE", "Commit message", 1, command.UnboundedList)
	branchArg  = command.Arg[string](
		"BRANCH",
		"Branch",
		BranchCompleter(),
	)
	mainFlag        = command.BoolFlag("main", 'm', "Whether to diff against main branch or just local diffs")
	prevCommitFlag  = command.BoolFlag("commit", 'c', "Whether to diff against the previous commit")
	addCompleter    = PrefixCompleter[[]string](".[^ ]")
	filesArg        = command.ListArg[string]("FILES", "Files to add", 0, command.UnboundedList, addCompleter)
	statusCompleter = PrefixCompleter[[]string]("..")
	statusFilesArg  = command.ListArg[string]("FILES", "Files to add", 0, command.UnboundedList, statusCompleter)
	repoName        = &command.ShellCommand[string]{
		ArgName:     "REPO",
		CommandName: "git",
		Args: []string{
			"config",
			"--get",
			"remote.origin.url",
		},
	}
	defRepoArg     = command.Arg[string]("DEFAULT_BRANCH", "Default branch for this git repo")
	forceDelete    = command.BoolFlag("force-delete", 'f', "force delete the branch")
	globalConfig   = command.BoolFlag("global", 'g', "Whether or not to change the global setting")
	newBranchFlag  = command.BoolFlag("new-branch", 'n', "Whether or not to checkout a new branch")
	whitespaceFlag = command.BoolValueFlag("whitespace", 'w', "Whether or not to show whitespace in diffs", "-w")
	uaArgs         = command.ListArg[string](
		"FILE", "Files to un-add",
		1, command.UnboundedList,
		// PrefixCompleter[[]string]("[^ ]."),
		command.ShellCommandCompleterWithOpts[[]string](&command.Completion{Distinct: true}, "git", "diff", "--cached", "--name-only", "--relative"),
	)
	diffArgs = command.ListArg[string](
		"FILE", "Files to diff",
		0, command.UnboundedList,
		command.ShellCommandCompleterWithOpts[[]string](&command.Completion{Distinct: true}, "git", "diff", "--name-only", "--relative"),
	)
	ucArgs = command.ListArg[string](
		"FILE", "Files to un-change",
		1, command.UnboundedList,
		PrefixCompleter[[]string](".[^ ]"),
	)
	gitLogArg = command.OptionalArg[int]("N", "Number of git logs to display", command.Positive[int](), command.Default(1))
)

func CLI() *git {
	return &git{}
}

func BranchCompleter() command.Completer[string] {
	return command.CompleterFromFunc(func(s string, d *command.Data) (*command.Completion, error) {
		c, err := command.ShellCommandCompleter[string](`git branch | grep -v "\*"`).Complete(s, d)
		if c == nil || err != nil {
			return c, err
		}

		var r []string
		for _, s := range c.Suggestions {
			if !strings.Contains(s, "*") {
				r = append(r, s)
			}
		}
		c.Suggestions = r
		return c, nil
	})
}

func GitAliasers() sourcerer.Option {
	return sourcerer.Aliasers(map[string][]string{
		"gp": {"g", "p"},
		// Don't include 'gl' since that is an alias of goleep
		"gpl":  {"g", "pl"},
		"gs":   {"g", "s"},
		"guco": {"g", "uco"},
		"gb":   {"g", "b"},
		"gc":   {"g", "c"},
		"gcnv": {"g", "c", "-n"},
		"cm":   {"g", "m"},
		"gcb":  {"g", "ch"},
		"gmm":  {"g", "mm"},
		"mm":   {"g", "mm"},
		"gcp":  {"g", "cp"},
		"gd":   {"g", "d"},
		"gdm":  {"g", "d", "-m"},
		"ga":   {"g", "a"},
		"guc":  {"g", "uc"},
		"gua":  {"g", "ua"},
		"ch":   {"g", "ch"},
		"sq":   {"g", "q"},
		"gbd":  {"g", "bd"},
		"glg":  {"g", "lg"},
		"gedo": {"g", "edo"},
		"gop":  {"g", "op"},
		"gush": {"g", "ush"},
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
	o.Stdoutf("Setting default branch for %s to %s\n", rn, v)
}

func (g *git) unsetDefualtBranch(o command.Output, d *command.Data) {
	if g.MainBranches == nil {
		return
	}
	rn := repoName.Get(d)
	delete(g.MainBranches, rn)
	o.Stdoutf("Deleting default branch for %s\n", rn)
}

func (g *git) MarkChanged() {
	g.changed = true
}

func PrefixCompleter[T any](prefixCode string) command.Completer[T] {
	return command.CompleterFromFunc(func(t T, d *command.Data) (*command.Completion, error) {
		prefixRegex := regexp.MustCompile(prefixCode)
		bc := &command.ShellCommand[[]string]{
			ArgName:     "opts",
			CommandName: "git",
			Args: []string{
				"status",
				// Note: this requires that `git config status.relativePaths true`
				"--porcelain=v2",
			},
		}
		results, err := bc.Run(nil, d)
		if err != nil {
			return nil, fmt.Errorf("failed to get git status: %v", err)
		}

		var suggestions []string
		for _, result := range results {
			// 1 .M ... 100644 100644 100644 e1548292489441c42682f38f2590e24d66a8587a e1548292489441c42682f38f2590e24d66a8587a sourcecontrol.go
			// Format is
			parts := strings.Split(result, " ")
			code := parts[1]
			// if file has a space in the name, we need to rejoin int
			file := strings.Join(parts[8:], " ")
			if prefixRegex.MatchString(code) {
				suggestions = append(suggestions, file)
			}
		}
		return &command.Completion{
			Distinct:        true,
			Suggestions:     suggestions,
			CaseInsensitive: true,
		}, nil
	})
}

func (g *git) Node() command.Node {
	return &command.BranchNode{
		Branches: map[string]command.Node{
			// Configs
			"cfg": command.SerialNodes(
				command.Description("Config settings"),
				&command.BranchNode{
					Branches: map[string]command.Node{
						"main": &command.BranchNode{
							Branches: map[string]command.Node{
								"show": command.SerialNodes(
									&command.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
										o.Stdoutf("Global main: %s\n", g.DefaultBranch)
										for k, v := range g.MainBranches {
											o.Stdoutf("%s: %s\n", k, v)
										}
										return nil
									}},
								),
								"set": command.SerialNodes(
									command.FlagProcessor(globalConfig),
									repoName,
									defRepoArg,
									&command.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
										if globalConfig.Get(d) {
											g.DefaultBranch = defRepoArg.Get(d)
										} else {
											g.setDefualtBranch(o, d, defRepoArg.Get(d))
										}
										g.changed = true
										return nil
									}},
								),
								"unset": command.SerialNodes(
									repoName,
									&command.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
										if globalConfig.Get(d) {
											g.DefaultBranch = ""
										} else {
											g.unsetDefualtBranch(o, d)
										}
										g.changed = true
										return nil
									}},
								),
							}},
					}},
			),

			// Simple commands
			"b": command.SerialNodes(
				command.Description("Branch"),
				command.SimpleExecutableProcessor("git branch"),
			),
			"l": command.SerialNodes(
				command.Description("Pull"),
				sshNode,
				command.SimpleExecutableProcessor(
					"git pull",
				),
			),
			"p": command.SerialNodes(
				command.Description("Push"),
				sshNode,
				command.SimpleExecutableProcessor(
					"git push",
				),
			),
			"pp": command.SerialNodes(
				command.Description("Pull and push"),
				sshNode,
				executableJoinByOS(
					"git pull",
					"git push",
				),
				command.SimpleExecutableProcessor(),
			),
			"sh": command.SerialNodes(
				command.Description("Create ssh-agent"),
				sshNode,
			),
			"uco": command.SerialNodes(
				command.Description("Undo commit"),
				command.SimpleExecutableProcessor("git reset HEAD~"),
			),
			"f": command.SerialNodes(
				command.Description("Git fetch"),
				command.SimpleExecutableProcessor("git fetch"),
			),
			"op": command.SerialNodes(
				command.Description("Git stash pop"),
				command.SimpleExecutableProcessor("git stash pop"),
			),
			"ush": command.SerialNodes(
				command.Description("Git stash push"),
				command.SimpleExecutableProcessor("git stash push"),
			),

			// Complex commands
			"edo": command.SerialNodes(
				command.Description("Adds local changes to the previous commit"),
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					bc := &command.ShellCommand[string]{
						CommandName: "git",
						Args: []string{
							"log",
							"-1",
							"--pretty=%B",
						},
						HideStderr: true,
					}
					s, err := bc.Run(nil, d)
					if err != nil {
						return nil, o.Annotatef(err, "failed to get previous commit message")
					}

					return joinByOS(
						"guco",
						"ga .",
						fmt.Sprintf("gc %q", s),
					)
				}),
			),
			// Git log
			"lg": command.SerialNodes(
				command.Description("Git log"),
				gitLogArg,
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					return []string{
						fmt.Sprintf("git log -n %d", gitLogArg.Get(d)),
					}, nil
				}),
			),
			// Checkout main
			"m": command.SerialNodes(
				command.Description("Checkout main"),
				repoName,
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					return []string{
						fmt.Sprintf("git checkout %s", g.defualtBranch(d)),
					}, nil
				}),
			),
			// Merge main
			"mm": command.SerialNodes(
				command.Description("Merge main"),
				repoName,
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					return []string{
						fmt.Sprintf("git merge %s", g.defualtBranch(d)),
					}, nil
				}),
			),
			// Commit
			"c": command.SerialNodes(
				command.Description("Commit"),
				command.FlagProcessor(
					nvFlag,
					pushFlag,
				),
				messageArg,
				command.If(
					sshNode,
					func(i *command.Input, d *command.Data) bool {
						return pushFlag.Get(d)
					},
				),
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					r := []string{
						fmt.Sprintf("git commit %s-m %q", nvFlag.Get(d), strings.Join(messageArg.Get(d), " ")),
					}
					if pushFlag.Get(d) {
						r = append(r,
							"git push",
						)
					}
					r = append(r, "echo Success!")

					return joinByOS(r...)
				}),
			),

			// Commit & push
			"cp": command.SerialNodes(
				command.Description("Commit and push"),
				command.FlagProcessor(
					nvFlag,
				),
				messageArg,
				sshNode,
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					return joinByOS(
						fmt.Sprintf("git commit %s-m %q", nvFlag.Get(d), strings.Join(messageArg.Get(d), " ")),
						"git push",
						"echo Success!",
					)
				}),
			),

			// Squash
			"q": command.CacheNode(commitCacheKey, g, command.SerialNodes(
				command.Description("Squash local commits"),
				command.FlagProcessor(
					nvFlag,
					pushFlag,
				),
				messageArg,
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					// TODO: Fix and test this
					// TODO: also make sure to combine with "&&" if relevant
					r := []string{
						"git reset --soft HEAD~3",
						fmt.Sprintf("git commit -m %q %s", strings.Join(messageArg.Get(d), " "), nvFlag.Get(d)),
					}
					if pushFlag.Get(d) {
						r = append(r, "git push")
					}
					r = append(r, "echo Success!")
					return joinByOS(r...)
				})),
			),

			// Checkout branch
			"ch": command.SerialNodes(
				command.Description("Checkout new branch"),
				command.FlagProcessor(
					newBranchFlag,
				),
				branchArg,
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
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
				command.FlagProcessor(forceDelete),
				branchArg,
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
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
				command.FlagProcessor(
					mainFlag,
					prevCommitFlag,
					whitespaceFlag,
				),
				diffArgs,
				repoName,
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					branch := "--"
					if mainFlag.Get(d) {
						branch = g.defualtBranch(d)
					}
					if prevCommitFlag.Get(d) {
						branch = `"$(git rev-parse @~1)"`
					}
					return []string{
						fmt.Sprintf("git diff %s %s %s", whitespaceFlag.Get(d), branch, strings.Join(diffArgs.Get(d), " ")),
					}, nil
				}),
			),

			// Undo change
			"uc": command.SerialNodes(
				command.Description("Undo change"),
				ucArgs,
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					return []string{
						fmt.Sprintf("git checkout -- %s", strings.Join(ucArgs.Get(d), " ")),
					}, nil
				}),
			),

			// Undo add
			"ua": command.SerialNodes(
				command.Description("Undo add"),
				uaArgs,
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					return []string{
						fmt.Sprintf("git reset %s", strings.Join(ucArgs.Get(d), " ")),
					}, nil
				}),
			),

			// Status
			"s": command.SerialNodes(
				command.Description("Status"),
				statusFilesArg,
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					return []string{fmt.Sprintf("git status %s", strings.Join(statusFilesArg.Get(d), " "))}, nil
				}),
			),

			// Add
			"a": command.SerialNodes(
				command.Description("Add"),
				filesArg,
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					fs := filesArg.Get(d)
					if len(fs) == 0 {
						return []string{"git add ."}, nil
					}
					return []string{fmt.Sprintf("git add %s", strings.Join(fs, " "))}, nil
				}),
			),

			// Rebase
			"rb": &command.BranchNode{
				Branches: map[string]command.Node{
					"a": command.SerialNodes(
						command.Description("Abort"),
						command.SimpleExecutableProcessor("git rebase --abort"),
						command.EchoExecuteData(),
					),
					"c": command.SerialNodes(
						command.Description("Continue"),
						command.SimpleExecutableProcessor("git rebase --continue"),
						command.EchoExecuteData(),
					),
				},
			},
		},
		Synonyms: command.BranchSynonyms(map[string][]string{
			"l": {"pl"},
		}),
	}
}
