package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"golang.org/x/term"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"syscall"

	"github.com/99designs/keyring"
)

type bitbucketConfig struct {
	Username string `json:username`
	Password string `json:password`
}

func interactive(msg string) string {
	r := bufio.NewReader(os.Stdin)
	fmt.Print(msg)
	s, _ := r.ReadString('\n')
	return s
}

func interactivePassword(msg string) string {
	fmt.Print(msg)
	b, _ := term.ReadPassword(int(syscall.Stdin))
	return string(b)
}

const SERVICE_NAME = "com.scottmmjackson.bb2todotxt"

func getBitbucketConfigFromKeychain() (*bitbucketConfig, error) {
	conf, err := keyring.Open(keyring.Config{
		ServiceName: SERVICE_NAME,
	})
	if err != nil {
		return nil, err
	}
	username, err := conf.Get("bitbucketUsername")
	if err != nil {
		return nil, err
	}
	password, err := conf.Get("bitbucketPassword")
	if err != nil {
		return nil, err
	}
	return &bitbucketConfig{
		Username: string(username.Data),
		Password: string(password.Data),
	}, nil
}

func interactiveBitbucketCredentials() (*bitbucketConfig, error) {
	username := strings.Trim(interactive("Bitbucket Username (blank or CTRL-C aborts):"), "\n")
	if username == "" {
		return nil, errors.New("No bitbucket username specified")
	}
	password := interactivePassword("Bitbucket App Password:")
	if password == "" {
		return nil, errors.New("No bitbucket password specified")
	}
	return &bitbucketConfig{
		Username: username,
		Password: password,
	}, nil
}
func commandLine() (*bitbucketConfig, string, string, int, error) {
	var bitbucketConfigMap *bitbucketConfig
	var err error

	slug := flag.String("slug", "", "repo slug")
	owner := flag.String("owner", "", "repo owner")
	id := flag.Int("id", 0, "pull request id")
	bitbucketConfigFile := flag.String("config", "", "Bitbucket config file")
	flag.Parse()
	if *bitbucketConfigFile == "" {
		bitbucketConfigMap, err = getBitbucketConfigFromKeychain()
		if err != nil {
			bitbucketConfigMap, err = interactiveBitbucketCredentials()
			if err != nil {
				return bitbucketConfigMap, "", "", 0, err
			}
		}
	} else {
		bitbucketConfigString, err := os.ReadFile(*bitbucketConfigFile)
		if err != nil {
			return bitbucketConfigMap, "", "", 0, err
		}
		err = json.Unmarshal(bitbucketConfigString, &bitbucketConfigMap)
		if err != nil {
			return bitbucketConfigMap, "", "", 0, err
		}
	}
	return bitbucketConfigMap, *slug, *owner, *id, nil
}

const BITBUCKET_URL = "bitbucket.org"

type Link struct {
	Href string `json:"href"`
}

type Links struct {
	Self Link `json:"self"`
	Html Link `json:"html"`
}

type AvatarLinks struct {
	Links
	Avatar Link `json:"avatar"`
}

type TaskCreator struct {
	DisplayName string `json:"display_name""`
	Links       AvatarLinks
	CreatorType string `json:"type"`
	Uuid        string `json:"uuid"`
	AccountId   string `json:"account_id""`
	Username    string `json:"username"`
}

type TaskContent struct {
	TaskType string `json:"type"`
	Raw      string `json:"raw"`
	Markup   string `json:"markup"`
	Html     string `json:"html"`
}

type Comment struct {
	Id    int   `json:"id"`
	Links Links `json:"links"`
}

type Task struct {
	Id        int         `json:"id"`
	State     string      `json:"state"`
	Content   TaskContent `json:"content"`
	Creator   TaskCreator `json:"creator"`
	Createdon string      `json:"created_on"`
	Updatedon string      `json:"updated_on"`
	Links     Links       `json:"links"`
	Comment   Comment     `json:"comment"`
}

type TaskResponse struct {
	Values   []Task `json:"values"`
	Pagelen  int    `json:"pagelen"`
	Size     int    `json:"size"`
	Page     int    `json:"page"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
}

func getTasks(bucketConfig *bitbucketConfig, uri string) ([]Task, string) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		log.Fatalf("error: %s", err)
	}
	req.SetBasicAuth(
		bucketConfig.Username,
		bucketConfig.Password,
	)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("error: %s", err)
	} else if resp.StatusCode >= 400 {
		log.Fatalf("error: %s", resp.Status)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("error: %s", err)
	}
	var taskResponse TaskResponse
	json.Unmarshal(body, &taskResponse)
	return taskResponse.Values, taskResponse.Next
}

var todoTmpl = template.Must(template.New("template").Parse(
	`{{ range . }}{{ if eq .State "UNRESOLVED"}}{{ .Content.Raw }} -- {{ .Comment.Links.Html.Href }}
{{else}}{{end}}{{end}}`,
))

func main() {
	bitbucketConfig, slug, owner, id, err := commandLine()
	if err != nil {
		flag.Usage()
		log.Fatalf("Unable to start: %s", err)
	}
	tasks := make([]Task, 0)
	uri := fmt.Sprintf(
		"https://%s/api/internal/repositories/%s/%s/pullrequests/%d/tasks",
		BITBUCKET_URL,
		owner, slug, id,
	)
	taskChunk, next := getTasks(bitbucketConfig, uri)
	for next != "" {
		tasks = append(tasks, taskChunk...)
		taskChunk, next = getTasks(bitbucketConfig, next)
	}
	todoTmpl.Execute(os.Stdout, tasks)
}
