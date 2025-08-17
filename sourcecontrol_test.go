package sourcecontrol

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/commandertest"
	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/sourcerer"
	"github.com/leep-frog/functional"
	"golang.org/x/exp/slices"
)

func repoRunContents() *commandtest.RunContents {
	return &commandtest.RunContents{
		Name: "git",
		Args: []string{
			"config",
			"--get",
			"remote.origin.url",
		},
	}
}

func TestExecution(t *testing.T) {
	type osCheck struct {
		wantExecutable []string
		wantStdout     []string
	}

	u := strings.Join([]string{
		`┳ --dry-run|-y`,
		`┃`,
		`┃   Add`,
		`┣━━ a [ FILES ... ]`,
		`┃`,
		`┃   Git amend`,
		`┣━━ am`,
		`┃`,
		`┃   Branch`,
		`┣━━ b`,
		`┃`,
		`┃   Delete branch`,
		`┣━━ bd BRANCH [ BRANCH ... ] --force-delete|-f`,
		`┃`,
		`┃   Commit`,
		`┣━━ c MESSAGE [ MESSAGE ... ] --no-verify|-n --push|-p`,
		`┃`,
		`┃   Config settings`,
		`┣━━ cfg ┓`,
		`┃   ┏━━━┛`,
		`┃   ┃`,
		`┃   ┗━━ main ┓`,
		`┃       ┏━━━━┛`,
		`┃       ┃`,
		`┃       ┣━━ set DEFAULT_BRANCH --global|-g`,
		`┃       ┃`,
		`┃       ┣━━ show`,
		`┃       ┃`,
		`┃       ┗━━ unset --global|-g`,
		`┃`,
		`┃   Checkout new branch`,
		`┣━━ ch BRANCH --new-branch|-n`,
		`┃`,
		`┃   Commit and push`,
		`┣━━ cp MESSAGE [ MESSAGE ... ] --no-verify|-n`,
		`┃`,
		`┃   Display current branch`,
		`┣━━ current --format|-f FORMAT --ignore-no-branch|-i --parent-format|-F PARENT_FORMAT --prefix|-p PREFIX --suffix|-s SUFFIX`,
		`┃`,
		`┃   Diff`,
		`┣━━ d [ FILE ... ] --main|-m --commit|-c --whitespace|-w --add|-a`,
		`┃`,
		`┃   End a branch after it has been merged`,
		`┣━━ end`,
		`┃`,
		`┃   Git fetch`,
		`┣━━ f`,
		`┃`,
		`┃   Pull`,
		`┣━━ [l|pl]`,
		`┃`,
		`┃   Git log`,
		`┣━━ lg [ N ] --diff|-d --whitespace|-w`,
		`┃`,
		`┃   Checkout main`,
		`┣━━ m`,
		`┃`,
		`┃   Merge main`,
		`┣━━ mm`,
		`┃`,
		`┃   Git stash pop`,
		`┣━━ op [ STASH_ARGS ... ]`,
		`┃`,
		`┃   Push`,
		`┣━━ p --upstream|-u`,
		`┃`,
		`┃   Checkout previous branch`,
		`┣━━ pb`,
		`┃`,
		`┃   Pull and push`,
		`┣━━ pp`,
		`┃`,
		`┃   Get PR link`,
		`┣━━ pr-link`,
		`┃`,
		`┣━━ rb ┓`,
		`┃   ┏━━┛`,
		`┃   ┃`,
		`┃   ┃   Abort`,
		`┃   ┣━━ a`,
		`┃   ┃`,
		`┃   ┃   Continue`,
		`┃   ┗━━ c`,
		`┃`,
		`┃   Remove`,
		`┣━━ rm FILES [ FILES ... ]`,
		`┃`,
		`┃   Status`,
		`┣━━ s [ FILES ... ]`,
		`┃`,
		`┃   Create ssh-agent`,
		`┣━━ sh`,
		`┃`,
		`┃   Undo add`,
		`┣━━ ua [ FILE ... ]`,
		`┃`,
		`┃   Undo change`,
		`┣━━ uc FILE [ FILE ... ]`,
		`┃`,
		`┃   Undo commit`,
		`┣━━ uco`,
		`┃`,
		`┃   Push upstream and output PR link`,
		`┣━━ up`,
		`┃`,
		`┃   Git stash push`,
		`┗━━ ush [ STASH_ARGS ... ]`,
		``,
		`Arguments:`,
		`  BRANCH: Branch`,
		`  DEFAULT_BRANCH: Default branch for this git repo`,
		`  FILE: Files to un-change`,
		`  FILES: Files to add`,
		`  MESSAGE: Commit message`,
		`  N: Number of git logs to display`,
		`    Default: 1`,
		`    NonNegative()`,
		"  STASH_ARGS: Args to pass to `git stash push/pop`",
		``,
		`Flags:`,
		`  [a] add: If set, then files will be added`,
		`  [c] commit: Whether to diff against the previous commit`,
		`  [d] diff: Whether or not to diff the current changes against N commits prior`,
		`  [y] dry-run: Dry-run mode`,
		`  [f] force-delete: force delete the branch`,
		`  [f] format: Golang format for the branch`,
		`    Default: %s`,
		``, // Default is %s\n, so need this newline
		`  [g] global: Whether or not to change the global setting`,
		`  [i] ignore-no-branch: Ignore any errors in the git branch command`,
		`  [m] main: Whether to diff against main branch or just local diffs`,
		`  [n] new-branch: Whether or not to checkout a new branch`,
		`  [n] no-verify: Whether or not to run pre-commit checks`,
		`  [F] parent-format: Golang format for the the parent branches`,
		`  [p] prefix: Prefix to include if a branch is detected`,
		`  [p] push: Whether or not to push afterwards`,
		`  [s] suffix: Suffix to include if a branch is detected`,
		`  [u] upstream: If set, push branch to upstream`,
		`  [w] whitespace: Whether or not to show whitespace in diffs`,
	}, "\n")
	_ = u

	for _, curOS := range []sourcerer.OS{sourcerer.Linux(), sourcerer.Windows()} {
		for _, test := range []struct {
			name     string
			g        *git
			want     *git
			etc      *commandtest.ExecuteTestCase
			osChecks map[string]*osCheck
		}{
			// TODO: Config tests
			// Simple command tests
			{
				name: "branch",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"b"},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{"git branch"},
					},
				},
			},
			{
				name: "pull",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"l"},
					WantExecuteData: &command.ExecuteData{
						FunctionWrap: true,
						Executable: []string{
							createSSHAgentCommand,
							"git pull",
						},
					},
				},
			},
			{
				name: "fetch",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"f"},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git fetch",
						},
					},
				},
			},
			// Git amend
			{
				name: "git amend succeeds",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"am"},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git commit --amend --no-edit`,
						},
					},
				},
			},
			// Git log
			{
				name: "git log with no args",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"lg"},
					WantData: &command.Data{Values: map[string]interface{}{
						gitLogArg.Name(): 1,
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git log -n 1",
						},
					},
				},
			},
			{
				name: "git log with arg",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"lg", "4"},
					WantData: &command.Data{Values: map[string]interface{}{
						gitLogArg.Name(): 4,
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git log -n 4",
						},
					},
				},
			},
			{
				name: "git log diff with no args",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"lg", "-d"},
					WantData: &command.Data{Values: map[string]interface{}{
						gitLogArg.Name():      1,
						gitLogDiffFlag.Name(): true,
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git diff HEAD~1 ",
						},
					},
				},
			},
			{
				name: "git log diff with args",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"lg", "-d", "7"},
					WantData: &command.Data{Values: map[string]interface{}{
						gitLogArg.Name():      7,
						gitLogDiffFlag.Name(): true,
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git diff HEAD~7 ",
						},
					},
				},
			},
			{
				name: "git log diff with no args and whitespace flag",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"lg", "-d", "-w"},
					WantData: &command.Data{Values: map[string]interface{}{
						gitLogArg.Name():      1,
						gitLogDiffFlag.Name(): true,
						whitespaceFlag.Name(): "-w",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git diff HEAD~1 -w",
						},
					},
				},
			},
			{
				name: "git log diff with args and whitespace flag",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"lg", "-d", "7", "-w"},
					WantData: &command.Data{Values: map[string]interface{}{
						gitLogArg.Name():      7,
						gitLogDiffFlag.Name(): true,
						whitespaceFlag.Name(): "-w",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git diff HEAD~7 -w",
						},
					},
				},
			},
			// Git stash push/pop
			{
				name: "git stash push with no args",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ush"},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git stash push ",
						},
					},
				},
			},
			{
				name: "git stash pop with no args",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"op"},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git stash pop ",
						},
					},
				},
			},
			{
				name: "git stash push with args",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ush", "abc", "123"},
					WantData: &command.Data{Values: map[string]interface{}{
						stashArgs.Name(): []string{"abc", "123"},
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git stash push "abc" "123"`,
						},
					},
				},
			},
			{
				name: "git stash pop with args",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"op", "def", "456"},
					WantData: &command.Data{Values: map[string]interface{}{
						stashArgs.Name(): []string{"def", "456"},
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git stash pop "def" "456"`,
						},
					},
				},
			},
			// Previous branch
			{
				name: "previous branch requires git root",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"pb"},
					WantRunContents: []*commandtest.RunContents{
						{
							Name: "git",
							Args: []string{"rev-parse", "--show-toplevel"},
						},
					},
					RunResponses: []*commandtest.FakeRun{
						{
							Err: fmt.Errorf("oh no"),
						},
					},
					WantErr:    fmt.Errorf("failed to execute shell command: oh no"),
					WantStderr: "failed to execute shell command: oh no\n",
				},
			},
			{
				name: "previous branch requires current branch",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"pb"},
					WantRunContents: []*commandtest.RunContents{
						{
							Name: "git",
							Args: []string{"rev-parse", "--show-toplevel"},
						},
						{
							Name: "git",
							Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
						},
					},
					RunResponses: []*commandtest.FakeRun{
						{
							Stdout: []string{"/some/git/root"},
						},
						{
							Err: fmt.Errorf("whoops"),
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName: "/some/git/root",
					}},
					WantErr:    fmt.Errorf("failed to execute shell command: whoops"),
					WantStderr: "failed to execute shell command: whoops\n",
				},
			},
			{
				name: "previous branch fails if no previous branch",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"pb"},
					WantRunContents: []*commandtest.RunContents{
						{
							Name: "git",
							Args: []string{"rev-parse", "--show-toplevel"},
						},
						{
							Name: "git",
							Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
						},
					},
					RunResponses: []*commandtest.FakeRun{
						{
							Stdout: []string{"/some/git/root"},
						},
						{
							Stdout: []string{"current-branch"},
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName:       "/some/git/root",
						currentBranchArg.ArgName: "current-branch",
					}},
					WantErr:    fmt.Errorf("no previous branch exists"),
					WantStderr: "no previous branch exists\n",
				},
			},
			{
				name: "previous branch works",
				g: &git{
					PreviousBranches: map[string]string{
						"/some/git/root":       "old-branch",
						"/some/git/other-root": "other-branch",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"pb"},
					WantRunContents: []*commandtest.RunContents{
						{
							Name: "git",
							Args: []string{"rev-parse", "--show-toplevel"},
						},
						{
							Name: "git",
							Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
						},
					},
					RunResponses: []*commandtest.FakeRun{
						{
							Stdout: []string{"/some/git/root"},
						},
						{
							Stdout: []string{"current-branch"},
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName:       "/some/git/root",
						currentBranchArg.ArgName: "current-branch",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git checkout old-branch`,
						},
					},
				},
				want: &git{
					PreviousBranches: map[string]string{
						"/some/git/root":       "current-branch",
						"/some/git/other-root": "other-branch",
					},
				},
			},
			// Checkout main
			{
				name: "checkout main",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"m"},
					RunResponses: []*commandtest.FakeRun{
						{Stdout: []string{"/git/root"}},
						{Stdout: []string{"current-branch"}},
						{Stdout: []string{"test-repo"}},
					},
					WantRunContents: []*commandtest.RunContents{
						{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}},
						{Name: "git", Args: []string{"rev-parse", "--abbrev-ref", "HEAD"}},
						repoRunContents(),
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName:       "/git/root",
						currentBranchArg.ArgName: "current-branch",
						repoUrl.Name():           "test-repo",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git checkout main",
						},
					},
				},
				want: &git{
					PreviousBranches: map[string]string{
						"/git/root": "current-branch",
					},
				},
			},
			{
				name: "checkout main if MainBranches defined",
				g: &git{
					MainBranches: map[string]string{},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"m"},
					RunResponses: []*commandtest.FakeRun{
						{Stdout: []string{"/git/root"}},
						{Stdout: []string{"current-branch"}},
						{Stdout: []string{"test-repo"}},
					},
					WantRunContents: []*commandtest.RunContents{
						{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}},
						{Name: "git", Args: []string{"rev-parse", "--abbrev-ref", "HEAD"}},
						repoRunContents(),
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName:       "/git/root",
						currentBranchArg.ArgName: "current-branch",
						repoUrl.Name():           "test-repo",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git checkout main",
						},
					},
				},
				want: &git{
					MainBranches: map[string]string{},
					PreviousBranches: map[string]string{
						"/git/root": "current-branch",
					},
				},
			},
			{
				name: "checkout main uses default branch for unknown repo",
				g: &git{
					DefaultBranch: "mainer",
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"m"},
					RunResponses: []*commandtest.FakeRun{
						{Stdout: []string{"/git/root"}},
						{Stdout: []string{"current-branch"}},
						{Stdout: []string{"test-repo"}},
					},
					WantRunContents: []*commandtest.RunContents{
						{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}},
						{Name: "git", Args: []string{"rev-parse", "--abbrev-ref", "HEAD"}},
						repoRunContents(),
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName:       "/git/root",
						currentBranchArg.ArgName: "current-branch",
						repoUrl.Name():           "test-repo",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git checkout mainer",
						},
					},
				},
				want: &git{
					DefaultBranch: "mainer",
					PreviousBranches: map[string]string{
						"/git/root": "current-branch",
					},
				},
			},
			{
				name: "checkout main uses default branch for unknown repo with MainBranches defined",
				g: &git{
					MainBranches:  map[string]string{},
					DefaultBranch: "mainer",
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"m"},
					RunResponses: []*commandtest.FakeRun{
						{Stdout: []string{"/git/root"}},
						{Stdout: []string{"current-branch"}},
						{Stdout: []string{"test-repo"}},
					},
					WantRunContents: []*commandtest.RunContents{
						{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}},
						{Name: "git", Args: []string{"rev-parse", "--abbrev-ref", "HEAD"}},
						repoRunContents(),
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName:       "/git/root",
						currentBranchArg.ArgName: "current-branch",
						repoUrl.Name():           "test-repo",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git checkout mainer",
						},
					},
				},
				want: &git{
					MainBranches:  map[string]string{},
					DefaultBranch: "mainer",
					PreviousBranches: map[string]string{
						"/git/root": "current-branch",
					},
				},
			},
			{
				name: "checkout main uses configured default branch for known repo",
				g: &git{
					DefaultBranch: "mainer",
					MainBranches: map[string]string{
						"test-repo": "mainest",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"m"},
					RunResponses: []*commandtest.FakeRun{
						{Stdout: []string{"/git/root"}},
						{Stdout: []string{"current-branch"}},
						{Stdout: []string{"test-repo"}},
					},
					WantRunContents: []*commandtest.RunContents{
						{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}},
						{Name: "git", Args: []string{"rev-parse", "--abbrev-ref", "HEAD"}},
						repoRunContents(),
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName:       "/git/root",
						currentBranchArg.ArgName: "current-branch",
						repoUrl.Name():           "test-repo",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git checkout mainest",
						},
					},
				},
				want: &git{
					DefaultBranch: "mainer",
					MainBranches: map[string]string{
						"test-repo": "mainest",
					},
					PreviousBranches: map[string]string{
						"/git/root": "current-branch",
					},
				},
			},
			// Merge main
			{
				name: "merge main",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"mm"},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*commandtest.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoUrl.Name(): "test-repo",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git merge main",
						},
					},
				},
			},
			{
				name: "merge main uses default branch for unknown repo",
				g: &git{
					DefaultBranch: "mainer",
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"mm"},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*commandtest.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoUrl.Name(): "test-repo",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git merge mainer",
						},
					},
				},
			},
			{
				name: "merge main uses configured default branch for known repo",
				g: &git{
					DefaultBranch: "mainer",
					MainBranches: map[string]string{
						"test-repo": "mainest",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"mm"},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*commandtest.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoUrl.Name(): "test-repo",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git merge mainest",
						},
					},
				},
			},
			// Push and pull
			{
				name: "push",
				etc: &commandtest.ExecuteTestCase{
					Args:            []string{"p"},
					WantExecuteData: &command.ExecuteData{Executable: []string{"", "git push"}, FunctionWrap: true},
				},
			},
			{
				name: "push upstream fails if can't get branch",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"p", "--upstream"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
						Err:    fmt.Errorf("failed to get some-branch"),
					}},
					WantStderr: "failed to execute shell command: failed to get some-branch\n",
					WantErr:    fmt.Errorf("failed to execute shell command: failed to get some-branch"),
					WantData: &command.Data{Values: map[string]interface{}{
						pushUpstreamFlag.Name(): true,
					}},
				},
			},
			{
				name: "push upstream succeeds",
				etc: &commandtest.ExecuteTestCase{
					Args:            []string{"p", "--upstream"},
					WantExecuteData: &command.ExecuteData{Executable: []string{"", `git push --set-upstream origin "some-branch"`}, FunctionWrap: true},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						pushUpstreamFlag.Name(): true,
						"CURRENT_BRANCH":        "some-branch",
					}},
					WantStdout: "git push --set-upstream origin \"some-branch\"\n",
				},
			},
			{
				name: "pull",
				etc: &commandtest.ExecuteTestCase{
					Args:            []string{"l"},
					WantExecuteData: &command.ExecuteData{Executable: []string{"", "git pull"}, FunctionWrap: true},
				},
			},
			{
				name: "pull and push",
				osChecks: map[string]*osCheck{
					"windows": {
						wantExecutable: []string{
							"",
							wCmd("git pull"),
							wCmd("git push"),
						},
					},
					"linux": {
						wantExecutable: []string{
							"",
							"git pull && git push",
						},
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args:            []string{"pp"},
					WantExecuteData: &command.ExecuteData{FunctionWrap: true},
				},
			},
			// Commit
			{
				name: "commit requires args",
				etc: &commandtest.ExecuteTestCase{
					Args:       []string{"c"},
					WantStderr: "Argument \"MESSAGE\" requires at least 1 argument, got 0\n",
					WantErr:    fmt.Errorf(`Argument "MESSAGE" requires at least 1 argument, got 0`),
				},
			},
			{
				name: "simple commit",
				osChecks: map[string]*osCheck{
					"windows": {
						wantExecutable: []string{
							wCmd(`git commit -m "did things"`),
							wCmd("echo Success!"),
						},
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"c", "did", "things"},
					WantData: &command.Data{Values: map[string]interface{}{
						messageArg.Name(): []string{"did", "things"},
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git commit -m "did things" && echo Success!`,
						},
					},
				},
			},
			{
				name: "commit no verify",
				osChecks: map[string]*osCheck{
					"windows": {
						wantExecutable: []string{
							wCmd(`git commit --no-verify -m "did things"`),
							wCmd("echo Success!"),
						},
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"c", "did", "things", "-n"},
					WantData: &command.Data{Values: map[string]interface{}{
						messageArg.Name(): []string{"did", "things"},
						nvFlag.Name():     nvFlag.TrueValue(),
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git commit --no-verify -m "did things" && echo Success!`,
						},
					},
				},
			},
			{
				name: "commit push",
				osChecks: map[string]*osCheck{
					"windows": {
						wantExecutable: []string{
							createSSHAgentCommand,
							wCmd(`git commit -m "did things"`),
							wCmd(`git push`),
							wCmd("echo Success!"),
						},
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"c", "did", "things", "-p"},
					WantData: &command.Data{Values: map[string]interface{}{
						messageArg.Name(): []string{"did", "things"},
						pushFlag.Name():   true,
					}},
					WantExecuteData: &command.ExecuteData{
						FunctionWrap: true,
						Executable: []string{
							createSSHAgentCommand,
							`git commit -m "did things" && git push && echo Success!`,
						},
					},
				},
			},
			{
				name: "commit no verify and push",
				osChecks: map[string]*osCheck{
					"windows": {
						wantExecutable: []string{
							createSSHAgentCommand,
							wCmd(`git commit --no-verify -m "did things"`),
							wCmd(`git push`),
							wCmd("echo Success!"),
						},
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"c", "did", "things", "--no-verify", "--push"},
					WantData: &command.Data{Values: map[string]interface{}{
						messageArg.Name(): []string{"did", "things"},
						nvFlag.Name():     nvFlag.TrueValue(),
						pushFlag.Name():   true,
					}},
					WantExecuteData: &command.ExecuteData{
						FunctionWrap: true,
						Executable: []string{
							createSSHAgentCommand,
							`git commit --no-verify -m "did things" && git push && echo Success!`,
						},
					},
				},
			},
			{
				name: "commit no verify and push as multi-flag",
				osChecks: map[string]*osCheck{
					"windows": {
						wantExecutable: []string{
							createSSHAgentCommand,
							wCmd(`git commit --no-verify -m "did things"`),
							wCmd(`git push`),
							wCmd("echo Success!"),
						},
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"c", "-np", "did", "things"},
					WantData: &command.Data{Values: map[string]interface{}{
						messageArg.Name(): []string{"did", "things"},
						nvFlag.Name():     nvFlag.TrueValue(),
						pushFlag.Name():   true,
					}},
					WantExecuteData: &command.ExecuteData{
						FunctionWrap: true,
						Executable: []string{
							createSSHAgentCommand,
							`git commit --no-verify -m "did things" && git push && echo Success!`,
						},
					},
				},
			},
			{
				name: "commit message with newlines",
				osChecks: map[string]*osCheck{
					"windows": {
						wantExecutable: []string{
							wCmd(strings.Join([]string{
								`git commit -m "did`,
								`things and`,
								``,
								`other things too"`,
							}, "\n")),
							wCmd("echo Success!"),
						},
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"c", "did\nthings", "and\n\nother things too"},
					WantData: &command.Data{Values: map[string]interface{}{
						messageArg.Name(): []string{"did\nthings", "and\n\nother things too"},
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							strings.Join([]string{
								`git commit -m "did`,
								`things and`,
								``,
								`other things too" && echo Success!`,
							}, "\n"),
						},
					},
				},
			},
			// Commit & push
			{
				name: "commit and push requires args",
				etc: &commandtest.ExecuteTestCase{
					Args:       []string{"cp"},
					WantStderr: "Argument \"MESSAGE\" requires at least 1 argument, got 0\n",
					WantErr:    fmt.Errorf(`Argument "MESSAGE" requires at least 1 argument, got 0`),
				},
			},
			{
				name: "simple commit and push",
				osChecks: map[string]*osCheck{
					"windows": {
						wantExecutable: []string{
							createSSHAgentCommand,
							wCmd(`git commit -m "did things"`),
							wCmd(`git push`),
							wCmd("echo Success!"),
						},
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"cp", "did", "things"},
					WantData: &command.Data{Values: map[string]interface{}{
						messageArg.Name(): []string{"did", "things"},
					}},
					WantExecuteData: &command.ExecuteData{
						FunctionWrap: true,
						Executable: []string{
							createSSHAgentCommand,
							`git commit -m "did things" && git push && echo Success!`,
						},
					},
				},
			},
			{
				name: "commit and push no verify",
				osChecks: map[string]*osCheck{
					"windows": {
						wantExecutable: []string{
							createSSHAgentCommand,
							wCmd(`git commit --no-verify -m "did things"`),
							wCmd(`git push`),
							wCmd("echo Success!"),
						},
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"cp", "did", "things", "-n"},
					WantData: &command.Data{Values: map[string]interface{}{
						messageArg.Name(): []string{"did", "things"},
						nvFlag.Name():     nvFlag.TrueValue(),
					}},
					WantExecuteData: &command.ExecuteData{
						FunctionWrap: true,
						Executable: []string{
							createSSHAgentCommand,
							`git commit --no-verify -m "did things" && git push && echo Success!`,
						},
					},
				},
			},
			// Checkout new branch
			{
				name: "checkout branch requires git root",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ch"},
					WantRunContents: []*commandtest.RunContents{
						{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}},
					},
					RunResponses: []*commandtest.FakeRun{
						{Err: fmt.Errorf("oh no")},
					},
					WantStderr: "failed to execute shell command: oh no\n",
					WantErr:    fmt.Errorf(`failed to execute shell command: oh no`),
				},
			},
			{
				name: "checkout branch requires current branch",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ch"},
					WantRunContents: []*commandtest.RunContents{
						{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}},
						{Name: "git", Args: []string{"rev-parse", "--abbrev-ref", "HEAD"}},
					},
					RunResponses: []*commandtest.FakeRun{
						{Stdout: []string{"/git/root"}},
						{Err: fmt.Errorf("whoops")},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName: "/git/root",
					}},
					WantStderr: "failed to execute shell command: whoops\n",
					WantErr:    fmt.Errorf(`failed to execute shell command: whoops`),
				},
			},
			{
				name: "checkout branch requires arg",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ch"},
					WantRunContents: []*commandtest.RunContents{
						{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}},
						{Name: "git", Args: []string{"rev-parse", "--abbrev-ref", "HEAD"}},
					},
					RunResponses: []*commandtest.FakeRun{
						{Stdout: []string{"/git/root"}},
						{Stdout: []string{"some-branch"}},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName:       "/git/root",
						currentBranchArg.ArgName: "some-branch",
						userArg.Name:             "person",
					}},
					WantStderr: "Argument \"BRANCH\" requires at least 1 argument, got 0\n",
					WantErr:    fmt.Errorf(`Argument "BRANCH" requires at least 1 argument, got 0`),
				},
			},
			{
				name: "checkout branch requires one arg",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ch", "tree", "limb"},
					WantRunContents: []*commandtest.RunContents{
						{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}},
						{Name: "git", Args: []string{"rev-parse", "--abbrev-ref", "HEAD"}},
						{Name: "git", Args: []string{"branch", "--list"}},
					},
					RunResponses: []*commandtest.FakeRun{
						{Stdout: []string{"/git/root"}},
						{Stdout: []string{"some-branch"}},
						{Stdout: []string{"some-branch", "other-branch"}},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName:       "/git/root",
						branchArg.Name():         "tree",
						currentBranchArg.ArgName: "some-branch",
						userArg.Name:             "person",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git checkout tree`,
						},
					},
					WantStderr: fmt.Sprintf("Unprocessed extra args: [limb]\n\n%s\n%s\n", "======= Command Usage =======", u),
					WantErr:    fmt.Errorf(`Unprocessed extra args: [limb]`),
				},
			},
			{
				name: "checkout fails if can't list branches",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ch", "tree"},
					WantRunContents: []*commandtest.RunContents{
						{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}},
						{Name: "git", Args: []string{"rev-parse", "--abbrev-ref", "HEAD"}},
						{Name: "git", Args: []string{"branch", "--list"}},
					},
					RunResponses: []*commandtest.FakeRun{
						{Stdout: []string{"/git/root"}},
						{Stdout: []string{"some-branch"}},
						{Stdout: []string{"some-branch", "heyo"}, Err: fmt.Errorf("whoops"), Stderr: []string{"argo"}},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName:       "/git/root",
						currentBranchArg.ArgName: "some-branch",
						userArg.Name:             "person",
					}},
					WantStderr: "Custom transformer failed: failed to get git branches: failed to execute shell command: whoops\n",
					WantErr:    fmt.Errorf(`Custom transformer failed: failed to get git branches: failed to execute shell command: whoops`),
				},
			},
			{
				name: "checks out a branch",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ch", "tree"},
					WantRunContents: []*commandtest.RunContents{
						{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}},
						{Name: "git", Args: []string{"rev-parse", "--abbrev-ref", "HEAD"}},
						{Name: "git", Args: []string{"branch", "--list"}},
					},
					RunResponses: []*commandtest.FakeRun{
						{Stdout: []string{"/git/root"}},
						{Stdout: []string{"some-branch"}},
						{Stdout: []string{"xyz"}},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName:       "/git/root",
						branchArg.Name():         "tree",
						currentBranchArg.ArgName: "some-branch",
						userArg.Name:             "person",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git checkout tree`,
						},
					},
				},
				want: &git{
					PreviousBranches: map[string]string{
						"/git/root": "some-branch",
					},
				},
			},
			{
				name: "checks out a new branch - creates map",
				g:    &git{},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ch", "tree", "-n"},
					WantRunContents: []*commandtest.RunContents{
						{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}},
						{Name: "git", Args: []string{"rev-parse", "--abbrev-ref", "HEAD"}},
						{Name: "git", Args: []string{"branch", "--list"}},
					},
					RunResponses: []*commandtest.FakeRun{
						{Stdout: []string{"/git/root"}},
						{Stdout: []string{"some-branch"}},
						{Stdout: []string{"xyz"}},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName:       "/git/root",
						branchArg.Name():         "tree",
						newBranchFlag.Name():     true,
						currentBranchArg.ArgName: "some-branch",
						userArg.Name:             "person",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git checkout -b tree`,
						},
					},
				},
				want: &git{
					PreviousBranches: map[string]string{
						"/git/root": "some-branch",
					},
					ParentBranches: map[string]string{
						"tree": "some-branch",
					},
				},
			},
			{
				name: "checks out a new branch - adds to map",
				g: &git{
					ParentBranches: map[string]string{},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ch", "tree", "-n"},
					WantRunContents: []*commandtest.RunContents{
						{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}},
						{Name: "git", Args: []string{"rev-parse", "--abbrev-ref", "HEAD"}},
						{Name: "git", Args: []string{"branch", "--list"}},
					},
					RunResponses: []*commandtest.FakeRun{
						{Stdout: []string{"/git/root"}},
						{Stdout: []string{"some-branch"}},
						{Stdout: []string{"xyz"}},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName:       "/git/root",
						branchArg.Name():         "tree",
						newBranchFlag.Name():     true,
						currentBranchArg.ArgName: "some-branch",
						userArg.Name:             "person",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git checkout -b tree`,
						},
					},
				},
				want: &git{
					PreviousBranches: map[string]string{
						"/git/root": "some-branch",
					},
					ParentBranches: map[string]string{
						"tree": "some-branch",
					},
				},
			},
			{
				name: "checks out a new branch - overrides value in map",
				g: &git{
					ParentBranches: map[string]string{
						"tree":  "old-branch",
						"other": "other-branch",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ch", "tree", "-n"},
					WantRunContents: []*commandtest.RunContents{
						{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}},
						{Name: "git", Args: []string{"rev-parse", "--abbrev-ref", "HEAD"}},
						{Name: "git", Args: []string{"branch", "--list"}},
					},
					RunResponses: []*commandtest.FakeRun{
						{Stdout: []string{"/git/root"}},
						{Stdout: []string{"some-branch"}},
						{Stdout: []string{"xyz"}},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName:       "/git/root",
						branchArg.Name():         "tree",
						newBranchFlag.Name():     true,
						currentBranchArg.ArgName: "some-branch",
						userArg.Name:             "person",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git checkout -b tree`,
						},
					},
				},
				want: &git{
					PreviousBranches: map[string]string{
						"/git/root": "some-branch",
					},
					ParentBranches: map[string]string{
						"tree":  "some-branch",
						"other": "other-branch",
					},
				},
			},
			{
				name: "adds user prefix if matches",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ch", "tree"},
					WantRunContents: []*commandtest.RunContents{
						{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}},
						{Name: "git", Args: []string{"rev-parse", "--abbrev-ref", "HEAD"}},
						{Name: "git", Args: []string{"branch", "--list"}},
					},
					RunResponses: []*commandtest.FakeRun{
						{Stdout: []string{"/git/root"}},
						{Stdout: []string{"some-branch"}},
						{Stdout: []string{
							"limb",
							"\tperson/tree  ",
							"person/root",
							"leaf",
						}},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName:       "/git/root",
						branchArg.Name():         "person/tree",
						currentBranchArg.ArgName: "some-branch",
						userArg.Name:             "person",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git checkout person/tree`,
						},
					},
				},
				want: &git{
					PreviousBranches: map[string]string{
						"/git/root": "some-branch",
					},
				},
			},
			{
				name: "uses branch with no prefix if both exist",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ch", "tree"},
					WantRunContents: []*commandtest.RunContents{
						{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}},
						{Name: "git", Args: []string{"rev-parse", "--abbrev-ref", "HEAD"}},
						{Name: "git", Args: []string{"branch", "--list"}},
					},
					RunResponses: []*commandtest.FakeRun{
						{Stdout: []string{"/git/root"}},
						{Stdout: []string{"some-branch"}},
						{Stdout: []string{
							"person/tree",
							" tree\t",
						}},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						gitRootDir.ArgName:       "/git/root",
						branchArg.Name():         "tree",
						currentBranchArg.ArgName: "some-branch",
						userArg.Name:             "person",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git checkout tree`,
						},
					},
				},
				want: &git{
					PreviousBranches: map[string]string{
						"/git/root": "some-branch",
					},
				},
			},
			// Delete branch
			{
				name: "delete branch requires arg",
				etc: &commandtest.ExecuteTestCase{
					Args:       []string{"bd"},
					WantStderr: "Argument \"BRANCH\" requires at least 1 argument, got 0\n",
					WantErr:    fmt.Errorf(`Argument "BRANCH" requires at least 1 argument, got 0`),
				},
			},
			{
				name: "deletes a branch",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"bd", "tree"},
					WantData: &command.Data{Values: map[string]interface{}{
						branchesArg.Name(): []string{"tree"},
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git branch -d "tree"`,
						},
					},
				},
			},
			{
				name: "deletes a branch in ParentBranches",
				g: &git{
					ParentBranches: map[string]string{
						"abc":  "def",
						"tree": "root",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"bd", "tree"},
					WantData: &command.Data{Values: map[string]interface{}{
						branchesArg.Name(): []string{"tree"},
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git branch -d "tree"`,
						},
					},
				},
				want: &git{
					ParentBranches: map[string]string{
						"abc": "def",
					},
				},
			},
			{
				name: "deletes multiple branches",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"bd", "tree", "limb"},
					WantData: &command.Data{Values: map[string]interface{}{
						branchesArg.Name(): []string{"tree", "limb"},
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git branch -d "tree" "limb"`,
						},
					},
				},
			},
			{
				name: "deletes multiple branches from ParentBranches",
				g: &git{
					ParentBranches: map[string]string{
						"abc":   "def",
						"tree":  "root",
						"other": "branch",
						"limb":  "leaf",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"bd", "tree", "limb"},
					WantData: &command.Data{Values: map[string]interface{}{
						branchesArg.Name(): []string{"tree", "limb"},
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git branch -d "tree" "limb"`,
						},
					},
				},
				want: &git{
					ParentBranches: map[string]string{
						"abc":   "def",
						"other": "branch",
					},
				},
			},
			{
				name: "force deletes a branch",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"bd", "-f", "tree"},
					WantData: &command.Data{Values: map[string]interface{}{
						branchArg.Name():   []string{"tree"},
						forceDelete.Name(): true,
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git branch -D "tree"`,
						},
					},
				},
			},
			// Undo add
			{
				name: "undo works with no args",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ua"},
					// WantData: &command.Data{Values: map[string]interface{}{}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git reset -- .`,
						},
					},
				},
			},
			{
				name: "undo resets files",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ua", "file.one", "some/where/file.2"},
					WantData: &command.Data{Values: map[string]interface{}{
						uaArgs.Name(): []string{
							"file.one",
							"some/where/file.2",
						},
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git reset -- file.one some/where/file.2`,
						},
					},
				},
			},
			// Undo change
			{
				name: "undo change requires args",
				etc: &commandtest.ExecuteTestCase{
					Args:       []string{"uc"},
					WantStderr: "Argument \"FILE\" requires at least 1 argument, got 0\n",
					WantErr:    fmt.Errorf(`Argument "FILE" requires at least 1 argument, got 0`),
				},
			},
			{
				name: "undo change undoes changed files",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"uc", "file.one", "some/where/file.2"},
					WantData: &command.Data{Values: map[string]interface{}{
						ucArgs.Name(): []string{
							"file.one",
							"some/where/file.2",
						},
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git checkout -- file.one some/where/file.2`,
						},
					},
				},
			},
			// Status
			{
				name: "status with no args",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"s"},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git status `,
						},
					},
				},
			},
			{
				name: "status with args args",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"s", "file.one", "some/where/file.2"},
					WantData: &command.Data{Values: map[string]interface{}{
						statusFilesArg.Name(): []string{
							"file.one",
							"some/where/file.2",
						},
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git status file.one some/where/file.2`,
						},
					},
				},
			},
			// Add
			{
				name: "add with no args",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"a"},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git add .`,
						},
					},
				},
			},
			{
				name: "add with args",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"a", "file.one", "some/where/file.2"},
					WantData: &command.Data{Values: map[string]interface{}{
						addFilesArg.Name(): []string{
							"file.one",
							"some/where/file.2",
						},
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git add file.one some/where/file.2`,
						},
					},
				},
			},
			{
				name: "add with no args and no-op flag",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"a", "-w"},
					WantData: &command.Data{Values: map[string]interface{}{
						noopWhitespaceFlag.Name(): true,
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git add .`,
						},
					},
				},
			},
			{
				name: "add with args and no-op flag",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"a", "file.one", "some/where/file.2", "--whitespace"},
					WantData: &command.Data{Values: map[string]interface{}{
						addFilesArg.Name(): []string{
							"file.one",
							"some/where/file.2",
						},
						noopWhitespaceFlag.Name(): true,
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git add file.one some/where/file.2`,
						},
					},
				},
			},
			// Remove
			{
				name: "rm with no args fails",
				etc: &commandtest.ExecuteTestCase{
					Args:       []string{"rm"},
					WantStderr: "Argument \"FILES\" requires at least 1 argument, got 0\n",
					WantErr:    fmt.Errorf("Argument \"FILES\" requires at least 1 argument, got 0"),
				},
			},
			{
				name: "rm with args",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"rm", "some-file.txt"},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"rm some-file.txt",
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						rmFilesArg.Name(): []string{
							"some-file.txt",
						},
					}},
				},
			},
			{
				name: "rm with -rf flag",
				etc: &commandtest.ExecuteTestCase{
					// Fine with not making specific flags for -rf because would be
					// different in windows so adding only adds work for us with little
					// (no?) actual benefit to usage
					Args: []string{"rm", "some-file.txt", "-rf", "other-file.go"},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"rm some-file.txt -rf other-file.go",
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						rmFilesArg.Name(): []string{
							"some-file.txt",
							"-rf",
							"other-file.go",
						},
					}},
				},
			},
			// Diff
			{
				name: "diff with no args",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"d"},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*commandtest.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						commander.Getwd.Name: filepath.Join("/", "fake", "root"),
						repoUrl.Name():       "test-repo",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git diff  -- `,
						},
					},
				},
			},
			{
				name: "diff with args",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"d", "this.file", "that/file/txt"},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*commandtest.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						commander.Getwd.Name: filepath.Join("/", "fake", "root"),
						repoUrl.Name():       "test-repo",
						diffArgs.Name(): []string{
							"this.file",
							"that/file/txt",
						},
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git diff  -- this.file that/file/txt`,
						},
					},
				},
			},
			{
				name: "diff against main branch",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"d", "-m"},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*commandtest.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						commander.Getwd.Name: filepath.Join("/", "fake", "root"),
						repoUrl.Name():       "test-repo",
						mainFlag.Name():      true,
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git diff  main `,
						},
					},
				},
			},
			{
				name: "diff against last commit",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"d", "-c"},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*commandtest.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						commander.Getwd.Name:  filepath.Join("/", "fake", "root"),
						repoUrl.Name():        "test-repo",
						prevCommitFlag.Name(): true,
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git diff  "$(git rev-parse @~1)" `,
						},
					},
				},
			},
			{
				name: "diff with whitespace flag",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"d", "-w"},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*commandtest.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						commander.Getwd.Name:  filepath.Join("/", "fake", "root"),
						repoUrl.Name():        "test-repo",
						whitespaceFlag.Name(): "-w",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git diff -w -- `,
						},
					},
				},
			},
			{
				name: "diff add with no args",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"d", "-a"},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*commandtest.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						commander.Getwd.Name: filepath.Join("/", "fake", "root"),
						repoUrl.Name():       "test-repo",
						addFlag.Name():       true,
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git add .`,
						},
					},
				},
			},
			{
				name: "diff add with args",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"d", "this.file", "that/file/txt", "-a"},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*commandtest.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						commander.Getwd.Name: filepath.Join("/", "fake", "root"),
						repoUrl.Name():       "test-repo",
						diffArgs.Name(): []string{
							"this.file",
							"that/file/txt",
						},
						addFlag.Name(): true,
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git add this.file that/file/txt`,
						},
					},
				},
			},
			{
				name: "diff add with other flags ignores other flags",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"d", "this.file", "that/file/txt", "-a", "--main", "-w"},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*commandtest.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						commander.Getwd.Name: filepath.Join("/", "fake", "root"),
						repoUrl.Name():       "test-repo",
						diffArgs.Name(): []string{
							"this.file",
							"that/file/txt",
						},
						addFlag.Name():        true,
						mainFlag.Name():       true,
						whitespaceFlag.Name(): "-w",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git add this.file that/file/txt`,
						},
					},
				},
			},
			// Rebase tests
			{
				name: "Rebase abort",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"rb", "a"},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git rebase --abort`,
						},
					},
					WantStdout: "git rebase --abort\n",
				},
			},
			{
				name: "Rebase abort",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"rb", "c"},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git rebase --continue`,
						},
					},
					WantStdout: "git rebase --continue\n",
				},
			},
			// Config tests
			{
				name: "Shows empty config",
				etc: &commandtest.ExecuteTestCase{
					Args:       []string{"cfg", "main", "show"},
					WantStdout: "No global default branch set; using main\n",
				},
			},
			{
				name: "Shows default branch config",
				g: &git{
					DefaultBranch: "other-main",
					MainBranches: map[string]string{
						"un":   "main-one",
						"deux": "main-two",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"cfg", "main", "show"},
					WantStdout: strings.Join([]string{
						"Global default branch: other-main",
						"deux: main-two",
						"un: main-one",
						"",
					}, "\n"),
				},
			},
			{
				name: "Sets default branch",
				g:    &git{},
				want: &git{
					MainBranches: map[string]string{
						"some-repo": "db",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"cfg", "main", "set", "db"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{
							"config",
							"--get",
							"remote.origin.url",
						},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						repoUrl.Name():    "some-repo",
						defRepoArg.Name(): "db",
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-repo"},
					}},
					WantStdout: "Setting default branch for some-repo to db\n",
				},
			},
			{
				name: "Sets global default branch",
				g:    &git{},
				want: &git{
					DefaultBranch: "Maine",
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"cfg", "main", "set", "Maine", "-g"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{
							"config",
							"--get",
							"remote.origin.url",
						},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						repoUrl.Name():      "some-repo",
						defRepoArg.Name():   "Maine",
						globalConfig.Name(): true,
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-repo"},
					}},
					WantStdout: "Setting global default branch to Maine\n",
				},
			},
			{
				name: "Unsets default branch",
				g: &git{
					MainBranches: map[string]string{
						"some-repo": "db",
						"other":     "heh",
					},
				},
				want: &git{
					MainBranches: map[string]string{
						"other": "heh",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"cfg", "main", "unset"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{
							"config",
							"--get",
							"remote.origin.url",
						},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						repoUrl.Name(): "some-repo",
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-repo"},
					}},
					WantStdout: "Deleting default branch for some-repo\n",
				},
			},
			{
				name: "Does nothing if no default branch map",
				g:    &git{},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"cfg", "main", "unset"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{
							"config",
							"--get",
							"remote.origin.url",
						},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						repoUrl.Name(): "some-repo",
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-repo"},
					}},
					WantStdout: "No default branch set for this repo\n",
				},
			},
			{
				name: "Does nothing if no default branch for repo",
				g: &git{
					MainBranches: map[string]string{
						"other": "dflt",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"cfg", "main", "unset"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{
							"config",
							"--get",
							"remote.origin.url",
						},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						repoUrl.Name(): "some-repo",
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-repo"},
					}},
					WantStdout: "No default branch set for this repo\n",
				},
			},
			{
				name: "Unsets global default branch",
				g: &git{
					DefaultBranch: "Maine",
				},
				want: &git{},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"cfg", "main", "unset", "-g"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{
							"config",
							"--get",
							"remote.origin.url",
						},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						repoUrl.Name():      "some-repo",
						globalConfig.Name(): true,
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-repo"},
					}},
					WantStdout: "Deleting global default branch\n",
				},
			},
			// pr-link tests
			{
				name: "pr-link requires current branch",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"pr-link"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{
							"rev-parse",
							"--abbrev-ref",
							"HEAD",
						},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Err: fmt.Errorf("oops"),
					}},
					WantStderr: "failed to execute shell command: oops\n",
					WantErr:    fmt.Errorf("failed to execute shell command: oops"),
				},
			},
			{
				name: "pr-link requires repo url",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"pr-link"},
					WantRunContents: []*commandtest.RunContents{
						{
							Name: "git",
							Args: []string{
								"rev-parse",
								"--abbrev-ref",
								"HEAD",
							},
						},
						{
							Name: "git",
							Args: []string{
								"config",
								"--get",
								"remote.origin.url",
							},
						},
					},
					RunResponses: []*commandtest.FakeRun{
						{
							Stdout: []string{"tree-branch"},
						},
						{
							Err: fmt.Errorf("oh no"),
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						currentBranchArg.ArgName: "tree-branch",
					}},
					WantStderr: "failed to execute shell command: oh no\n",
					WantErr:    fmt.Errorf("failed to execute shell command: oh no"),
				},
			},
			{
				name: "pr-link requires repo with valid format",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"pr-link"},
					WantRunContents: []*commandtest.RunContents{
						{
							Name: "git",
							Args: []string{
								"rev-parse",
								"--abbrev-ref",
								"HEAD",
							},
						},
						{
							Name: "git",
							Args: []string{
								"config",
								"--get",
								"remote.origin.url",
							},
						},
					},
					RunResponses: []*commandtest.FakeRun{
						{
							Stdout: []string{"tree-branch"},
						},
						{
							Stdout: []string{"some-new-format:org/repo.git"},
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						currentBranchArg.ArgName: "tree-branch",
						repoUrl.ArgName:          "some-new-format:org/repo.git",
					}},
					WantStderr: "Unknown git url format: some-new-format:org/repo.git\n",
					WantErr:    fmt.Errorf("Unknown git url format: some-new-format:org/repo.git"),
				},
			},
			{
				name: "pr-link fails if no parent branch or default main branch",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"pr-link"},
					WantRunContents: []*commandtest.RunContents{
						{
							Name: "git",
							Args: []string{
								"rev-parse",
								"--abbrev-ref",
								"HEAD",
							},
						},
						{
							Name: "git",
							Args: []string{
								"config",
								"--get",
								"remote.origin.url",
							},
						},
					},
					RunResponses: []*commandtest.FakeRun{
						{
							Stdout: []string{"tree-branch"},
						},
						{
							Stdout: []string{"git@github.com:user/repo.git"},
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						currentBranchArg.ArgName: "tree-branch",
						repoUrl.ArgName:          "git@github.com:user/repo.git",
					}},
					WantStderr: "Unknown parent branch for branch tree-branch; and no default main branch set\n",
					WantErr:    fmt.Errorf("Unknown parent branch for branch tree-branch; and no default main branch set"),
				},
			},
			{
				name: "pr-link works if parent branch set",
				g: &git{
					ParentBranches: map[string]string{
						"tree-branch": "trunk",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"pr-link"},
					WantRunContents: []*commandtest.RunContents{
						{
							Name: "git",
							Args: []string{
								"rev-parse",
								"--abbrev-ref",
								"HEAD",
							},
						},
						{
							Name: "git",
							Args: []string{
								"config",
								"--get",
								"remote.origin.url",
							},
						},
					},
					RunResponses: []*commandtest.FakeRun{
						{
							Stdout: []string{"tree-branch"},
						},
						{
							Stdout: []string{"git@github.com:user/repo.git"},
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						currentBranchArg.ArgName: "tree-branch",
						repoUrl.ArgName:          "git@github.com:user/repo.git",
					}},
					WantStdout: "https://github.com/user/repo/compare/trunk...tree-branch?expand=1\n",
				},
			},
			{
				name: "pr-link works if default main branch set",
				g: &git{
					MainBranches: map[string]string{
						"git@github.com:user/repo.git": "maine",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"pr-link"},
					WantRunContents: []*commandtest.RunContents{
						{
							Name: "git",
							Args: []string{
								"rev-parse",
								"--abbrev-ref",
								"HEAD",
							},
						},
						{
							Name: "git",
							Args: []string{
								"config",
								"--get",
								"remote.origin.url",
							},
						},
					},
					RunResponses: []*commandtest.FakeRun{
						{
							Stdout: []string{"tree-branch"},
						},
						{
							Stdout: []string{"git@github.com:user/repo.git"},
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						currentBranchArg.ArgName: "tree-branch",
						repoUrl.ArgName:          "git@github.com:user/repo.git",
					}},
					WantStdout: "https://github.com/user/repo/compare/maine...tree-branch?expand=1\n",
				},
			},
			{
				name: "pr-link uses parent branch over default main branch",
				g: &git{
					ParentBranches: map[string]string{
						"tree-branch": "trunk",
					},
					MainBranches: map[string]string{
						"git@github.com:user/repo.git": "maine",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"pr-link"},
					WantRunContents: []*commandtest.RunContents{
						{
							Name: "git",
							Args: []string{
								"rev-parse",
								"--abbrev-ref",
								"HEAD",
							},
						},
						{
							Name: "git",
							Args: []string{
								"config",
								"--get",
								"remote.origin.url",
							},
						},
					},
					RunResponses: []*commandtest.FakeRun{
						{
							Stdout: []string{"tree-branch"},
						},
						{
							Stdout: []string{"git@github.com:user/repo.git"},
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						currentBranchArg.ArgName: "tree-branch",
						repoUrl.ArgName:          "git@github.com:user/repo.git",
					}},
					WantStdout: "https://github.com/user/repo/compare/trunk...tree-branch?expand=1\n",
				},
			},
			{
				name: "pr-link works for https remote origin",
				g: &git{
					ParentBranches: map[string]string{
						"tree-branch": "trunk",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"pr-link"},
					WantRunContents: []*commandtest.RunContents{
						{
							Name: "git",
							Args: []string{
								"rev-parse",
								"--abbrev-ref",
								"HEAD",
							},
						},
						{
							Name: "git",
							Args: []string{
								"config",
								"--get",
								"remote.origin.url",
							},
						},
					},
					RunResponses: []*commandtest.FakeRun{
						{
							Stdout: []string{"tree-branch"},
						},
						{
							Stdout: []string{"https://github.com/user/repo.git"},
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						currentBranchArg.ArgName: "tree-branch",
						repoUrl.ArgName:          "https://github.com/user/repo.git",
					}},
					WantStdout: "https://github.com/user/repo/compare/trunk...tree-branch?expand=1\n",
				},
			},
			// Current branch test
			{
				name: "fails if current branch has error",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"current"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
						Err:    fmt.Errorf("whoops"),
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						formatFlag.Name(): "%s\n",
					}},
					WantStderr: "failed to execute shell command: whoops\n",
					WantErr:    fmt.Errorf("failed to execute shell command: whoops"),
				},
			},
			{
				name: "ignores failure if ignore flag provided",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"current", "-i"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
						Stderr: []string{"oopsie"},
						Err:    fmt.Errorf("whoops"),
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						formatFlag.Name():     "%s\n",
						ignoreNoBranch.Name(): true,
					}},
				},
			},
			{
				name: "prints current branch",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"current"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						formatFlag.Name(): "%s\n",
					}},
					WantStdout: "some-branch\n",
				},
			},
			{
				name: "prints current branch with custom format",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"current", "-f", "hello, %s; goodbye"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						formatFlag.Name(): "hello, %s; goodbye",
					}},
					WantStdout: "hello, some-branch; goodbye",
				},
			},
			{
				name: "current branch with parent format but no parent",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"current", "-F", "%s --> "},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						formatFlag.Name():       "%s\n",
						parentFormatFlag.Name(): "%s --> ",
					}},
					WantStdout: "some-branch\n",
				},
			},
			{
				name: "current branch with parent format works",
				g: &git{
					ParentBranches: map[string]string{
						"some-branch": "dad",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"current", "-F", "%s --> "},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						formatFlag.Name():       "%s\n",
						parentFormatFlag.Name(): "%s --> ",
					}},
					WantStdout: "dad --> some-branch\n",
				},
			},
			{
				name: "current branch with parent format works with multiple parents",
				g: &git{
					ParentBranches: map[string]string{
						"some-branch": "dad",
						"dad":         "granddad",
						"granddad":    "great granddad",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"current", "-F", "%s --> "},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						formatFlag.Name():       "%s\n",
						parentFormatFlag.Name(): "%s --> ",
					}},
					WantStdout: "great granddad --> granddad --> dad --> some-branch\n",
				},
			},
			{
				name: "current branch works with prefix and suffix",
				g: &git{
					ParentBranches: map[string]string{
						"some-branch": "dad",
						"dad":         "granddad",
						"granddad":    "great granddad",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"current", "-F", "%s --> ", "-p", "((", "-s", "]]"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						formatFlag.Name():       "%s\n",
						parentFormatFlag.Name(): "%s --> ",
						prefixFlag.Name():       "((",
						suffixFlag.Name():       "]]",
					}},
					WantStdout: "((great granddad --> granddad --> dad --> some-branch\n]]",
				},
			},
			{
				name: "current branch fails if cycle with base branch",
				g: &git{
					ParentBranches: map[string]string{
						"some-branch":    "other-branch",
						"other-branch":   "another-branch",
						"another-branch": "some-branch",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"current", "-F", "%s --> "},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						formatFlag.Name():       "%s\n",
						parentFormatFlag.Name(): "%s --> ",
					}},
					WantStderr: "cycle detected in parent branches\n",
					WantErr:    fmt.Errorf("cycle detected in parent branches"),
				},
			},
			{
				name: "current branch fails if cycle with parent branches",
				g: &git{
					ParentBranches: map[string]string{
						"some-branch":    "other-branch",
						"other-branch":   "another-branch",
						"another-branch": "other-branch",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"current", "-F", "%s --> "},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						formatFlag.Name():       "%s\n",
						parentFormatFlag.Name(): "%s --> ",
					}},
					WantStderr: "cycle detected in parent branches\n",
					WantErr:    fmt.Errorf("cycle detected in parent branches"),
				},
			},
			// upstream + pr-link tests
			{
				name: "upstream + pr-link fails if current branch error",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"up"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stderr: []string{"argh"},
						Err:    fmt.Errorf("oops"),
						Stdout: []string{"some-branch"},
					}},
					WantStderr: "argh\nfailed to execute shell command: oops\n",
					WantErr:    fmt.Errorf("failed to execute shell command: oops"),
				},
			},
			{
				name: "upstream + pr-link fails if repoUrl error",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"up"},
					WantRunContents: []*commandtest.RunContents{
						{
							Name: "git",
							Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
						},
						{
							Name: "git",
							Args: []string{"config", "--get", "remote.origin.url"},
						},
					},
					RunResponses: []*commandtest.FakeRun{
						{
							Stdout: []string{"some-branch"},
						},
						{
							Stdout: []string{"git@github.com:user/some-repo.git"},
							Stderr: []string{"oh no"},
							Err:    fmt.Errorf("fudge"),
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						currentBranchArg.ArgName: "some-branch",
					}},
					WantStderr: "oh no\nfailed to execute shell command: fudge\n",
					WantErr:    fmt.Errorf("failed to execute shell command: fudge"),
				},
			},
			{
				name: "upstream + pr-link fails if push error",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"up"},
					WantRunContents: []*commandtest.RunContents{
						{
							Name: "git",
							Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
						},
						{
							Name: "git",
							Args: []string{"config", "--get", "remote.origin.url"},
						},
						{
							Name: "git",
							Args: []string{"push", "--set-upstream", "origin", "some-branch"},
						},
					},
					RunResponses: []*commandtest.FakeRun{
						{
							Stdout: []string{"some-branch"},
						},
						{
							Stdout: []string{"git@github.com:user/some-repo.git"},
						},
						{
							Stdout: []string{"push output"},
							Stderr: []string{"ugh"},
							Err:    fmt.Errorf("whoops"),
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						currentBranchArg.ArgName: "some-branch",
						repoUrl.ArgName:          "git@github.com:user/some-repo.git",
					}},
					WantStderr: "failed to run git push: failed to execute shell command: whoops\n",
					WantErr:    fmt.Errorf("failed to run git push: failed to execute shell command: whoops"),
				},
			},
			{
				name: "upstream + pr-link fails if printPRLink fails",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"up"},
					WantRunContents: []*commandtest.RunContents{
						{
							Name: "git",
							Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
						},
						{
							Name: "git",
							Args: []string{"config", "--get", "remote.origin.url"},
						},
						{
							Name: "git",
							Args: []string{"push", "--set-upstream", "origin", "some-branch"},
						},
					},
					RunResponses: []*commandtest.FakeRun{
						{
							Stdout: []string{"some-branch"},
						},
						{
							Stdout: []string{"git@github.com:user/some-repo.git"},
						},
						{
							Stdout: []string{"push output"},
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						currentBranchArg.ArgName: "some-branch",
						repoUrl.ArgName:          "git@github.com:user/some-repo.git",
					}},
					WantStderr: "Unknown parent branch for branch some-branch; and no default main branch set\n",
					WantErr:    fmt.Errorf("Unknown parent branch for branch some-branch; and no default main branch set"),
				},
			},
			{
				name: "upstream + pr-link works",
				g: &git{
					ParentBranches: map[string]string{
						"some-branch": "parent-branch",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"up"},
					WantRunContents: []*commandtest.RunContents{
						{
							Name: "git",
							Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
						},
						{
							Name: "git",
							Args: []string{"config", "--get", "remote.origin.url"},
						},
						{
							Name: "git",
							Args: []string{"push", "--set-upstream", "origin", "some-branch"},
						},
					},
					RunResponses: []*commandtest.FakeRun{
						{
							Stdout: []string{"some-branch"},
						},
						{
							Stdout: []string{"git@github.com:user/some-repo.git"},
						},
						{
							Stdout: []string{"push output"},
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						currentBranchArg.ArgName: "some-branch",
						repoUrl.ArgName:          "git@github.com:user/some-repo.git",
					}},
					WantStdout: "https://github.com/user/some-repo/compare/parent-branch...some-branch?expand=1\n",
				},
			},
			// DryRun tests
			{
				name: "dry run - git log",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"lg", "-y"},
					WantData: &command.Data{Values: map[string]interface{}{
						dryRunFlag.Name(): true,
						gitLogArg.Name():  1,
					}},
					WantStdout: strings.Join([]string{
						"# Dry Run Summary",
						"# Number of executor functions: 0",
						"# Shell executables:",
						"git log -n 1",
						"",
					}, "\n"),
				},
			},
			{
				name: "dry run - git commit",
				osChecks: map[string]*osCheck{
					"windows": {
						wantStdout: []string{
							"# Dry Run Summary",
							"# Number of executor functions: 0",
							"# Shell executables:",
							"",
							`git commit -m "hello there"`,
							`if (!$?) { throw "Command failed: git commit -m 'hello there'" }`,
							`git push`,
							`if (!$?) { throw "Command failed: git push" }`,
							`echo Success!`,
							`if (!$?) { throw "Command failed: echo Success!" }`,
							"",
						},
					},
					"linux": {
						wantStdout: []string{
							"# Dry Run Summary",
							"# Number of executor functions: 0",
							"# Shell executables:",
							"",
							`git commit -m "hello there" && git push && echo Success!`,
							"",
						},
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"-y", "c", "hello", "there", "-p"},
					WantData: &command.Data{Values: map[string]interface{}{
						dryRunFlag.Name(): true,
						messageArg.Name(): []string{"hello", "there"},
						pushFlag.Name():   true,
					}},
					WantExecuteData: &command.ExecuteData{FunctionWrap: true},
				},
			},
			// End branch tests
			{
				name: "end branch requires current branch",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"end"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{
							"rev-parse",
							"--abbrev-ref",
							"HEAD",
						},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Err: fmt.Errorf("oops"),
					}},
					WantStderr: "failed to execute shell command: oops\n",
					WantErr:    fmt.Errorf("failed to execute shell command: oops"),
				},
			},
			{
				name: "end branch requires parent branch",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"end"},
					WantRunContents: []*commandtest.RunContents{
						{
							Name: "git",
							Args: []string{
								"rev-parse",
								"--abbrev-ref",
								"HEAD",
							},
						},
					},
					RunResponses: []*commandtest.FakeRun{
						{
							Stdout: []string{"tree-branch"},
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						currentBranchArg.ArgName: "tree-branch",
					}},
					WantStderr: "branch tree-branch does not have a known parent branch\n",
					WantErr:    fmt.Errorf("branch tree-branch does not have a known parent branch"),
				},
			},
			{
				name: "end branch succeeds",
				g: &git{
					ParentBranches: map[string]string{
						"tree-branch": "trunk",
					},
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"end"},
					WantRunContents: []*commandtest.RunContents{
						{
							Name: "git",
							Args: []string{
								"rev-parse",
								"--abbrev-ref",
								"HEAD",
							},
						},
					},
					RunResponses: []*commandtest.FakeRun{
						{
							Stdout: []string{"tree-branch"},
						},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						currentBranchArg.ArgName: "tree-branch",
					}},
				},
				osChecks: map[string]*osCheck{
					"windows": {
						wantExecutable: []string{
							wCmd("git checkout trunk"),
							wCmd("git pull"),
							wCmd("gbd tree-branch"),
						},
						wantStdout: []string{
							wCmd("git checkout trunk"),
							wCmd("git pull"),
							wCmd("gbd tree-branch"),
							"",
						},
					},
					"linux": {
						wantExecutable: []string{
							"git checkout trunk && git pull && gbd tree-branch",
						},
						wantStdout: []string{
							"git checkout trunk && git pull && gbd tree-branch",
							"",
						},
					},
				},
			},
			/* Useful for commenting out tests. */
		} {
			t.Run(fmt.Sprintf("[%s] %s", curOS.Name(), test.name), func(t *testing.T) {
				test.etc.Env = map[string]string{
					"USER": "person",
				}
				commandtest.StubGetwd(t, filepath.Join("/", "fake", "root"), nil)
				commandtest.StubValue(t, &sourcerer.CurrentOS, curOS)
				if oschk, ok := test.osChecks[curOS.Name()]; ok {
					if test.etc.WantExecuteData == nil {
						test.etc.WantExecuteData = &command.ExecuteData{}
					}
					test.etc.WantExecuteData.Executable = oschk.wantExecutable
					if test.etc.WantStdout == "" {
						test.etc.WantStdout = strings.Join(oschk.wantStdout, "\n")
					}
				}

				if test.g == nil {
					test.g = CLI()
				}
				test.etc.Node = test.g.Node()
				commandertest.ExecuteTest(t, test.etc)
				commandertest.ChangeTest(t, test.want, test.g, cmpopts.IgnoreUnexported(git{}), cmpopts.EquateEmpty())
			})
		}
	}
}

