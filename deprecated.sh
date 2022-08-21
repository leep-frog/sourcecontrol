#!/bin/bash

# Functions without autocomplete requirements
alias gp='git push' # done
alias gpl='git pull' # done
alias gs='git status' # done
alias gsp='git status --porcelain' # don't need
alias guco='git reset HEAD~'  # undo commit // done
alias gb='git branch' # done
alias gc='_commit' # done
alias gcnv='_commit_no_verify' # done
alias cm='git checkout master' # done
alias gcb='git checkout -b' # done
alias gmm='git merge master' # done

function gcp {
  _commit "$*" && gp && echo SUCCESS!
}

function _commit { # done
  git commit -m "$*"
}

function _commit_no_verify { # done
  git commit -m "$*" --no-verify
}

# Commands with autocomplete setup
alias gdm='git diff master' # done
alias gd='git diff --' # done
alias ga='git add' # done
alias guc='git checkout --'  # undo change // done
alias gua='git reset'  # git undo add // done
alias ch='git checkout' # done
alias gbd='git branch -d' # irrelevant
alias gbd='git branch -D' # irrelevant

function _with_prefix {
  regex="^$1 "
  results="$(git status --porcelain | grep "$regex" | cut -c 4-)"

  # Now make results relative to current directory
  relative_results=""
  toplevel="$(git rev-parse --show-toplevel)"
  for git_path in $results
  do
    full_path="$toplevel/$git_path"
    path="$(realpath --relative-to="." "$full_path")"
    relative_results="$relative_results $path"
  done

  # Now do compreply stuff
  _complete "$results"
}

function _ga {
  _with_prefix '.[^ ]'
}

function _gd {
  _complete "$(git diff --name-only --relative)"
}

function _gr {
  _complete "$(git diff --cached --name-only --relative)"
}

complete -F _ga ga
complete -F _gd gd
complete -F _gr gua
