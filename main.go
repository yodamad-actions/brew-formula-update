package main

import (
	"context"
	"encoding/json"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v68/github"
	"github.com/sethvargo/go-githubactions"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

func main() {

	// get input
	formulaFile := githubactions.GetInput("file")  // "/slidesk.rb"
	owner := githubactions.GetInput("owner")       // "yodamad"
	homebrewRepo := githubactions.GetInput("repo") // "homebrew-tools"
	version := githubactions.GetInput("version")
	fields := githubactions.GetInput("fields")
	token := githubactions.GetInput("token")

	updateFormula(formulaFile, owner, homebrewRepo, version, fields, token)
}

func updateFormula(formulaFile, owner, homebrewRepo, version, fields, token string) {
	client := github.NewClient(nil).WithAuthToken(token)
	ctx := context.Background()
	workdir := "/tmp/foo"
	filePath := workdir + "/" + formulaFile

	repo, _, err := client.Repositories.Get(ctx, owner, homebrewRepo)
	if err != nil {
		log.Fatalf("Cannot get repository %s/%s : %v", owner, homebrewRepo, err)
		return
	}

	gitRepo, err := git.PlainClone(workdir, false, &git.CloneOptions{
		URL:      repo.GetCloneURL(),
		Progress: os.Stdout,
	})
	if err != nil {
		log.Fatalf("Cannot clone repository : %v", err)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Cannot open %s : %v", formulaFile, err)
		return
	}
	defer file.Close()

	workTree, err := gitRepo.Worktree()
	if err != nil {
		log.Fatalf("Cannot get git work tree : %v", err)
		return
	}

	// Create a new branch
	branchName := "pr-" + version
	branchRefName := plumbing.NewBranchReferenceName(branchName)
	branchCoOpts := git.CheckoutOptions{
		Branch: branchRefName,
		Force:  true,
		Create: true,
	}
	if err := workTree.Checkout(&branchCoOpts); err != nil {
		log.Fatalf("Cannot checkout new branch : %v", err)
		return
	}

	versionPattern := `\d+\.*\d*\.*\d*`
	shaPattern := "\"[a-zA-Z0-9]*\""
	versionRegex := regexp.MustCompile(versionPattern)
	shaRegex := regexp.MustCompile(shaPattern)

	input, err := os.ReadFile(filePath)
	lines := strings.Split(string(input), "\n")

	outputs := getValues(fields)

	for i, line := range lines {
		if strings.Contains(line, "version") {
			lines[i] = versionRegex.ReplaceAllString(line, version)
		}
		if sha := getValue(line, outputs); sha != "" {
			lines[i] = shaRegex.ReplaceAllString(line, "\""+sha+"\"")
		}
	}
	output := strings.Join(lines, "\n")
	err = os.WriteFile(filePath, []byte(output), 0644)
	if err != nil {
		log.Fatalf("Cannot update %s file : %v", formulaFile, err)
		return
	}
	workTree.Add(".")

	commit, err := workTree.Commit("Update to "+version, &git.CommitOptions{
		Author: &object.Signature{
			Name: "GitHub Action",
			When: time.Now(),
		},
	})
	if err != nil {
		log.Fatalf("Cannot commit : %v", err)
		return
	}

	_, err = gitRepo.CommitObject(commit)
	if err != nil {
		log.Fatalf("Cannot commit : %v", err)
		return
	}

	err = gitRepo.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: owner, // This can be anything except an empty string
			Password: token,
		},
	})
	if err != nil {
		log.Fatalf("Cannot push : %v", err)
		return
	}

	pr := &github.NewPullRequest{
		Title:               github.Ptr("Update " + formulaFile + " to " + version),
		Head:                github.Ptr(branchName),
		Base:                github.Ptr("main"),
		Body:                github.Ptr("Update " + formulaFile + " formula version and sha256"),
		MaintainerCanModify: github.Ptr(true),
	}
	_, _, err = client.PullRequests.Create(ctx, owner, homebrewRepo, pr)
	if err != nil {
		log.Fatalf("Cannot create Pull Request : %v", err)
		return
	}
}

func getValues(fields string) map[string]string {
	// a map container to decode the JSON structure into
	c := make(map[string]json.RawMessage)
	// unmarschal JSON
	err := json.Unmarshal([]byte(fields), &c)

	// panic on error
	if err != nil {
		log.Fatalf("Cannot unmarshal fields value : %v", err)
		return nil
	}
	all := make(map[string]string)

	// iteration counter
	i := 0

	// copy c's keys into k
	for s := range c {
		data, _ := c[s].MarshalJSON()
		pp := strings.TrimSuffix(strings.TrimPrefix(string(data), "\""), "\"")
		pps := strings.Split(pp, "-")
		all[pps[0]] = pps[1]
		i++
	}
	return all
}

func getValue(line string, mapping map[string]string) string {
	for k := range mapping {
		if strings.Contains(line, k) {
			return mapping[k]
		}
	}
	return ""
}
