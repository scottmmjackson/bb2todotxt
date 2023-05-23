package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kirsle/configdir"
)

const VERSION = "v0.0.3"

type bitbucketConfig struct {
	Username string `json:username`
	Password string `json:password`
}

const PACKAGE_NAME = "com.scottmmjackson.bb2todotxt"

var WELL_KNOWN_CONFIG_FILE_PATH = filepath.Join(configdir.LocalConfig(PACKAGE_NAME), "bitbucket.json")

func resolveConfigFile(commandLinePath string) (*bitbucketConfig, error) {
	var err error
	var bitbucketConfigString []byte
	var bitbucketConfigMap *bitbucketConfig
	if commandLinePath != "" {
		_, err = os.Stat(commandLinePath)
		if err != nil {
			return nil, fmt.Errorf("Unable to open config file at %s: %s", WELL_KNOWN_CONFIG_FILE_PATH, err)
		}
		bitbucketConfigString, err = os.ReadFile(commandLinePath)
		if err != nil {
			return nil, fmt.Errorf("Unable to upen config file at %s: %s", commandLinePath, err)
		}
	} else {
		_, err = os.Stat(WELL_KNOWN_CONFIG_FILE_PATH)
		if err != nil {
			return nil, fmt.Errorf("Unable to open config file at %s: %s", WELL_KNOWN_CONFIG_FILE_PATH, err)
		}
		bitbucketConfigString, err = os.ReadFile(WELL_KNOWN_CONFIG_FILE_PATH)
		if err != nil {
			return nil, fmt.Errorf("Unable to open config file at %s: %s", WELL_KNOWN_CONFIG_FILE_PATH, err)
		}
	}
	err = json.Unmarshal(bitbucketConfigString, &bitbucketConfigMap)
	if err != nil {
		return nil, err
	}
	return bitbucketConfigMap, nil
}

func commandLine() (*bitbucketConfig, string, string, int, error) {
	var bitbucketConfigMap *bitbucketConfig
	version := flag.Bool("v", false, "print version and quit")
	slug := flag.String("slug", "", "repo slug")
	owner := flag.String("owner", "", "repo owner")
	id := flag.Int("id", 0, "pull request id")
	bitbucketConfigFile := flag.String("config", "", "Bitbucket config file")
	flag.Parse()
	if *version {
		fmt.Println(VERSION)
		os.Exit(0)
	}
	bitbucketConfigMap, err := resolveConfigFile(*bitbucketConfigFile)
	if err != nil {
		return nil, "", "", 0, err
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
	}
	defer resp.Body.Close()
	if resp.StatusCode > 300 {
		log.Fatalf("error: %s", resp.Status)
	}
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
	var taskChunk []Task
	for uri != "" {
		taskChunk, uri = getTasks(bitbucketConfig, uri)
		tasks = append(tasks, taskChunk...)
	}
	todoTmpl.Execute(os.Stdout, tasks)
}
