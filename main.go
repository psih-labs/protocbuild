package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/go-github/v34/github"
	"github.com/otiai10/copy"
	"golang.org/x/oauth2"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	plumbing "github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"gopkg.in/yaml.v2"
)

type defaultLang struct {
	Name string `yaml:"name"`
	Args string `yaml:"args,omitempty"`
}

var commitMsg = "AutoUpdateGeneratedProto"
var workspaceRoot = GetEnv("WORKSPACE_ROOT", "./")
var (
	defaultRepoRoot    = "repos"
	defaultBranch      = "master"
	defaultOutput      = "gen"
	defaultRoot        = "pb"
	defaultGitHost     = "github.com"
	defaultGitUserName = "protobufbot"
	defaultGitEmail    = "protobufbot@github.com"
)

type conf struct {
	Root   string `yaml:"root"`
	Output string `yaml:"output"`
	Git    struct {
		Org      string `yaml:"org"`
		Reporoot string `yaml:"reporoot"`
		Host     string `yaml:"host"`
		Branch   string `yaml:"branch"`
		Token    string `yaml:"token"`
	} `yaml:"git"`
	Sources []struct {
		Name      string   `yaml:"name"`
		Languages []string `yaml:"languages"`
	} `yaml:"sources"`
	DefaultLang []defaultLang `yaml:"default_lang"`
}

func main() {
	fmt.Println("Workspace Root ", workspaceRoot)
	var c conf
	c.getConf()
	if len(c.Root) == 0 {
		c.Root = defaultRoot
	}
	if len(c.Output) == 0 {
		c.Output = defaultOutput
	}

	if len(c.Git.Org) == 0 {
		c.Git.Org = GetEnv("GIT_ORG", "")
	}
	if len(c.Git.Host) == 0 {
		c.Git.Host = GetEnv("GIT_HOST", defaultGitHost)
	}
	if len(c.Git.Reporoot) == 0 {
		c.Git.Reporoot = defaultRepoRoot
	}
	if len(c.Git.Branch) == 0 {
		c.Git.Branch = defaultBranch
	}
	cleanup(c)
	projectName := GetEnv("GIT_REPO", "my-project-name")
	fmt.Println("Opening Dir ", workspaceRoot+c.Root)
	file, err := os.Open(workspaceRoot + c.Root)
	if err != nil {
		log.Fatalf("failed opening directory: %s", err)
	}
	defer file.Close()

	list, _ := file.Readdirnames(0) // 0 to read all files and folders

	var reponames []string
	for _, name := range list {
		target := name
		var langs []defaultLang
		for _, s := range c.Sources {
			if s.Name == name {
				for _, l := range s.Languages {
					langs = append(langs, defaultLang{Name: l, Args: findlang(c, l).Args})
				}
			}
		}
		if len(langs) == 0 {
			langs = c.DefaultLang
		}
		for _, l := range langs {
			targetfolder := defaultRoot + "-" + projectName + "-" + l.Name + "-" + target
			reponames = append(reponames, targetfolder)
			outDir := workspaceRoot + c.Output + "/" + targetfolder
			fmt.Println("Creating dir ", outDir)
			err := os.MkdirAll(outDir, 0755)
			if err != nil {
				log.Fatalf("Failed to create dir: %v", err)
			}
			files, err := ioutil.ReadDir(workspaceRoot + c.Root + "/" + target)
			if err != nil {
				log.Fatal(err)
			}
			var protocFiles []string
			for _, f := range files {
				protocFiles = append(protocFiles, workspaceRoot+c.Root+"/"+target+"/"+f.Name())
			}

			var command []string
			command = append(command, "-I"+workspaceRoot+c.Root)
			command = append(command, "--"+l.Name+"_out="+l.Args+outDir)
			command = append(command, protocFiles...)
			cmd := exec.Command("protoc", command...)
			outBytes, err := cmd.CombinedOutput()
			if err != nil {
				panic(string(outBytes))
			}
		}
	}
	setupGit(c, reponames)
}

