package sourcecontrol

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/sourcerer"
	"golang.org/x/exp/maps"
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
	return commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
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
	sshNode = commander.SerialNodes(
		commander.FunctionWrap(),
		commander.SimpleExecutableProcessor(createSSHAgentCommand),
	)
	nvFlag           = commander.BoolValueFlag("no-verify", 'n', "Whether or not to run pre-commit checks", "--no-verify ")
	formatFlag       = commander.Flag("format", 'f', "Golang format for the branch", commander.Default("%s\n"))
	parentFormatFlag = commander.Flag[string]("parent-format", 'F', "Golang format for the the parent branches")
	prefixFlag       = commander.Flag[string]("prefix", 'p', "Prefix to include if a branch is detected")
	suffixFlag       = commander.Flag[string]("suffix", 's', "Suffix to include if a branch is detected")
	ignoreNoBranch   = commander.BoolFlag("ignore-no-branch", 'i', "Ignore any errors in the git branch command")
	pushFlag         = commander.BoolFlag("push", 'p', "Whether or not to push afterwards")
	messageArg       = commander.ListArg[string]("MESSAGE", "Commit message", 1, command.UnboundedList)
	branchArg        = commander.Arg(
		"BRANCH",
		"Branch",
		BranchCompleter(),
	)
	branchesArg = commander.ListArg(
		"BRANCH",
		"Branch",
		1,
		command.UnboundedList,
		BranchesCompleter(),
	)
	mainFlag       = commander.BoolFlag("main", 'm', "Whether to diff against main branch or just local diffs")
	prevCommitFlag = commander.BoolFlag("commit", 'c', "Whether to diff against the previous commit")

	// The two dots represent [file state in the cache (e.g. added/green), file state not in the cache (red file)]
	redFileCompleter   = PrefixCompleter[[]string](true, regexp.MustCompile(`^.[^\.]$`))
	greenFileCompleter = PrefixCompleter[[]string](false, regexp.MustCompile(`^[^\.].$`))

	diffCompleter = commander.CompleterFromFunc(func(ss []string, d *command.Data) (*command.Completion, error) {

		// Get git root
		gitRootDir := &commander.ShellCommand[string]{
			CommandName: "git",
			Args: []string{
				"rev-parse",
				"--show-toplevel",
			},
			ForwardStdout: false,
			HideStderr:    true,
			// TODO: DontRunOnExecute/Usage or field for when it should run
		}
		gitRoot, err := gitRootDir.Run(nil, d)
		if err != nil {
			return nil, fmt.Errorf("failed to get git root: %v", err)
		}

		// Get diffable files
		diffableFiles := &commander.ShellCommand[[]string]{
			CommandName: "git",
			Args: []string{
				"diff",
				"--name-only",
			},
			ForwardStdout: false,
			HideStderr:    true,
		}
		files, err := diffableFiles.Run(nil, d)
		if err != nil {
			return nil, fmt.Errorf("failed to get diffable files: %v", err)
		}

		// Get absolute path for diffable files
		var absFiles []string
		for _, f := range files {
			absFiles = append(absFiles, filepath.Join(gitRoot, f))
		}

		// Create suggestions
		var suggestions []string
		pwd := commander.Getwd.Get(d)
		for _, f := range absFiles {
			relPath, err := filepath.Rel(pwd, f)
			if err != nil {
				return nil, fmt.Errorf("failed to get relative path: %v", err)
			}
			suggestions = append(suggestions, relPath)
		}

		return &command.Completion{
			Suggestions:     suggestions,
			Distinct:        true,
			CaseInsensitive: true,
		}, nil
	})
	greenFileCompleterNoDeletes = commander.ShellCommandCompleterWithOpts[[]string](&command.Completion{Distinct: true, CaseInsensitive: true}, "git", "diff", "--cached", "--name-only", "--relative")

	filesArg         = commander.ListArg[string]("FILES", "Files to add", 0, command.UnboundedList, redFileCompleter)
	allFileCompleter = PrefixCompleter[[]string](true, regexp.MustCompile(".*"))
	statusFilesArg   = commander.ListArg[string]("FILES", "Files to add", 0, command.UnboundedList, allFileCompleter)
	repoUrl          = &commander.ShellCommand[string]{
		ArgName:     "REPO",
		CommandName: "git",
		Args: []string{
			"config",
			"--get",
			"remote.origin.url",
		},
	}
	defRepoArg         = commander.Arg[string]("DEFAULT_BRANCH", "Default branch for this git repo")
	forceDelete        = commander.BoolFlag("force-delete", 'f', "force delete the branch")
	globalConfig       = commander.BoolFlag("global", 'g', "Whether or not to change the global setting")
	newBranchFlag      = commander.BoolFlag("new-branch", 'n', "Whether or not to checkout a new branch")
	whitespaceFlag     = commander.BoolValueFlag("whitespace", 'w', "Whether or not to show whitespace in diffs", "-w")
	noopWhitespaceFlag = commander.BoolFlag(whitespaceFlag.Name(), whitespaceFlag.ShortName(), "No-op so that when running add after `gd ... -w` we can keep the -w at the end", commander.Hidden[bool]())
	uaArgs             = commander.ListArg[string](
		"FILE", "Files to un-add",
		1, command.UnboundedList,
		greenFileCompleter,
	)
	diffArgs = commander.ListArg[string](
		"FILE", "Files to diff",
		0, command.UnboundedList,
		diffCompleter,
	)
	ucArgs = commander.ListArg[string](
		"FILE", "Files to un-change",
		1, command.UnboundedList,
		redFileCompleter,
	)
	gitLogArg      = commander.OptionalArg[int]("N", "Number of git logs to display", commander.NonNegative[int](), commander.Default(1))
	gitLogDiffFlag = commander.BoolFlag("diff", 'd', "Whether or not to diff the current changes against N commits prior")
	stashArgs      = commander.ListArg[string](
		"STASH_ARGS", "Args to pass to `git stash push/pop`",
		0, command.UnboundedList,
		allFileCompleter,
	)
	pushUpstreamFlag = commander.BoolFlag("upstream", 'u', "If set, push branch to upstream")
	currentBranchArg = createCurrentBranchArg(false)
)

