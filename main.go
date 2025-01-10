package main

import (
	"context"
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
	sha := githubactions.GetInput("sha256")
	field := githubactions.GetInput("field")
	token := githubactions.GetInput("token")

	updateFormula(formulaFile, owner, homebrewRepo, version, sha, field, token)
}

func updateFormula(formulaFile, owner, homebrewRepo, version, sha, field, token string) {
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

	for i, line := range lines {
		if strings.Contains(line, "version") {
			lines[i] = versionRegex.ReplaceAllString(line, version)
		}
		if strings.Contains(line, field) {
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
	}
}
