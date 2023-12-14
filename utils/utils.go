package utils

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
)

func RepoDir() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Fatalln("gibo can't determine your user cache directory. Please file an issue at https://github.com/simonwhitaker/gibo/issues")
	}
	return filepath.Join(cacheDir, "gibo")
}

func cloneRepo(repo string) error {
	err := os.MkdirAll(repo, 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	_, err = git.PlainClone(repo, false, &git.CloneOptions{
		URL:   "https://github.com/github/gitignore.git",
		Depth: 1,
	})
	if err != nil && err != git.ErrRepositoryAlreadyExists {
		return err
	}
	return nil
}

func cloneIfNeeded() error {
	fileInfo, err := os.Stat(RepoDir())
	if os.IsNotExist(err) {
		err = cloneRepo(RepoDir())
		if err != nil {
			return err
		}
	} else if !fileInfo.IsDir() {
		return fmt.Errorf("%v exists but is not a directory", RepoDir())
	}
	return nil
}

// pathForBoilerplate returns the filepath for a given boilerplate name. The
// search is case-insensitive; passing `python` to `name` will find the path to
// `Python.gitignore`, for example.
func pathForBoilerplate(name string) (string, error) {
	filename := name + ".gitignore"
	var result string = ""
	filepath.WalkDir(RepoDir(), func(path string, d fs.DirEntry, err error) error {
		if strings.ToLower(filepath.Base(path)) == strings.ToLower(filename) {
			result = path
			// Exit WalkDir early, we've found our match
			return filepath.SkipAll
		}
		return nil
	})
	if len(result) > 0 {
		return result, nil
	}
	return "", fmt.Errorf("%v: boilerplate not found", name)
}

func PrintBoilerplate(name string) error {
	if err := cloneIfNeeded(); err != nil {
		return err
	}
	path, err := pathForBoilerplate(name)
	if err != nil {
		return err
	}

	// Get the revision hash for the current head
	r, err := git.PlainOpen(RepoDir())
	if err != nil {
		return err
	}
	revision, err := r.Head()
	if err != nil {
		return err
	}
	remoteWebRoot := "https://raw.github.com/github/gitignore/" + revision.Hash().String()
	relativePath := strings.TrimPrefix(path, RepoDir())
	// On Windows, relativePath will use backslashes, but we need to append it to
	// a URL, which uses forward slashes.
	if os.PathSeparator == '\\' {
		relativePath = strings.ReplaceAll(relativePath, "\\", "/")
	}

	fmt.Println("### Generated by gibo (https://github.com/simonwhitaker/gibo)")
	fmt.Printf("### %v%v\n\n", remoteWebRoot, relativePath)
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	io.Copy(os.Stdout, f)
	return nil
}

func ListBoilerplates() ([]string, error) {
	var result []string
	if err := cloneIfNeeded(); err != nil {
		return nil, err
	}
	err := filepath.WalkDir(RepoDir(), func(path string, d fs.DirEntry, err error) error {
		base := filepath.Base(path)
		ext := filepath.Ext(base)
		if ext == ".gitignore" {
			name := strings.TrimSuffix(base, ext)
			result = append(result, name)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(result)
	return result, nil
}

func ListBoilerplatesNoError() []string {
	result, _ := ListBoilerplates()
	return result
}

func Update() (string, error) {
	cloneIfNeeded()
	r, err := git.PlainOpen(RepoDir())
	if err != nil {
		return "", err
	}
	w, err := r.Worktree()
	if err != nil {
		return "", err
	}
	err = w.Pull(&git.PullOptions{})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return "", err
	} else if err != nil {
		return "Already up to date", nil
	}
	return "Updated", nil
}