func createCurrentBranchArg(hideStderr bool) *commander.ShellCommand[string] {
	return &commander.ShellCommand[string]{
		ArgName:     "CURRENT_BRANCH",
		CommandName: "git",
		Args: []string{
			"rev-parse",
			"--abbrev-ref",
			"HEAD",
		},
		DontRunOnComplete: true,
		HideStderr:        hideStderr,
	}
}

func CLI() *git {
	return &git{}
}

// TODO: CompleteWrapper (CompleteExtender?) here too
func branchCompleter(s string, d *command.Data) (*command.Completion, error) {
	c, err := commander.ShellCommandCompleter[string]("git", "branch", "--list").Complete(s, d)
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
}

func BranchCompleter() commander.Completer[string] {
	return commander.CompleterFromFunc[string](branchCompleter)
}

func BranchesCompleter() commander.Completer[[]string] {

	return commander.CompleterFromFunc[[]string](func(ss []string, d *command.Data) (*command.Completion, error) {
		c, err := branchCompleter(ss[len(ss)-1], d)
		if c == nil || err != nil {
			return c, err
		}

		c.Distinct = true
		return c, nil
	})
}

func GitAliasers() sourcerer.Option {
	return sourcerer.Aliasers(map[string][]string{
		"gp":  {"g", "p"},
		"gup": {"g", "up"},
		// Don't include 'gl' since that is an alias of goleep
		"gpl":  {"g", "pl"},
		"gs":   {"g", "s"},
		"guco": {"g", "uco"},
		"gb":   {"g", "b"},
		"gc":   {"g", "c"},
		"gcnv": {"g", "c", "-n"},
		"cm":   {"g", "m"},
		"gcb":  {"g", "ch"},
		"gnb":  {"g", "ch", "-n"},
		"gbn":  {"g", "ch", "-n"},
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
		"gl":   {"g", "pr-link"},
	})
}

type git struct {
	MainBranches   map[string]string
	DefaultBranch  string
	ParentBranches map[string]string
	changed        bool
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
	if m, ok := g.MainBranches[repoUrl.Get(d)]; ok {
		return m
	}
	if len(g.DefaultBranch) == 0 {
		return DefaultDefaultBranch
	}
	return g.DefaultBranch
}

