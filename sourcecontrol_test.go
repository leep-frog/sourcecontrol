package sourcecontrol

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command"
)

func repoRunContents() []string {
	return []string{"set -e", "set -o pipefail", "git rev-parse --show-toplevel | xargs basename"}
}

func TestExecution(t *testing.T) {
	for _, test := range []struct {
		name string
		g    *git
		want *git
		etc  *command.ExecuteTestCase
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
			etc: &command.ExecuteTestCase{
				Args: []string{"edo"},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{"my previous commit message"},
				}},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					`git log -1 --pretty=%B`,
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
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					`git log -1 --pretty=%B`,
				}},
				WantStderr: "failed to get previous commit message: failed to execute bash command: oops\n",
				WantErr:    fmt.Errorf("failed to get previous commit message: failed to execute bash command: oops"),
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
		// Checkout main
		{
			name: "checkout main",
			etc: &command.ExecuteTestCase{
				Args: []string{"m"},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{"test-repo"},
				}},
				WantRunContents: [][]string{repoRunContents()},
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
				WantRunContents: [][]string{repoRunContents()},
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
				WantRunContents: [][]string{repoRunContents()},
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
				WantRunContents: [][]string{repoRunContents()},
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
				WantRunContents: [][]string{repoRunContents()},
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
				WantRunContents: [][]string{repoRunContents()},
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
				WantStderr: "Unprocessed extra args: [limb]\n",
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
				WantStderr: "Unprocessed extra args: [limb]\n",
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
						`git reset file.one some/where/file.2`,
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
				WantRunContents: [][]string{repoRunContents()},
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
				WantRunContents: [][]string{repoRunContents()},
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
				WantRunContents: [][]string{repoRunContents()},
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
			name: "diff with whitespace flag",
			etc: &command.ExecuteTestCase{
				Args: []string{"d", "-w"},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{"test-repo"},
				}},
				WantRunContents: [][]string{repoRunContents()},
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
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.g == nil {
				test.g = CLI()
			}
			test.etc.Node = test.g.Node()
			command.ExecuteTest(t, test.etc)
			command.ChangeTest(t, test.want, test.g, cmpopts.IgnoreUnexported(git{}), cmpopts.EquateEmpty())
		})
	}
}
