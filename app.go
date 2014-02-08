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
		fmt.Print(err)
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
	repository Repository
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
	runCommand(r.Conf.TmpDir, "git", "clone", r.Origin, r.Name)
	runCommand(r.RepositoryDir(), "git", "remote", "add", "github", r.Github)
}

func (r *Repository) Checkout(branch string) {
	runCommand(r.RepositoryDir(), "git", "checkout", branch)
}

func (r *Repository) Pull(branch string) {
	runCommand(r.RepositoryDir(), "git", "pull", "github", branch)
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
		if r.Name == data.repository.Name {
			repo = r
			break
		}
	}
	repo.Conf = RepositoryConfig
	repo.PullDefault()
	repo.PushDefault()
	for _, branch := range repo.Branches {
		repo.Pull(branch)
		repo.Push(branch)
	}
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
	fmt.Print(RepositoryConfig)
	for _, repo := range RepositoryConfig.Repositories {
		repo.Conf = RepositoryConfig
		repo.InitRepository()
	}
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
