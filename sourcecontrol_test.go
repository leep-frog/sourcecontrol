package sourcecontrol

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
	"github.com/leep-frog/functional"
	"golang.org/x/exp/slices"
)

func repoRunContents() *command.RunContents {
	return &command.RunContents{
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

	u, err := command.Use(CLI().Node(), command.ParseExecuteArgs(nil))
	if err != nil {
		t.Fatalf("Failed to generate usage")
	}

	for _, curOS := range []sourcerer.OS{sourcerer.Linux(), sourcerer.Windows()} {
		for _, test := range []struct {
			name     string
			g        *git
			want     *git
			etc      *command.ExecuteTestCase
			osChecks map[string]*osCheck
		}{
			// TODO: Config tests
			// Simple command tests
			{
				name: "branch",
				etc: &command.ExecuteTestCase{
					Args: []string{"b"},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{"git branch"},
					},
				},
			},
			{
				name: "pull",
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
					Args: []string{"f"},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git fetch",
						},
					},
				},
			},
			// Git redo
			{
				name: "git redo succeeds",
				osChecks: map[string]*osCheck{
					"windows": {
						wantExecutable: []string{
							wCmd("guco"),
							wCmd("ga ."),
							wCmd(`gc "my previous commit message"`),
						},
					},
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"edo"},
					RunResponses: []*command.FakeRun{{
						Stdout: []string{"my previous commit message"},
					}},
					WantRunContents: []*command.RunContents{{
						Name: "git",
						Args: []string{
							"log",
							"-1",
							"--pretty=%B",
						},
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`guco && ga . && gc "my previous commit message"`,
						},
					},
				},
			},
			{
				name: "git redo fails",
				etc: &command.ExecuteTestCase{
					Args: []string{"edo"},
					RunResponses: []*command.FakeRun{{
						Err: fmt.Errorf("oops"),
					}},
					WantRunContents: []*command.RunContents{{
						Name: "git",
						Args: []string{"log", "-1", "--pretty=%B"},
					}},
					WantStderr: "failed to get previous commit message: failed to execute shell command: oops\n",
					WantErr:    fmt.Errorf("failed to get previous commit message: failed to execute shell command: oops"),
				},
			},
			// Git log
			{
				name: "git log with no args",
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
					Args: []string{"lg", "-d"},
					WantData: &command.Data{Values: map[string]interface{}{
						gitLogArg.Name():      1,
						gitLogDiffFlag.Name(): true,
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git diff HEAD~1",
						},
					},
				},
			},
			{
				name: "git log diff with args",
				etc: &command.ExecuteTestCase{
					Args: []string{"lg", "-d", "7"},
					WantData: &command.Data{Values: map[string]interface{}{
						gitLogArg.Name():      7,
						gitLogDiffFlag.Name(): true,
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git diff HEAD~7",
						},
					},
				},
			},
			// Git stash push/pop
			{
				name: "git stash push with no args",
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
					Args: []string{"m"},
					RunResponses: []*command.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*command.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoName.Name(): "test-repo",
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
				etc: &command.ExecuteTestCase{
					Args: []string{"m"},
					RunResponses: []*command.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*command.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoName.Name(): "test-repo",
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
				etc: &command.ExecuteTestCase{
					Args: []string{"m"},
					RunResponses: []*command.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*command.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoName.Name(): "test-repo",
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
				etc: &command.ExecuteTestCase{
					Args: []string{"m"},
					RunResponses: []*command.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*command.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoName.Name(): "test-repo",
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
				etc: &command.ExecuteTestCase{
					Args: []string{"m"},
					RunResponses: []*command.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*command.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoName.Name(): "test-repo",
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
				etc: &command.ExecuteTestCase{
					Args: []string{"mm"},
					RunResponses: []*command.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*command.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoName.Name(): "test-repo",
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
				etc: &command.ExecuteTestCase{
					Args: []string{"mm"},
					RunResponses: []*command.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*command.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoName.Name(): "test-repo",
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
				etc: &command.ExecuteTestCase{
					Args: []string{"mm"},
					RunResponses: []*command.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*command.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoName.Name(): "test-repo",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							"git merge mainest",
						},
					},
				},
			},
			// Commit
			{
				name: "commit requires args",
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
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
			// Commit & push
			{
				name: "commit and push requires args",
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
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
				name: "checkout branch requires arg",
				etc: &command.ExecuteTestCase{
					Args:       []string{"ch"},
					WantStderr: "Argument \"BRANCH\" requires at least 1 argument, got 0\n",
					WantErr:    fmt.Errorf(`Argument "BRANCH" requires at least 1 argument, got 0`),
				},
			},
			{
				name: "checkout branch requires one arg",
				etc: &command.ExecuteTestCase{
					Args: []string{"ch", "tree", "limb"},
					WantData: &command.Data{Values: map[string]interface{}{
						branchArg.Name(): "tree",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git checkout tree`,
						},
					},
					WantStderr: fmt.Sprintf("Unprocessed extra args: [limb]\n\n%s\n%s\n", command.UsageErrorSectionStart, u.String()),
					WantErr:    fmt.Errorf(`Unprocessed extra args: [limb]`),
				},
			},
			{
				name: "checks out a branch",
				etc: &command.ExecuteTestCase{
					Args: []string{"ch", "tree"},
					WantData: &command.Data{Values: map[string]interface{}{
						branchArg.Name(): "tree",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git checkout tree`,
						},
					},
				},
			},
			{
				name: "checks out a new branch",
				etc: &command.ExecuteTestCase{
					Args: []string{"ch", "tree", "-n"},
					WantData: &command.Data{Values: map[string]interface{}{
						branchArg.Name():     "tree",
						newBranchFlag.Name(): true,
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git checkout -b tree`,
						},
					},
				},
			},
			// Delete new branch
			{
				name: "delete branch requires arg",
				etc: &command.ExecuteTestCase{
					Args:       []string{"bd"},
					WantStderr: "Argument \"BRANCH\" requires at least 1 argument, got 0\n",
					WantErr:    fmt.Errorf(`Argument "BRANCH" requires at least 1 argument, got 0`),
				},
			},
			{
				name: "delete branch requires one arg",
				etc: &command.ExecuteTestCase{
					Args: []string{"bd", "tree", "limb"},
					WantData: &command.Data{Values: map[string]interface{}{
						branchArg.Name(): "tree",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git branch -d tree`,
						},
					},
					WantStderr: fmt.Sprintf("Unprocessed extra args: [limb]\n\n%s\n%s\n", command.UsageErrorSectionStart, u.String()),
					WantErr:    fmt.Errorf(`Unprocessed extra args: [limb]`),
				},
			},
			{
				name: "deletes a branch",
				etc: &command.ExecuteTestCase{
					Args: []string{"bd", "tree"},
					WantData: &command.Data{Values: map[string]interface{}{
						branchArg.Name(): "tree",
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git branch -d tree`,
						},
					},
				},
			},
			{
				name: "force deletes a branch",
				etc: &command.ExecuteTestCase{
					Args: []string{"bd", "-f", "tree"},
					WantData: &command.Data{Values: map[string]interface{}{
						branchArg.Name():   "tree",
						forceDelete.Name(): true,
					}},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git branch -D tree`,
						},
					},
				},
			},
			// Undo add
			{
				name: "undo requires args",
				etc: &command.ExecuteTestCase{
					Args:       []string{"ua"},
					WantStderr: "Argument \"FILE\" requires at least 1 argument, got 0\n",
					WantErr:    fmt.Errorf(`Argument "FILE" requires at least 1 argument, got 0`),
				},
			},
			{
				name: "undo resets files",
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
					Args:       []string{"uc"},
					WantStderr: "Argument \"FILE\" requires at least 1 argument, got 0\n",
					WantErr:    fmt.Errorf(`Argument "FILE" requires at least 1 argument, got 0`),
				},
			},
			{
				name: "undo change undoes changed files",
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
					Args: []string{"a"},
					WantExecuteData: &command.ExecuteData{
						Executable: []string{
							`git add .`,
						},
					},
				},
			},
			{
				name: "add with args args",
				etc: &command.ExecuteTestCase{
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
			// Diff
			{
				name: "diff with no args",
				etc: &command.ExecuteTestCase{
					Args: []string{"d"},
					RunResponses: []*command.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*command.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoName.Name(): "test-repo",
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
				etc: &command.ExecuteTestCase{
					Args: []string{"d", "this.file", "that/file/txt"},
					RunResponses: []*command.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*command.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoName.Name(): "test-repo",
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
				etc: &command.ExecuteTestCase{
					Args: []string{"d", "-m"},
					RunResponses: []*command.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*command.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoName.Name(): "test-repo",
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
				etc: &command.ExecuteTestCase{
					Args: []string{"d", "-c"},
					RunResponses: []*command.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*command.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoName.Name():       "test-repo",
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
				etc: &command.ExecuteTestCase{
					Args: []string{"d", "-w"},
					RunResponses: []*command.FakeRun{{
						Stdout: []string{"test-repo"},
					}},
					WantRunContents: []*command.RunContents{repoRunContents()},
					WantData: &command.Data{Values: map[string]interface{}{
						repoName.Name():       "test-repo",
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
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
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
				etc: &command.ExecuteTestCase{
					Args: []string{"cfg", "main", "show"},
					WantStdout: strings.Join([]string{
						"Global default branch: other-main",
						"deux: main-two",
						"un: main-one",
						"",
					}, "\n"),
				},
			},
		} {
			t.Run(fmt.Sprintf("[%s] %s", curOS.Name(), test.name), func(t *testing.T) {
				command.StubValue(t, &sourcerer.CurrentOS, curOS)
				if oschk, ok := test.osChecks[curOS.Name()]; ok {
					test.etc.WantExecuteData.Executable = oschk.wantExecutable
				}

				if test.g == nil {
					test.g = CLI()
				}
				test.etc.Node = test.g.Node()
				command.ExecuteTest(t, test.etc)
				command.ChangeTest(t, test.want, test.g, cmpopts.IgnoreUnexported(git{}), cmpopts.EquateEmpty())
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
		ctc       *command.CompleteTestCase
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
			ctc: &command.CompleteTestCase{
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
			ctc: &command.CompleteTestCase{
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
			ctc: &command.CompleteTestCase{
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
			ctc: &command.CompleteTestCase{
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
			test.ctc.RunResponses = []*command.FakeRun{{
				Stdout: statuses,
			}}
			test.ctc.WantRunContents = []*command.RunContents{{
				Name: "git",
				Args: []string{"status", "--porcelain=v2"},
			}}

			test.ctc.Want = functional.Map[*gitStatusFile, string](test.wantFiles, func(f *gitStatusFile) string { return f.name })
			slices.Sort(test.ctc.Want)

			command.CompleteTest(t, test.ctc)
		})
	}
}

func TestAutocomplete(t *testing.T) {
	for _, test := range []struct {
		name string
		ctc  *command.CompleteTestCase
	}{
		{
			name: "Completions for diff",
			ctc: &command.CompleteTestCase{
				Args:          "cmd d ",
				SkipDataCheck: true,
				Want:          []string{"abc", "def"},
				WantRunContents: []*command.RunContents{{
					Name: "git",
					Args: []string{"diff", "--name-only", "--relative"},
				}},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{"abc", "def"},
				}},
			},
		},
		{
			name: "Branch completions",
			ctc: &command.CompleteTestCase{
				Args:          "cmd ch ",
				SkipDataCheck: true,
				Want:          []string{"b-1", "b-3"},
				WantRunContents: []*command.RunContents{{
					Name: "git",
					Args: []string{"branch", "--list"},
				}},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{"  b-1 ", "* 	b-2", "		b-3		"},
				}},
			},
		},
		{
			name: "Handles no	branch completions",
			ctc: &command.CompleteTestCase{
				Args:          "cmd ch ",
				SkipDataCheck: true,
				WantRunContents: []*command.RunContents{{
					Name: "git",
					Args: []string{"branch", "--list"},
				}},
				RunResponses: []*command.FakeRun{{}},
			},
		},
		{
			name: "Handles branch completion error",
			ctc: &command.CompleteTestCase{
				Args:          "cmd ch ",
				SkipDataCheck: true,
				WantRunContents: []*command.RunContents{{
					Name: "git",
					Args: []string{"branch", "--list"},
				}},
				RunResponses: []*command.FakeRun{{
					Err: fmt.Errorf("oops"),
				}},
				WantErr: fmt.Errorf("failed to fetch autocomplete suggestions with shell command: failed to execute shell command: oops"),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := &git{}
			test.ctc.Node = g.Node()
			command.CompleteTest(t, test.ctc)
		})
	}
}

func TestAliasers(t *testing.T) {
	t.Run("GitAliasers() returns without issues", func(t *testing.T) {
		GitAliasers()
	})
}