type gitStatusFile struct {
	name               string
	porcelain          []string
	diffNameOnly       bool
	diffNameOnlyCached bool
}

// Names of files below are the order of actions that happened to the file.
var (
	modifiedFile = &gitStatusFile{
		"modified.go",
		[]string{"1 .M N... 100644 100644 100644 7efc2d1ea4fa9c61329411bae30090ff3d0cf2be 7efc2d1ea4fa9c61329411bae30090ff3d0cf2be modified.go"},
		true,
		false,
	}
	modifiedCachedFile = &gitStatusFile{
		"modified-cached.go",
		[]string{"1 M. N... 100644 100644 100644 7efc2d1ea4fa9c61329411bae30090ff3d0cf2be e4680edc5a0a0f60ae4e01414f711e6a55a8d8d9 modified-cached.go"},
		false,
		true,
	}
	modifiedCachedModifiedFile = &gitStatusFile{
		"modified-cached-modified.go",
		[]string{"1 MM N... 100644 100644 100644 7efc2d1ea4fa9c61329411bae30090ff3d0cf2be e4680edc5a0a0f60ae4e01414f711e6a55a8d8d9 modified-cached-modified.go"},
		true,
		true,
	}
	// TODO:modifiedCachedDeletedFile
	deletedFile = &gitStatusFile{
		"deleted.go",
		[]string{"1 .D N... 100644 100644 000000 37abf5327a1c2b98ea66b8be27243b7690350236 37abf5327a1c2b98ea66b8be27243b7690350236 deleted.go"},
		true,
		false,
	}
	deletedCachedFile = &gitStatusFile{
		"deleted-cached.go",
		[]string{"1 D. N... 100644 000000 000000 37abf5327a1c2b98ea66b8be27243b7690350236 0000000000000000000000000000000000000000 deleted-cached.go"},
		false,
		true,
	}
	deletedCachedCreatedFile = &gitStatusFile{
		"deleted-cached-created.go",
		[]string{
			// Shows as both deleted file and new file
			"1 D. N... 100644 000000 000000 37abf5327a1c2b98ea66b8be27243b7690350236 0000000000000000000000000000000000000000 deleted-cached-created.go",
			"? deleted-cached-created.go",
		},
		false,
		true,
	}
	createdFile = &gitStatusFile{
		"created.go",
		[]string{"? created.go"},
		false,
		false,
	}
	createdCachedFile = &gitStatusFile{
		"created-cached.go",
		[]string{"1 A. N... 000000 100644 100644 0000000000000000000000000000000000000000 49cc8ef0e116cef009fe0bd72473a964bbd07f9b created-cached.go"},
		false,
		true,
	}
	createdCachedModifiedFile = &gitStatusFile{
		"created-cached-modified.go",
		[]string{"1 AM N... 000000 100644 100644 0000000000000000000000000000000000000000 49cc8ef0e116cef009fe0bd72473a964bbd07f9b created-cached-modified.go"},
		true,
		true,
	}
	createdCachedDeletedFile = &gitStatusFile{
		"created-cached-deleted.go",
		[]string{"1 AD N... 000000 100644 000000 0000000000000000000000000000000000000000 49cc8ef0e116cef009fe0bd72473a964bbd07f9b created-cached-deleted.go"},
		true,
		true,
	}

	allFiles = []*gitStatusFile{
		modifiedFile,
		modifiedCachedFile,
		modifiedCachedModifiedFile,
		deletedFile,
		deletedCachedFile,
		deletedCachedCreatedFile,
		createdFile,
		createdCachedFile,
		createdCachedModifiedFile,
		createdCachedDeletedFile,
	}

	diffNameFiles       = functional.Filter(allFiles, func(f *gitStatusFile) bool { return f.diffNameOnly })
	diffNameCachedFiles = functional.Filter(allFiles, func(f *gitStatusFile) bool { return f.diffNameOnlyCached })
)

