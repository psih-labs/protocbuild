package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v2"
)

type defaultLang struct {
	Name string `yaml:"name"`
	Args string `yaml:"args,omitempty"`
}

var commitMsg = "AutoUpdateGeneratedProto"
var workspaceRoot = os.Getenv("WORKSPACE_ROOT")
var tmpcmnds = workspaceRoot + "tmpcmnds"

type conf struct {
	Root   string `yaml:"root"`
	Output string `yaml:"output"`
	Git    struct {
		Org      string `yaml:"org"`
		Reporoot string `yaml:"reporoot"`
		Host     string `yaml:"host"`
		Branch   string `yaml:"branch"`
	} `yaml:"git"`
	Sources []struct {
		Name      string   `yaml:"name"`
		Languages []string `yaml:"languages"`
	} `yaml:"sources"`
	DefaultLang []defaultLang `yaml:"default_lang"`
}

func main() {
	var c conf
	c.getConf()
	cleanup(c)
	file, err := os.Open(workspaceRoot + c.Root)
	if err != nil {
		log.Fatalf("failed opening directory: %s", err)
	}
	defer file.Close()
	f := tmpcreate()

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
			targetfolder := "pb-" + l.Name + "-" + target
			reponames = append(reponames, targetfolder)
			outDir := workspaceRoot + c.Output + "/" + targetfolder
			err := runCmd("mkdir -p " + outDir)
			if err != nil {
				log.Fatalf("Failed to create dir: %v", err)
			}
			dindWorkspace := os.Getenv("DIND_WORKSPACE")

			if err != nil {
				log.Fatalf("Failed to get current dir: %v", err)
			}
			cmdstr := "docker run -v " + dindWorkspace + ":/workspace --rm grpckit protoc -I" + workspaceRoot + c.Root + " --" + l.Name + "_out=" + l.Args + outDir + " " + workspaceRoot + c.Root + "/" + target + "/*"
			tmpwrite(f, cmdstr)
		}
	}
	defer f.Close()
	tmprun()
	setupGit(c, reponames)
}

func setupGit(c conf, reponames []string) {
	log.Println("Setting Up Git")
	if err := runCmd("./setupgit.sh"); err != nil {
		log.Fatalf("Failed to run setupgitsh: %v", err)
	}
	os.RemoveAll(workspaceRoot + c.Git.Reporoot)
	branch := c.Git.Branch
	if err := runCmd("mkdir -p " + c.Git.Reporoot); err != nil {
		log.Fatalf("Failed to create git folder: %v", err)
	}
	f := tmpcreate()

	var newbranch []bool
	for _, r := range reponames {
		log.Printf("Setting up repo %v", r)
		gitssh := "git@" + c.Git.Host + ":" + c.Git.Org + "/" + r + ".git "
		repopath := workspaceRoot + c.Git.Reporoot + "/" + r
		if err := runCmd("git clone --single-branch --branch " + branch + " " + gitssh + repopath); err != nil {
			newbranch = append(newbranch, true)
			log.Printf("Remote Repo doesnt exists: %v", err)
			if err := runCmd("mkdir -p " + repopath); err != nil {
				log.Fatalf("Failed to create git proto repo folder: %v", err)
			}
			tmpwrite(f, "cd "+repopath)
			tmpwrite(f, "git init")
			tmpwrite(f, "git remote add origin "+gitssh)
			tmpwrite(f, "cd ../../")
		} else {
			newbranch = append(newbranch, false)
		}
	}
	tmpwrite(f, "rsync -a "+workspaceRoot+c.Output+"/ "+workspaceRoot+c.Git.Reporoot)
	for i, r := range reponames {
		upstream := ""
		newbr := ""
		if newbranch[i] == true {
			upstream = "-u"
			newbr = "-b"
		}
		repopath := workspaceRoot + c.Git.Reporoot + "/" + r
		tmpwrite(f, "cd "+repopath)
		tmpwrite(f, "git checkout "+newbr+" "+branch)
		tmpwrite(f, "git add -A")
		tmpwrite(f, "git commit --allow-empty -m "+commitMsg)
		tmpwrite(f, "git push "+upstream+" origin "+branch)
		//tmpwrite(f, "cd ../../")
	}
	tmprun()
	defer f.Close()
}
func cleanup(c conf) {
	os.RemoveAll(workspaceRoot + c.Output)
	os.RemoveAll(workspaceRoot + c.Git.Reporoot)
	os.Remove(workspaceRoot + tmpcmnds)
}

func tmpcreate() *os.File {
	os.Remove(tmpcmnds)
	f, err := os.OpenFile(tmpcmnds, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed create new dir: %v", err)
	}
	return f
}
func tmprun() {
	if err := runCmd("./run.sh"); err != nil {
		log.Fatalf("Failed to run sh: %v", err)
	}
}
func tmpwrite(f *os.File, cmdstr string) {
	if _, err := f.Write([]byte(cmdstr + "\n")); err != nil {
		log.Fatalf("Failed to write to file: %v", err)
	}
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
