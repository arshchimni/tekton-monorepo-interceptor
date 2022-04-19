package diff

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/arshchimni/tekton-monorepo-interceptor/log"
	"github.com/google/go-github/v43/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetChangedFiles(t *testing.T) {
	cases := []struct {
		event github.PushEvent
		want  []string
	}{
		{
			github.PushEvent{
				Before: github.String("beforeRef"),
				After:  github.String("afterRef"),
				Repo: &github.PushEventRepository{
					FullName: github.String("test/platform"),
				},
			},
			[]string{"folder/subfolder1/file",
				"folder/subfolder2/file",
				"folder/subfolder3/file"},
		},
	}
	for _, tc := range cases {
		client, apiHandler, teardown := setup()
		defer teardown()

		d := DiffImpl{
			client: client,
			logger: log.NewDiscard(),
		}

		pattern := fmt.Sprintf("/repos/%s/compare/%s...%s", tc.event.Repo.GetFullName(), tc.event.GetBefore(), tc.event.GetAfter())
		apiHandler.HandleFunc(pattern, func(w http.ResponseWriter, req *http.Request) {
			expectedURL := fmt.Sprintf("/repos/%s/compare/%s...%s", tc.event.Repo.GetFullName(), tc.event.GetBefore(), tc.event.GetAfter())
			require.True(t, strings.HasPrefix(req.URL.Path, expectedURL), "url path '%s' does not match expected '%s'", req.URL.Path, expectedURL)

			res := github.CommitsComparison{
				Files: []*github.CommitFile{
					{Filename: github.String("folder/subfolder1/file")},
					{Filename: github.String("folder/subfolder2/file")},
					{Filename: github.String("folder/subfolder3/file")},
				},
			}

			b, err := json.Marshal(res)
			require.NoError(t, err)

			_, err = w.Write(b)
			require.NoError(t, err)
		})

		got, err := d.GetChangedFiles(context.Background(), &tc.event)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		if !assert.Equal(t, got, tc.want) {
			t.Fatalf("got %v but want %v", got, tc.want)
		}
	}
}

// setup sets up a test HTTP server along with a github.Client that is
// configured to talk to that test server. Tests should register handlers on
// mux which provide mock responses for the API method being tested.
func setup() (client *github.Client, apiHandler *http.ServeMux, teardown func()) {
	apiHandler = http.NewServeMux()

	// server is a test HTTP server used to provide mock API responses.
	server := httptest.NewServer(apiHandler)

	// client is the GitHub client being tested and is
	// configured to use test server.
	client = github.NewClient(nil)
	url, _ := url.Parse(server.URL + "/")
	client.BaseURL = url

	return client, apiHandler, server.Close
}
