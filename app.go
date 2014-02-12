package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
)

func runCommand(dir, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	_, err := cmd.Output()
	if err != nil {
		fmt.Println(name, args)
		fmt.Println(err)
	}
}

type Repository struct {
	Name     string
	Github   string
	Origin   string
	Branches []string
	Conf     Config
}

type Config struct {
	Repositories []Repository
	TmpDir       string `json:"tmp_dir"`
}

type ResponseData struct {
	Repo Repository `json:"repository"`
}

var RepositoryConfig Config

func (r *Repository) RepositoryDir() string {
	return fmt.Sprintf("%s/%s", r.Conf.TmpDir, r.Name)
}

func (r *Repository) exists() (bool, error) {
	_, err := os.Stat(r.RepositoryDir())
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (r *Repository) InitRepository() {
	exists, _ := r.exists()
	if exists {
		return
	}
	fmt.Printf("Init repo: %s\n", r.Name)
	runCommand(r.Conf.TmpDir, "git", "clone", r.Origin, r.Name)
	runCommand(r.RepositoryDir(), "git", "remote", "add", "github", r.Github)
	fmt.Printf("Done\n")
}

func (r *Repository) Checkout(branch string) {
	runCommand(r.RepositoryDir(), "git", "checkout", branch)
}

func (r *Repository) Fetch() {
	runCommand(r.RepositoryDir(), "git", "fetch", "github")
}

func (r *Repository) Pull(branch string) {
	if branch == "master" {
		runCommand(r.RepositoryDir(), "git", "checkout", "master")
		runCommand(r.RepositoryDir(), "git", "pull", "github", "master")
		return
	}
	runCommand(r.RepositoryDir(), "git", "branch", "-d", branch)
	runCommand(r.RepositoryDir(), "git", "branch", branch, "github/"+branch)
}

func (r *Repository) PullDefault() {
	runCommand(r.RepositoryDir(), "git", "pull", "github")
}

func (r *Repository) Push(branch string) {
	runCommand(r.RepositoryDir(), "git", "push", "origin", branch)
}

func (r *Repository) PushDefault() {
	runCommand(r.RepositoryDir(), "git", "push", "origin")
}

func handler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var data ResponseData
	decoder.Decode(&data)
	var repo Repository
	for _, r := range RepositoryConfig.Repositories {
		if r.Name == data.Repo.Name {
			repo = r
			break
		}
	}
	repo.Conf = RepositoryConfig
	repo.Fetch()
	for _, branch := range repo.Branches {
		repo.Pull(branch)
	}
	repo.PushDefault()
}

func main() {
	fi, err := os.Open("config.json")
	if err != nil {
		panic(err)
	}
	// close fi on exit and check for its returned error
	defer func() {
		if err := fi.Close(); err != nil {
			panic(err)
		}
	}()
	// make a read buffer
	r := bufio.NewReader(fi)
	decoder := json.NewDecoder(r)
	decoder.Decode(&RepositoryConfig)
	for _, repo := range RepositoryConfig.Repositories {
		repo.Conf = RepositoryConfig
		repo.InitRepository()
	}
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
