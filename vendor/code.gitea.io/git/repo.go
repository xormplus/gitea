// Copyright 2015 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package git

import (
	"bytes"
	"container/list"
	"errors"
	"os"
	"path"
	"path/filepath"
	"time"
)

// Repository represents a Git repository.
type Repository struct {
	Path string

	commitCache *ObjectCache
	tagCache    *ObjectCache
}

const prettyLogFormat = `--pretty=format:%H`

func (repo *Repository) parsePrettyFormatLogToList(logs []byte) (*list.List, error) {
	l := list.New()
	if len(logs) == 0 {
		return l, nil
	}

	parts := bytes.Split(logs, []byte{'\n'})

	for _, commitID := range parts {
		commit, err := repo.GetCommit(string(commitID))
		if err != nil {
			return nil, err
		}
		l.PushBack(commit)
	}

	return l, nil
}

// IsRepoURLAccessible checks if given repository URL is accessible.
func IsRepoURLAccessible(url string) bool {
	_, err := NewCommand("ls-remote", "-q", "-h", url, "HEAD").Run()
	if err != nil {
		return false
	}
	return true
}

// InitRepository initializes a new Git repository.
func InitRepository(repoPath string, bare bool) error {
	os.MkdirAll(repoPath, os.ModePerm)

	cmd := NewCommand("init")
	if bare {
		cmd.AddArguments("--bare")
	}
	_, err := cmd.RunInDir(repoPath)
	return err
}

// OpenRepository opens the repository at the given path.
func OpenRepository(repoPath string) (*Repository, error) {
	repoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, err
	} else if !isDir(repoPath) {
		return nil, errors.New("no such file or directory")
	}

	return &Repository{
		Path:        repoPath,
		commitCache: newObjectCache(),
		tagCache:    newObjectCache(),
	}, nil
}

// CloneRepoOptions options when clone a repository
type CloneRepoOptions struct {
	Timeout time.Duration
	Mirror  bool
	Bare    bool
	Quiet   bool
	Branch  string
}

// Clone clones original repository to target path.
func Clone(from, to string, opts CloneRepoOptions) (err error) {
	toDir := path.Dir(to)
	if err = os.MkdirAll(toDir, os.ModePerm); err != nil {
		return err
	}

	cmd := NewCommand("clone")
	if opts.Mirror {
		cmd.AddArguments("--mirror")
	}
	if opts.Bare {
		cmd.AddArguments("--bare")
	}
	if opts.Quiet {
		cmd.AddArguments("--quiet")
	}
	if len(opts.Branch) > 0 {
		cmd.AddArguments("-b", opts.Branch)
	}
	cmd.AddArguments(from, to)

	if opts.Timeout <= 0 {
		opts.Timeout = -1
	}

	_, err = cmd.RunTimeout(opts.Timeout)
	return err
}

// PullRemoteOptions options when pull from remote
type PullRemoteOptions struct {
	Timeout time.Duration
	All     bool
	Rebase  bool
	Remote  string
	Branch  string
}

// Pull pulls changes from remotes.
func Pull(repoPath string, opts PullRemoteOptions) error {
	cmd := NewCommand("pull")
	if opts.Rebase {
		cmd.AddArguments("--rebase")
	}
	if opts.All {
		cmd.AddArguments("--all")
	} else {
		cmd.AddArguments(opts.Remote)
		cmd.AddArguments(opts.Branch)
	}

	if opts.Timeout <= 0 {
		opts.Timeout = -1
	}

	_, err := cmd.RunInDirTimeout(opts.Timeout, repoPath)
	return err
}

// Push pushs local commits to given remote branch.
func Push(repoPath, remote, branch string) error {
	_, err := NewCommand("push", remote, branch).RunInDir(repoPath)
	return err
}

// CheckoutOptions options when heck out some branch
type CheckoutOptions struct {
	Timeout   time.Duration
	Branch    string
	OldBranch string
}

// Checkout checkouts a branch
func Checkout(repoPath string, opts CheckoutOptions) error {
	cmd := NewCommand("checkout")
	if len(opts.OldBranch) > 0 {
		cmd.AddArguments("-b")
	}

	if opts.Timeout <= 0 {
		opts.Timeout = -1
	}

	cmd.AddArguments(opts.Branch)

	if len(opts.OldBranch) > 0 {
		cmd.AddArguments(opts.OldBranch)
	}

	_, err := cmd.RunInDirTimeout(opts.Timeout, repoPath)
	return err
}

// ResetHEAD resets HEAD to given revision or head of branch.
func ResetHEAD(repoPath string, hard bool, revision string) error {
	cmd := NewCommand("reset")
	if hard {
		cmd.AddArguments("--hard")
	}
	_, err := cmd.AddArguments(revision).RunInDir(repoPath)
	return err
}

// MoveFile moves a file to another file or directory.
func MoveFile(repoPath, oldTreeName, newTreeName string) error {
	_, err := NewCommand("mv").AddArguments(oldTreeName, newTreeName).RunInDir(repoPath)
	return err
}