func TestAutocompletePorcelain(t *testing.T) {
	for _, test := range []struct {
		name      string
		wantFiles []*gitStatusFile
		ctc       *commandtest.CompleteTestCase
	}{
		{
			name: "Completions for add",
			wantFiles: []*gitStatusFile{
				modifiedFile,
				modifiedCachedModifiedFile,
				deletedFile,
				deletedCachedCreatedFile,
				createdFile,
				createdCachedModifiedFile,
				createdCachedDeletedFile,
			},
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd a ",
				SkipDataCheck: true,
			},
		},
		{
			name: "Completions for rm",
			wantFiles: []*gitStatusFile{
				modifiedFile,
				modifiedCachedModifiedFile,
				deletedFile,
				deletedCachedCreatedFile,
				createdFile,
				createdCachedModifiedFile,
				createdCachedDeletedFile,
			},
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd rm ",
				SkipDataCheck: true,
			},
		},
		{
			name: "Completions for undo change",
			wantFiles: []*gitStatusFile{
				modifiedFile,
				modifiedCachedModifiedFile,
				deletedFile,
				deletedCachedCreatedFile,
				createdFile,
				createdCachedModifiedFile,
				createdCachedDeletedFile,
			},
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd uc ",
				SkipDataCheck: true,
			},
		},
		{
			name: "Completions for undo add",
			wantFiles: []*gitStatusFile{
				modifiedCachedFile,
				modifiedCachedModifiedFile,
				deletedCachedFile,
				deletedCachedCreatedFile,
				createdCachedFile,
				createdCachedModifiedFile,
				createdCachedDeletedFile,
			},
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd ua ",
				SkipDataCheck: true,
			},
		},
		{
			name: "Completions for status",
			wantFiles: []*gitStatusFile{
				modifiedFile,
				modifiedCachedFile,
				modifiedCachedModifiedFile,
				deletedFile,
				deletedCachedFile,
				deletedCachedCreatedFile,
				createdFile,
				createdCachedFile,
				createdCachedModifiedFile,
				createdCachedDeletedFile,
			},
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd s ",
				SkipDataCheck: true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := &git{}
			test.ctc.Node = g.Node()
			var statuses []string
			for _, f := range allFiles {
				statuses = append(statuses, f.porcelain...)
			}
			test.ctc.RunResponses = []*commandtest.FakeRun{{
				Stdout: statuses,
			}}
			test.ctc.WantRunContents = []*commandtest.RunContents{{
				Name: "git",
				Args: []string{"status", "--porcelain=v2"},
			}}

			test.ctc.Want = &command.Autocompletion{
				Suggestions: functional.Map[*gitStatusFile, string](test.wantFiles, func(f *gitStatusFile) string { return f.name }),
			}
			slices.Sort(test.ctc.Want.Suggestions)

			commandertest.AutocompleteTest(t, test.ctc)
		})
	}
}

