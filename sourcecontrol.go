package sourcecontrol

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

const DefaultDefaultBranch = "main"

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
	// See https://github.com/leep-frog/ssh
	// createSSHAgentCommand = "ssh-add"
	createSSHAgentCommand = ""
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
	mainFlag       = command.BoolFlag("main", 'm', "Whether to diff against main branch or just local diffs")
	prevCommitFlag = command.BoolFlag("commit", 'c', "Whether to diff against the previous commit")

	// The two dots represent [file state in the cache (e.g. added/green), file state not in the cache (red file)]
	redFileCompleter            = PrefixCompleter[[]string](true, regexp.MustCompile(`^.[^\.]$`))
	greenFileCompleter          = PrefixCompleter[[]string](false, regexp.MustCompile(`^[^\.].$`))
	redFileCompleterNoDeletes   = command.ShellCommandCompleterWithOpts[[]string](&command.Completion{Distinct: true, CaseInsensitive: true}, "git", "diff", "--name-only", "--relative")
	greenFileCompleterNoDeletes = command.ShellCommandCompleterWithOpts[[]string](&command.Completion{Distinct: true, CaseInsensitive: true}, "git", "diff", "--cached", "--name-only", "--relative")

	filesArg         = command.ListArg[string]("FILES", "Files to add", 0, command.UnboundedList, redFileCompleter)
	allFileCompleter = PrefixCompleter[[]string](true, regexp.MustCompile(".*"))
	statusFilesArg   = command.ListArg[string]("FILES", "Files to add", 0, command.UnboundedList, allFileCompleter)
	repoName         = &command.ShellCommand[string]{
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
		greenFileCompleter,
	)
	diffArgs = command.ListArg[string](
		"FILE", "Files to diff",
		0, command.UnboundedList,
		redFileCompleterNoDeletes,
	)
	ucArgs = command.ListArg[string](
		"FILE", "Files to un-change",
		1, command.UnboundedList,
		redFileCompleter,
	)
	gitLogArg      = command.OptionalArg[int]("N", "Number of git logs to display", command.Positive[int](), command.Default(1))
	gitLogDiffFlag = command.BoolFlag("diff", 'd', "Whether or not to diff the current changes against N commits prior")
	stashArgs      = command.ListArg[string](
		"STASH_ARGS", "Args to pass to `git stash push/pop`",
		0, command.UnboundedList,
		allFileCompleter,
	)
)

func CLI() *git {
	return &git{}
}

