package sourcecontrol

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command/command"
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
	}

	u := strings.Join([]string{
		`┓`,
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
		`┃   Diff`,
		`┣━━ d [ FILE ... ] --main|-m --commit|-c --whitespace|-w`,
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
		`┃   Status`,
		`┣━━ s [ FILES ... ]`,
		`┃`,
		`┃   Create ssh-agent`,
		`┣━━ sh`,
		`┃`,
		`┃   Undo add`,
		`┣━━ ua FILE [ FILE ... ]`,
		`┃`,
		`┃   Undo change`,
		`┣━━ uc FILE [ FILE ... ]`,
		`┃`,
		`┃   Undo commit`,
		`┣━━ uco`,
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
		`  [c] commit: Whether to diff against the previous commit`,
		`  [d] diff: Whether or not to diff the current changes against N commits prior`,
		`  [f] force-delete: force delete the branch`,
		`  [g] global: Whether or not to change the global setting`,
		`  [m] main: Whether to diff against main branch or just local diffs`,
		`  [n] new-branch: Whether or not to checkout a new branch`,
		`  [n] no-verify: Whether or not to run pre-commit checks`,
		`  [p] push: Whether or not to push afterwards`,
		`  [u] upstream: If set, push branch to upstream`,
		`  [w] whitespace: Whether or not to show whitespace in diffs`,
	}, "\n")

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
			// Checkout main
			{
				name: "checkout main",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"m"},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*commandtest.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoUrl.Name(): "test-repo",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git checkout main",
						},
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
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*commandtest.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoUrl.Name(): "test-repo",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git checkout main",
						},
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
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*commandtest.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoUrl.Name(): "test-repo",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git checkout mainer",
						},
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
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*commandtest.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoUrl.Name(): "test-repo",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git checkout mainer",
						},
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
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*commandtest.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoUrl.Name(): "test-repo",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git checkout mainest",
						},
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
				name: "checkout branch requires current branch",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ch"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Err: fmt.Errorf("whoops"),
					}},
					WantStderr: "failed to execute shell command: whoops\n",
					WantErr:    fmt.Errorf(`failed to execute shell command: whoops`),
				},
			},
			{
				name: "checkout branch requires arg",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ch"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						currentBranchArg.ArgName: "some-branch",
					}},
					WantStderr: "Argument \"BRANCH\" requires at least 1 argument, got 0\n",
					WantErr:    fmt.Errorf(`Argument "BRANCH" requires at least 1 argument, got 0`),
				},
			},
			{
				name: "checkout branch requires one arg",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ch", "tree", "limb"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						branchArg.Name():         "tree",
						currentBranchArg.ArgName: "some-branch",
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
				name: "checks out a branch",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ch", "tree"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						branchArg.Name():         "tree",
						currentBranchArg.ArgName: "some-branch",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git checkout tree`,
						},
					},
				},
			},
			{
				name: "checks out a new branch - creates map",
				g:    &git{},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"ch", "tree", "-n"},
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						branchArg.Name():         "tree",
						newBranchFlag.Name():     true,
						currentBranchArg.ArgName: "some-branch",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git checkout -b tree`,
						},
					},
				},
				want: &git{
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
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						branchArg.Name():         "tree",
						newBranchFlag.Name():     true,
						currentBranchArg.ArgName: "some-branch",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git checkout -b tree`,
						},
					},
				},
				want: &git{
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
					WantRunContents: []*commandtest.RunContents{{
						Name: "git",
						Args: []string{"rev-parse", "--abbrev-ref", "HEAD"},
					}},
					RunResponses: []*commandtest.FakeRun{{
						Stdout: []string{"some-branch"},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						branchArg.Name():         "tree",
						newBranchFlag.Name():     true,
						currentBranchArg.ArgName: "some-branch",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git checkout -b tree`,
						},
					},
				},
				want: &git{
					ParentBranches: map[string]string{
						"tree":  "some-branch",
						"other": "other-branch",
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
				name: "undo requires args",
				etc: &commandtest.ExecuteTestCase{
					Args:       []string{"ua"},
					WantStderr: "Argument \"FILE\" requires at least 1 argument, got 0\n",
					WantErr:    fmt.Errorf(`Argument "FILE" requires at least 1 argument, got 0`),
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
						filesArg.Name(): []string{
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
						filesArg.Name(): []string{
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
						filesArg.Name(): []string{
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
						repoUrl.Name(): "test-repo",
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
						repoUrl.Name(): "test-repo",
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
						repoUrl.Name():  "test-repo",
						mainFlag.Name(): true,
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
					WantStdout: "https://github.com/user/repo/compare/trunk...tree-branch?expanded=1\n",
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
					WantStdout: "https://github.com/user/repo/compare/maine...tree-branch?expanded=1\n",
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
					WantStdout: "https://github.com/user/repo/compare/trunk...tree-branch?expanded=1\n",
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
					WantStdout: "https://github.com/user/repo/compare/trunk...tree-branch?expanded=1\n",
				},
			},
			/* Useful for commenting out tests. */
		} {
			t.Run(fmt.Sprintf("[%s] %s", curOS.Name(), test.name), func(t *testing.T) {
				commandtest.StubValue(t, &sourcerer.CurrentOS, curOS)
				if oschk, ok := test.osChecks[curOS.Name()]; ok {
					if test.etc.WantExecuteData == nil {
						test.etc.WantExecuteData = &command.ExecuteData{}
					}
					test.etc.WantExecuteData.Executable = oschk.wantExecutable
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
		name string
		ctc  *commandtest.CompleteTestCase
	}{
		{
			name: "Completions for diff",
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd d ",
				SkipDataCheck: true,
				Want: &command.Autocompletion{
					Suggestions: []string{"abc", "def"},
				},
				WantRunContents: []*commandtest.RunContents{{
					Name: "git",
					Args: []string{"diff", "--name-only", "--relative"},
				}},
				RunResponses: []*commandtest.FakeRun{{
					Stdout: []string{"abc", "def"},
				}},
			},
		},
		{
			name: "Completions for diff (case insensitve)",
			ctc: &commandtest.CompleteTestCase{
				Args:          "cmd d A",
				SkipDataCheck: true,
				Want: &command.Autocompletion{
					Suggestions: []string{"abc"},
				},
				WantRunContents: []*commandtest.RunContents{{
					Name: "git",
					Args: []string{"diff", "--name-only", "--relative"},
				}},
				RunResponses: []*commandtest.FakeRun{{
					Stdout: []string{"abc", "def"},
				}},
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
	} {
		t.Run(test.name, func(t *testing.T) {
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