func TestAutocomplete(t *testing.T) {
	for _, test := range []struct {
		name     string
		ctc      *commandtest.CompleteTestCase
		getwd    string
		getwdErr error
	}{
		{
			name:     "Completions for diff fails if getwd error",
			getwd:    filepath.Join("/", "fake", "root"),
			getwdErr: fmt.Errorf("wd oops"),
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd d ",
				SkipDataCheck: true,
				WantErr:       fmt.Errorf("failed to get current directory: wd oops"),
			},
		},
		{
			name:  "Completions for diff fails if git root fails",
			getwd: filepath.Join("/", "fake", "root"),
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd d ",
				SkipDataCheck: true,
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "--show-toplevel"},
					},
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Err:    fmt.Errorf("oh no"),
						Stdout: []string{"bloop"},
						Stderr: []string{"blop"},
					},
				},
				WantErr: fmt.Errorf("failed to get git root: failed to execute shell command: oh no"),
			},
		},
		{
			name:  "Completions for diff fails if git diff command fails",
			getwd: filepath.Join("/", "fake", "root"),
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd d ",
				SkipDataCheck: true,
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "--show-toplevel"},
					},
					{
						Name: "git",
						Args: []string{"diff", "--name-only"},
					},
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{filepath.Join("/", "fake", "root")},
					},
					{
						Err:    fmt.Errorf("whoops"),
						Stdout: []string{"abc", "def"},
						Stderr: []string{"ghi", "jkl", "mno"},
					},
				},
				WantErr: fmt.Errorf("failed to get diffable files: failed to execute shell command: whoops"),
			},
		},
		{
			name:  "Completions for diff fails if relative filepath fails",
			getwd: filepath.Join("/", "fake", "root"),
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd d ",
				SkipDataCheck: true,
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "--show-toplevel"},
					},
					{
						Name: "git",
						Args: []string{"diff", "--name-only"},
					},
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"not-absolute-path"},
					},
					{
						Stdout: []string{"abc", "def"},
					},
				},
				WantErr: fmt.Errorf("failed to get relative path: Rel: can't make %s relative to %s", filepath.Join("not-absolute-path", "abc"), filepath.Join("/", "fake", "root")),
			},
		},
		{
			name:  "Completions for diff works when in the same root",
			getwd: filepath.Join("/", "fake", "root"),
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd d ",
				SkipDataCheck: true,
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "--show-toplevel"},
					},
					{
						Name: "git",
						Args: []string{"diff", "--name-only"},
					},
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{filepath.Join("/", "fake", "root")},
					},
					{
						Stdout: []string{"abc", filepath.Join("def", "ghi")},
					},
				},
				Want: &command.Autocompletion{
					Suggestions: []string{"abc", filepath.Join("def", "ghi")},
				},
			},
		},
		{
			name:  "Completions for diff works when diffs are in the parent directory",
			getwd: filepath.Join("/", "fake", "root", "some-folder"),
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd d ",
				SkipDataCheck: true,
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "--show-toplevel"},
					},
					{
						Name: "git",
						Args: []string{"diff", "--name-only"},
					},
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{filepath.Join("/", "fake", "root")},
					},
					{
						Stdout: []string{
							"abc",
							filepath.Join("def", "ghi"),
							filepath.Join("some-folder", "123"),
							filepath.Join("some-folder", "sub-folder", "456"),
						},
					},
				},
				Want: &command.Autocompletion{
					Suggestions: []string{
						filepath.Join("..", "abc"),
						filepath.Join("..", "def", "ghi"),
						filepath.Join("123"),
						filepath.Join("sub-folder", "456"),
					},
				},
			},
		},
		{
			// This wouldn't ever really happen (as we wouldn't be in a git directory
			// if the git folder is the sub-folder), but figured we can add a test
			// just to codify what it would technically do in this situation.
			name: "Completions for diff works when diffs are in sub-directory",

			getwd: filepath.Join("/", "fake"),
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd d ",
				SkipDataCheck: true,
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "--show-toplevel"},
					},
					{
						Name: "git",
						Args: []string{"diff", "--name-only"},
					},
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{filepath.Join("/", "fake", "root")},
					},
					{
						Stdout: []string{
							"abc",
							filepath.Join("def", "ghi"),
							filepath.Join("some-folder", "123"),
							filepath.Join("some-folder", "sub-folder", "456"),
						},
					},
				},
				Want: &command.Autocompletion{
					Suggestions: []string{
						filepath.Join("root", "abc"),
						filepath.Join("root", "def", "ghi"),
						filepath.Join("root", "some-folder", "123"),
						filepath.Join("root", "some-folder", "sub-folder", "456"),
					},
				},
			},
		},
		{
			name:  "Completions for diff (case insensitive)",
			getwd: filepath.Join("/", "fake", "root"),
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd d A",
				SkipDataCheck: true,
				Want: &command.Autocompletion{
					Suggestions: []string{"abc"},
				},
				WantRunContents: []*commandtest.RunContents{
					{
						Name: "git",
						Args: []string{"rev-parse", "--show-toplevel"},
					},
					{
						Name: "git",
						Args: []string{"diff", "--name-only"},
					},
				},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{filepath.Join("/", "fake", "root")},
					},
					{
						Stdout: []string{"abc", "def"},
					},
				},
			},
		},
		// Branch completion tests
		{
			name: "Branch completions",
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd ch ",
				SkipDataCheck: true,
				Want: &command.Autocompletion{
					Suggestions: []string{"b-1", "b-3"},
				},
				WantRunContents: []*commandtest.RunContents{{
					Name: "git",
					Args: []string{"branch", "--list"},
				}},
				RunResponses: []*commandtest.FakeRun{{
					Stdout: []string{"  b-1 ", "* 	b-2", "		b-3		"},
				}},
			},
		},
		{
			name: "Handles no	branch completions",
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd ch ",
				SkipDataCheck: true,
				WantRunContents: []*commandtest.RunContents{{
					Name: "git",
					Args: []string{"branch", "--list"},
				}},
				RunResponses: []*commandtest.FakeRun{{}},
			},
		},
		{
			name: "Handles branch completion error",
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd ch ",
				SkipDataCheck: true,
				WantRunContents: []*commandtest.RunContents{{
					Name: "git",
					Args: []string{"branch", "--list"},
				}},
				RunResponses: []*commandtest.FakeRun{{
					Err: fmt.Errorf("oops"),
				}},
				WantErr: fmt.Errorf("failed to fetch autocomplete suggestions with shell command: failed to execute shell command: oops"),
			},
		},
		{
			name: "Branch completion strips user prefix",
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd ch ",
				SkipDataCheck: true,
				Want: &command.Autocompletion{
					Suggestions: []string{
						"b-1",
						"b-2",
						"other/b-1",
						"person/b-1",
						"person/b-2",
					},
				},
				WantRunContents: []*commandtest.RunContents{{
					Name: "git",
					Args: []string{"branch", "--list"},
				}},
				RunResponses: []*commandtest.FakeRun{{
					Stdout: []string{"  b-1 ", "* 	b-2", "		other/b-1		", "person/b-1", " \tperson/b-2 "},
				}},
			},
		},
		// Git add completions
		{
			name: "PrefixCompleter handles error",
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd a ",
				SkipDataCheck: true,
				WantRunContents: []*commandtest.RunContents{{
					Name: "git",
					Args: []string{"status", "--porcelain=v2"},
				}},
				RunResponses: []*commandtest.FakeRun{{
					Err: fmt.Errorf("whoops"),
				}},
				WantErr: fmt.Errorf("failed to get git status: failed to execute shell command: whoops"),
			},
		},
		// Branches completion tests
		{
			name: "Branches completions",
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd bd ",
				SkipDataCheck: true,
				Want: &command.Autocompletion{
					Suggestions: []string{"b-1", "b-3"},
				},
				WantRunContents: []*commandtest.RunContents{{
					Name: "git",
					Args: []string{"branch", "--list"},
				}},
				RunResponses: []*commandtest.FakeRun{{
					Stdout: []string{"  b-1 ", "* 	b-2", "		b-3		"},
				}},
			},
		},
		{
			name: "Branches completions is distinct",
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd bd b-1 ",
				SkipDataCheck: true,
				Want: &command.Autocompletion{
					Suggestions: []string{"b-3"},
				},
				WantRunContents: []*commandtest.RunContents{{
					Name: "git",
					Args: []string{"branch", "--list"},
				}},
				RunResponses: []*commandtest.FakeRun{{
					Stdout: []string{"  b-1 ", "* 	b-2", "		b-3		"},
				}},
			},
		},
		{
			name: "Branches completions handles error",
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd bd ",
				SkipDataCheck: true,
				// Want: &command.Autocompletion{
				// 	Suggestions: []string{"b-3"},
				// },
				WantRunContents: []*commandtest.RunContents{{
					Name: "git",
					Args: []string{"branch", "--list"},
				}},
				RunResponses: []*commandtest.FakeRun{{
					Err: fmt.Errorf("oh no"),
				}},
				WantErr: fmt.Errorf("failed to fetch autocomplete suggestions with shell command: failed to execute shell command: oh no"),
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.getwd != "" {
				commandtest.StubGetwd(t, test.getwd, test.getwdErr)
			}
			test.ctc.Env = map[string]string{
				"USER": "person",
			}
			g := &git{}
			test.ctc.Node = g.Node()
			commandertest.AutocompleteTest(t, test.ctc)
		})
	}
}

func TestMetadata(t *testing.T) {
	t.Run("GitAliasers() returns without issues", func(t *testing.T) {
		GitAliasers()
	})

	g := &git{}

	t.Run("Metadata functions returns without issues", func(t *testing.T) {
		g.Name()
		g.Setup()
	})

	t.Run("Fails if unknown OS", func(t *testing.T) {
		etc := &commandtest.ExecuteTestCase{
			Node:            g.Node(),
			Args:            []string{"pp"},
			WantErr:         fmt.Errorf(`Unknown OS ("other")`),
			WantStderr:      "Unknown OS (\"other\")\n",
			WantExecuteData: &command.ExecuteData{Executable: []string{""}, FunctionWrap: true},
		}
		fos := &fakeOS{sourcerer.Linux(), "other"}
		commandtest.StubValue(t, &sourcerer.CurrentOS, fos.os())
		commandertest.ExecuteTest(t, etc)
	})
}

type fakeOS struct {
	sourcerer.OS

	name string
}

func (f *fakeOS) Name() string     { return f.name }
func (f *fakeOS) os() sourcerer.OS { return f }