func BranchCompleter() command.Completer[string] {
	return command.CompleterFromFunc(func(s string, d *command.Data) (*command.Completion, error) {
		c, err := command.ShellCommandCompleter[string]("git", "branch", "--list").Complete(s, d)
		if c == nil || err != nil {
			return c, err
		}

		var r []string
		for _, s := range c.Suggestions {
			if !strings.Contains(s, "*") {
				r = append(r, strings.TrimSpace(s))
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
		"gam":  {"g", "am"},
		"gop":  {"g", "op"},
		"gush": {"g", "ush"},
	})
}

type git struct {
	MainBranches  map[string]string
	DefaultBranch string
	changed       bool
}

func (g *git) Changed() bool {
	return g.changed
}
func (*git) Setup() []string { return nil }
func (*git) Name() string    { return "g" }

func (g *git) GetDefaultBranch(d *command.Data) string {
	if g.MainBranches == nil {
		if len(g.DefaultBranch) == 0 {
			return DefaultDefaultBranch
		}
		return g.DefaultBranch
	}
	if m, ok := g.MainBranches[repoName.Get(d)]; ok {
		return m
	}
	if len(g.DefaultBranch) == 0 {
		return DefaultDefaultBranch
	}
	return g.DefaultBranch
}

func PrefixCompleter[T any](includeUnknown bool, prefixCodes ...*regexp.Regexp) command.Completer[T] {
	return command.CompleterFromFunc(func(t T, d *command.Data) (*command.Completion, error) {
		// prefixRegex := regexp.MustCompile(prefixCode)
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
		has := map[string]bool{}
		addSuggesteion := func(s string) {
			if has[s] {
				return
			}
			has[s] = true
			suggestions = append(suggestions, s)
		}
		for _, result := range results {
			// Format is the following for tracked files
			// 1 .M ... 100644 100644 100644 e1548292489441c42682f38f2590e24d66a8587a e1548292489441c42682f38f2590e24d66a8587a sourcecontrol.go

			// Format is the following for untracked files
			// ? new_file.go

			parts := strings.Split(result, " ")

			// Check if it's an untracked file
			if parts[0] == "?" {
				file := strings.Join(parts[1:], " ")
				if includeUnknown {
					addSuggesteion(file)
				}
				continue
			}

			// if file has a space in the name, we need to rejoin int
			file := strings.Join(parts[8:], " ")
			for _, rgx := range prefixCodes {
				if rgx.MatchString(parts[1]) {
					addSuggesteion(file)
					break
				}
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
										if len(g.DefaultBranch) == 0 {
											o.Stdoutln("No global default branch set; using", DefaultDefaultBranch)
										} else {
											o.Stdoutln("Global default branch:", g.DefaultBranch)
										}

										keys := maps.Keys(g.MainBranches)
										slices.Sort(keys)
										for _, k := range keys {
											o.Stdoutf("%s: %s\n", k, g.MainBranches[k])
										}
										return nil
									}},
								),
								"set": command.SerialNodes(
									command.FlagProcessor(globalConfig),
									repoName,
									defRepoArg,
									&command.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
										g.changed = true

										if globalConfig.Get(d) {
											g.DefaultBranch = defRepoArg.Get(d)
											o.Stdoutln("Setting global default branch to", defRepoArg.Get(d))
											return nil
										}

										if g.MainBranches == nil {
											g.MainBranches = map[string]string{}
										}
										g.MainBranches[repoName.Get(d)] = defRepoArg.Get(d)
										o.Stdoutf("Setting default branch for %s to %s\n", repoName.Get(d), defRepoArg.Get(d))
										return nil
									}},
								),
								"unset": command.SerialNodes(
									command.FlagProcessor(globalConfig),
									repoName,
									&command.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
										if globalConfig.Get(d) {
											g.DefaultBranch = ""
											g.changed = true
											o.Stdoutln("Deleting global default branch")
											return nil
										}

										if g.MainBranches == nil {
											o.Stdoutln("No default branch set for this repo")
											return nil
										}
										rn := repoName.Get(d)
										if _, ok := g.MainBranches[rn]; !ok {
											o.Stdoutln("No default branch set for this repo")
											return nil
										}
										delete(g.MainBranches, rn)
										o.Stdoutln("Deleting default branch for", rn)
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
				stashArgs,
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					var r []string
					for _, c := range stashArgs.Get(d) {
						r = append(r, fmt.Sprintf("%q", c))
					}
					return []string{
						fmt.Sprintf("git stash pop %s", strings.Join(r, " ")),
					}, nil
				}),
			),
			"ush": command.SerialNodes(
				command.Description("Git stash push"),
				stashArgs,
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					var r []string
					for _, c := range stashArgs.Get(d) {
						r = append(r, fmt.Sprintf("%q", c))
					}
					return []string{
						fmt.Sprintf("git stash push %s", strings.Join(r, " ")),
					}, nil
				}),
			),

			// Complex commands
			"am": command.SerialNodes(
				command.Description("Git amend"),
				command.SimpleExecutableProcessor("git commit --amend --no-edit"),
			),
			// Git log
			"lg": command.SerialNodes(
				command.Description("Git log"),
				command.FlagProcessor(
					gitLogDiffFlag,
				),
				gitLogArg,
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					if gitLogDiffFlag.Get(d) {
						return []string{
							// Default to HEAD~1 because diffing against the same commit is just "gd" behavior.
							fmt.Sprintf("git diff HEAD~%d", gitLogArg.Get(d)),
						}, nil
					}
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
						fmt.Sprintf("git checkout %s", g.GetDefaultBranch(d)),
					}, nil
				}),
			),
			// Merge main
			"mm": command.SerialNodes(
				command.Description("Merge main"),
				repoName,
				command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					return []string{
						fmt.Sprintf("git merge %s", g.GetDefaultBranch(d)),
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
						// Replace quoted newlines with actual newlines
						strings.ReplaceAll(
							fmt.Sprintf("git commit %s-m %q", nvFlag.Get(d), strings.Join(messageArg.Get(d), " ")),
							`\n`,
							"\n",
						),
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
			/*"q": command.CacheNode(commitCacheKey, g, command.SerialNodes(
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
			),*/

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
						branch = g.GetDefaultBranch(d)
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
						fmt.Sprintf("git reset -- %s", strings.Join(ucArgs.Get(d), " ")),
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
