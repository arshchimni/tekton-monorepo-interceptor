package diff

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v43/github"
	"go.uber.org/zap"
)

type Diff interface {
	GetChangedFiles(ctx context.Context, event *github.PushEvent) ([]string, error)
}

type DiffImpl struct {
	client *github.Client
	logger *zap.Logger
}

func NewDiff(logger *zap.Logger) (Diff, error) {
	httpClient := &http.Client{}
	ghClient := github.NewClient(httpClient)

	return &DiffImpl{
		client: ghClient,
		logger: logger,
	}, nil
}

func (g *DiffImpl) GetChangedFiles(ctx context.Context, event *github.PushEvent) ([]string, error) {
	repoName := event.Repo.GetFullName()

	split := strings.Split(repoName, "/")
	if len(split) != 2 {
		return nil, fmt.Errorf("repo name not in format <owner>/<repo> %s", repoName)
	}

	beforeSHA := event.GetBefore()
	afterSHA := event.GetAfter()

	compare, _, err := g.client.Repositories.CompareCommits(ctx, split[0], split[1], beforeSHA, afterSHA, &github.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("compare %s/%s %s to %s: %w", split[0], split[1], beforeSHA, afterSHA, err)
	}

	var changedFiles []string
	for _, f := range compare.Files {
		if f.GetFilename() != "" {
			changedFiles = append(changedFiles, f.GetFilename())
		}
	}

	return changedFiles, nil
}
