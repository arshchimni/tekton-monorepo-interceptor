package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/arshchimni/tekton-monorepo-interceptor/diff"
	"github.com/arshchimni/tekton-monorepo-interceptor/log"
	"github.com/google/go-github/v43/github"
	"github.com/stretchr/testify/require"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tj/assert"
	"google.golang.org/grpc/codes"
)

type fakeDiff struct{}

func (f *fakeDiff) GetChangedFiles(ctx context.Context, event *github.PushEvent) ([]string, error) {
	return []string{"folder/subfolder1/file",
		"folder/subfolder2/file",
		"folder/subfolder3/file"}, nil
}

func newFakediff() diff.Diff {
	return &fakeDiff{}
}

func TestInterceptGitPayload(t *testing.T) {
	s := New(log.NewDiscard(), newFakediff())
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %s", err)
	}
	go func() {
		if err := s.Serve(ln); err != nil {
			t.Errorf("failed to Serve: %s", err)
		}
		defer s.server.Close()
	}()

	testcases := map[string]string{
		"monorepo": "/monorepo",
	}
	for name, path := range testcases {
		t.Run(name, func(t *testing.T) {
			request, err := getInterceptorRequest()
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, "http://"+ln.Addr().String()+path, request)
			require.NoError(t, err)

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)

			defer res.Body.Close()
			assert.Equal(t, http.StatusOK, res.StatusCode)
			b, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			var iRes v1beta1.InterceptorResponse
			err = json.Unmarshal(b, &iRes)
			require.NoError(t, err)

			assert.Equal(t, []interface{}{
				"folder/subfolder1/file",
				"folder/subfolder2/file",
				"folder/subfolder3/file",
			}, iRes.Extensions["filesChanged"])

			assert.Equal(t, codes.OK, iRes.Status.Code)
			assert.Equal(t, "", iRes.Status.Message, "")
			assert.True(t, iRes.Continue)
		})
	}
}

func getInterceptorRequest() (io.Reader, error) {
	event := github.PushEvent{
		Before: github.String("beforeRef"),
		After:  github.String("afterRef"),
		Repo: &github.PushEventRepository{
			FullName: github.String("test/platform"),
		}}
	e, err := json.Marshal(event)
	if err != nil {
		return bytes.NewReader(e), err
	}
	request := v1beta1.InterceptorRequest{
		Body: string(e),
	}

	b, err := json.Marshal(request)
	if err != nil {
		return bytes.NewReader(b), err
	}
	return bytes.NewReader(b), nil
}