func PrefixCompleter[T any](includeUnknown bool, prefixCodes ...*regexp.Regexp) commander.Completer[T] {
	return commander.CompleterFromFunc(func(t T, d *command.Data) (*command.Completion, error) {
		// prefixRegex := regexp.MustCompile(prefixCode)
		bc := &commander.ShellCommand[[]string]{
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
		addSuggestion := func(s string) {
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
					addSuggestion(file)
				}
				continue
			}

			// if file has a space in the name, we need to rejoin int
			file := strings.Join(parts[8:], " ")
			for _, rgx := range prefixCodes {
				if rgx.MatchString(parts[1]) {
					addSuggestion(file)
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
	return &commander.BranchNode{
		Branches: map[string]command.Node{
			// Configs
			"cfg": commander.SerialNodes(
				commander.Description("Config settings"),
				&commander.BranchNode{
					Branches: map[string]command.Node{
						"main": &commander.BranchNode{
							Branches: map[string]command.Node{
								"show": commander.SerialNodes(
									&commander.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
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
								"set": commander.SerialNodes(
									commander.FlagProcessor(globalConfig),
									repoUrl,
									defRepoArg,
									&commander.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
										g.changed = true

										if globalConfig.Get(d) {
											g.DefaultBranch = defRepoArg.Get(d)
											o.Stdoutln("Setting global default branch to", defRepoArg.Get(d))
											return nil
										}

										if g.MainBranches == nil {
											g.MainBranches = map[string]string{}
										}
										g.MainBranches[repoUrl.Get(d)] = defRepoArg.Get(d)
										o.Stdoutf("Setting default branch for %s to %s\n", repoUrl.Get(d), defRepoArg.Get(d))
										return nil
									}},
								),
								"unset": commander.SerialNodes(
									commander.FlagProcessor(globalConfig),
									repoUrl,
									&commander.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
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
										rn := repoUrl.Get(d)
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
			"b": commander.SerialNodes(
				commander.Description("Branch"),
				commander.SimpleExecutableProcessor("git branch"),
			),
			"current": commander.SerialNodes(
				commander.Description("Display current branch"),
				commander.FlagProcessor(
					formatFlag,
					ignoreNoBranch,
					parentFormatFlag,
					prefixFlag,
					suffixFlag,
				),
				commander.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
					cba := createCurrentBranchArg(true)
					branch, err := cba.Run(o, d)
					if err != nil {
						if ignoreNoBranch.Get(d) {
							return nil
						}
						return o.Err(err)
					}

					output := []string{
						prefixFlag.GetOrDefault(d, ""),
					}

					if parentFormatFlag.Provided(d) {
						contains := map[string]bool{
							branch: true,
						}
						var branchPath []string
						for parent, ok := g.ParentBranches[branch]; ok; parent, ok = g.ParentBranches[parent] {
							if contains[parent] {
								return o.Stderrln("cycle detected in parent branches")
							}
							contains[parent] = true
							branchPath = append(branchPath, parent)
						}
						slices.Reverse(branchPath)
						for _, parent := range branchPath {
							output = append(output, fmt.Sprintf(parentFormatFlag.Get(d), parent))
						}
					}
					output = append(output, fmt.Sprintf(formatFlag.Get(d), branch), suffixFlag.GetOrDefault(d, ""))
					o.Stdout(strings.Join(output, ""))
					return nil
				}, nil),
			),
			"l": commander.SerialNodes(
				commander.Description("Pull"),
				sshNode,
				commander.SimpleExecutableProcessor(
					"git pull",
				),
			),
			// upstream push with pr link
			"up": commander.SerialNodes(
				commander.Description("Push upstream and output PR link"),
				currentBranchArg,
				repoUrl,

				// git push upstream
				&commander.ExecutorProcessor{func(o command.Output, d *command.Data) error {
					sc := &commander.ShellCommand[string]{
						CommandName: "git",
						Args: []string{
							"push",
							"--set-upstream",
							"origin",
							currentBranchArg.Get(d),
						},
						HideStderr: true,
					}
					if _, err := sc.Run(o, d); err != nil {
						return o.Annotatef(err, "failed to run git push")
					}

					return g.printPRLink(o, d)
				}},
			),
			"p": commander.SerialNodes(
				commander.Description("Push"),
				commander.FlagProcessor(pushUpstreamFlag),
				commander.IfData(pushUpstreamFlag.Name(), currentBranchArg),
				sshNode,
				commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					if pushUpstreamFlag.Get(d) {
						pushCmd := fmt.Sprintf("git push --set-upstream origin %q", currentBranchArg.Get(d))
						o.Stdoutln(pushCmd)
						return []string{pushCmd}, nil
					}
					return []string{"git push"}, nil
				}),
			),
			"pp": commander.SerialNodes(
				commander.Description("Pull and push"),
				sshNode,
				executableJoinByOS(
					"git pull",
					"git push",
				),
				commander.SimpleExecutableProcessor(),
			),
			"sh": commander.SerialNodes(
				commander.Description("Create ssh-agent"),
				sshNode,
			),
			"uco": commander.SerialNodes(
				commander.Description("Undo commit"),
				commander.SimpleExecutableProcessor("git reset HEAD~"),
			),
			"f": commander.SerialNodes(
				commander.Description("Git fetch"),
				commander.SimpleExecutableProcessor("git fetch"),
			),
			"op": commander.SerialNodes(
				commander.Description("Git stash pop"),
				stashArgs,
				commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					var r []string
					for _, c := range stashArgs.Get(d) {
						r = append(r, fmt.Sprintf("%q", c))
					}
					return []string{
						fmt.Sprintf("git stash pop %s", strings.Join(r, " ")),
					}, nil
				}),
			),
			"ush": commander.SerialNodes(
				commander.Description("Git stash push"),
				stashArgs,
				commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
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
			"am": commander.SerialNodes(
				commander.Description("Git amend"),
				commander.SimpleExecutableProcessor("git commit --amend --no-edit"),
			),
			// Git log
			"lg": commander.SerialNodes(
				commander.Description("Git log"),
				commander.FlagProcessor(
					gitLogDiffFlag,
					whitespaceFlag,
				),
				gitLogArg,
				commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					if gitLogDiffFlag.Get(d) {
						return []string{
							fmt.Sprintf("git diff HEAD~%d %v", gitLogArg.Get(d), whitespaceFlag.Get(d)),
						}, nil
					}
					return []string{
						fmt.Sprintf("git log -n %d", gitLogArg.Get(d)),
					}, nil
				}),
			),
			// Checkout main
			"m": commander.SerialNodes(
				commander.Description("Checkout main"),
				repoUrl,
				commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					return []string{
						fmt.Sprintf("git checkout %s", g.GetDefaultBranch(d)),
					}, nil
				}),
			),
			// Merge main
			"mm": commander.SerialNodes(
				commander.Description("Merge main"),
				repoUrl,
				commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					return []string{
						fmt.Sprintf("git merge %s", g.GetDefaultBranch(d)),
					}, nil
				}),
			),
			// Commit
			"c": commander.SerialNodes(
				commander.Description("Commit"),
				commander.FlagProcessor(
					nvFlag,
					pushFlag,
				),
				messageArg,
				commander.If(
					sshNode,
					func(i *command.Input, d *command.Data) bool {
						return pushFlag.Get(d)
					},
				),
				commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
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
			"cp": commander.SerialNodes(
				commander.Description("Commit and push"),
				commander.FlagProcessor(
					nvFlag,
				),
				messageArg,
				sshNode,
				commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					return joinByOS(
						fmt.Sprintf("git commit %s-m %q", nvFlag.Get(d), strings.Join(messageArg.Get(d), " ")),
						"git push",
						"echo Success!",
					)
				}),
			),

			// Squash
			/*"q": command.CacheNode(commitCacheKey, g, commander.SerialNodes(
				commander.Description("Squash local commits"),
				commander.FlagProcessor(
					nvFlag,
					pushFlag,
				),
				messageArg,
				commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
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
			"pr-link": commander.SerialNodes(
				commander.Description("Get PR link"),
				currentBranchArg,
				repoUrl,
				commander.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
					return g.printPRLink(o, d)
				}, nil),
			),
			"ch": commander.SerialNodes(
				commander.Description("Checkout new branch"),
				commander.FlagProcessor(
					newBranchFlag,
				),
				currentBranchArg,
				branchArg,
				commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {

					branchName := branchArg.Get(d)

					flag := ""
					if newBranchFlag.Get(d) {
						flag = "-b "
						if g.ParentBranches == nil {
							g.ParentBranches = map[string]string{}
						}
						g.ParentBranches[branchName] = currentBranchArg.Get(d)
						g.changed = true
					}
					return []string{
						fmt.Sprintf("git checkout %s%s", flag, branchName),
					}, nil
				}),
			),

			// Delete branch
			"bd": commander.SerialNodes(
				commander.Description("Delete branch"),
				commander.FlagProcessor(forceDelete),
				branchesArg,
				commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					flag := "-d"
					if forceDelete.Get(d) {
						flag = "-D"
					}

					var branches []string
					for _, b := range branchesArg.Get(d) {
						branches = append(branches, fmt.Sprintf("%q", b))

						if g.ParentBranches != nil {
							if _, ok := g.ParentBranches[b]; ok {
								delete(g.ParentBranches, b)
								g.changed = true
							}
						}
					}

					return []string{
						fmt.Sprintf("git branch %s %s", flag, strings.Join(branches, " ")),
					}, nil
				}),
			),

			// Diff
			"d": commander.SerialNodes(
				commander.Description("Diff"),
				commander.Getwd,
				commander.FlagProcessor(
					mainFlag,
					prevCommitFlag,
					whitespaceFlag,
				),
				diffArgs,
				repoUrl,
				commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
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
			"uc": commander.SerialNodes(
				commander.Description("Undo change"),
				ucArgs,
				commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					return []string{
						fmt.Sprintf("git checkout -- %s", strings.Join(ucArgs.Get(d), " ")),
					}, nil
				}),
			),

			// Undo add
			"ua": commander.SerialNodes(
				commander.Description("Undo add"),
				uaArgs,
				commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					return []string{
						fmt.Sprintf("git reset -- %s", strings.Join(ucArgs.Get(d), " ")),
					}, nil
				}),
			),

			// Status
			"s": commander.SerialNodes(
				commander.Description("Status"),
				statusFilesArg,
				commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					return []string{fmt.Sprintf("git status %s", strings.Join(statusFilesArg.Get(d), " "))}, nil
				}),
			),

			// Add
			"a": commander.SerialNodes(
				commander.FlagProcessor(
					noopWhitespaceFlag,
				),
				commander.Description("Add"),
				filesArg,
				commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					fs := filesArg.Get(d)
					if len(fs) == 0 {
						return []string{"git add ."}, nil
					}
					return []string{fmt.Sprintf("git add %s", strings.Join(fs, " "))}, nil
				}),
			),

			// Rebase
			"rb": &commander.BranchNode{
				Branches: map[string]command.Node{
					"a": commander.SerialNodes(
						commander.Description("Abort"),
						commander.SimpleExecutableProcessor("git rebase --abort"),
						commander.EchoExecuteData(),
					),
					"c": commander.SerialNodes(
						commander.Description("Continue"),
						commander.SimpleExecutableProcessor("git rebase --continue"),
						commander.EchoExecuteData(),
					),
				},
			},
		},
		Synonyms: commander.BranchSynonyms(map[string][]string{
			"l": {"pl"},
		}),
	}
}

func (g *git) printPRLink(o command.Output, d *command.Data) error {
	url := repoUrl.Get(d)
	orgRepo := strings.TrimSuffix(url, ".git")
	if strings.Contains(url, "git@") {
		orgRepo = strings.TrimPrefix(orgRepo, "git@github.com:")
	} else if strings.Contains(url, "https:") {
		orgRepo = strings.TrimPrefix(orgRepo, "https://github.com/")
	} else {
		return o.Stderrf("Unknown git url format: %s\n", url)
	}

	cb := currentBranchArg.Get(d)

	if pb, ok := g.ParentBranches[cb]; ok {
		o.Stdoutf("https://github.com/%s/compare/%s...%s?expand=1\n", orgRepo, pb, cb)
		return nil
	} else if mb, ok := g.MainBranches[url]; ok {
		o.Stdoutf("https://github.com/%s/compare/%s...%s?expand=1\n", orgRepo, mb, cb)
		return nil
	} else {
		return o.Stderrf("Unknown parent branch for branch %s; and no default main branch set\n", cb)
	}
}