func setupGit(c conf, reponames []string) {
	log.Println("Setting Up Git")
	gitToken := GetEnv("GIT_TOKEN", c.Git.Token)

	os.RemoveAll(workspaceRoot + c.Git.Reporoot)
	branch := c.Git.Branch
	if err := os.MkdirAll(c.Git.Reporoot, 0755); err != nil {
		log.Fatalf("Failed to create git folder: %v", err)
	}
	author := &object.Signature{
		Name:  GetEnv("GIT_USERNAME", defaultGitUserName),
		Email: GetEnv("GIT_EMAIL", defaultGitEmail),
		When:  time.Now(),
	}
	var gh *github.Client
	if c.Git.Host == "github.com" {

		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: gitToken},
		)
		tc := oauth2.NewClient(context.Background(), ts)

		gh = github.NewClient(tc)
	}

	for _, r := range reponames {
		newBranch := false
		log.Printf("Setting up repo %v", r)
		gitssh := "https://" + c.Git.Host + "/" + c.Git.Org + "/" + r + ".git "
		repopath := workspaceRoot + c.Git.Reporoot + "/" + r

		gitRepo, err := git.PlainClone(repopath, false, &git.CloneOptions{
			// The intended use of a GitHub personal access token is in replace of your password
			// because access tokens can easily be revoked.
			// https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/
			Auth: &http.BasicAuth{
				Username: "abc123", // yes, this can be anything except an empty string
				Password: gitToken,
			},
			URL:      gitssh,
			Progress: os.Stdout,
		})
		if err != nil {
			newBranch = true
			log.Printf("Remote Repo doesnt exists: %v", err)
			if err := os.MkdirAll(repopath, 0755); err != nil {
				log.Fatalf("Failed to create git proto repo folder: %v", err)
			}
			gitRepo, err = git.PlainInit(repopath, false)
			gitRepo.CreateRemote(&config.RemoteConfig{
				Name: "origin",
				URLs: []string{gitssh},
			})
		}

		w, err := gitRepo.Worktree()

		if err != nil {
			panic(err)
		}

		w.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(branch),
			Create: newBranch,
		})

		err = copy.Copy(workspaceRoot+c.Output+"/"+r, workspaceRoot+c.Git.Reporoot+"/"+r)

		if err != nil {
			panic(err)
		}

		err = w.AddWithOptions(&git.AddOptions{
			All: true,
		})
		if err != nil {
			panic(err)
		}
		commit, err := w.Commit(commitMsg, &git.CommitOptions{
			All:    true,
			Author: author,
		})
		if err != nil {
			panic(err)
		}
		_, err = gitRepo.CommitObject(commit)
		if err != nil {
			panic(err)
		}
		err = gitRepo.Push(&git.PushOptions{
			Auth: &http.BasicAuth{
				Username: "abc123", // yes, this can be anything except an empty string
				Password: gitToken,
			},
		})
		if err != nil {
			if c.Git.Host == "github.com" {
				repo := &github.Repository{
					Name:    &r,
					Private: github.Bool(true),
				}
				gh.Repositories.Create(context.Background(), c.Git.Org, repo)
				err = gitRepo.Push(&git.PushOptions{
					Auth: &http.BasicAuth{
						Username: "abc123", // yes, this can be anything except an empty string
						Password: gitToken,
					},
				})
				if err != nil {
					panic(err)
				}
			} else {
				panic(err)
			}
		} else {
			fmt.Println("Pushed repo", gitssh)
		}
	}
}
func cleanup(c conf) {
	os.RemoveAll(workspaceRoot + c.Output)
	os.RemoveAll(workspaceRoot + c.Git.Reporoot)
}

func (c *conf) getConf() *conf {
	yamlFile, err := ioutil.ReadFile(workspaceRoot + "protocbuild.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}
func runCmd(s string) error {
	fmt.Println(s)
	args := strings.Fields(s)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return err
}
func findlang(c conf, lang string) defaultLang {
	for _, l := range c.DefaultLang {
		if l.Name == lang {
			return l
		}
	}
	log.Fatalf("lang %v not found in yaml default_lang", lang)
	return defaultLang{}
}
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
func RootDir() string {
	_, b, _, _ := runtime.Caller(0)
	d := path.Join(path.Dir(b))
	return filepath.Dir(d)
}
