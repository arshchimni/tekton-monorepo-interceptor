package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/go-github/v43/github"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) InterceptGitPayload() http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			s.respondErr(w, 500, codes.Internal, err)
			return
		}
		defer req.Body.Close()
		var event *github.PushEvent
		err = json.Unmarshal(b, &event)
		if err != nil {
			s.respondErr(w, 400, codes.InvalidArgument, err)
			return
		}
		filesChanged, err := s.diff.GetChangedFiles(req.Context(), event)

		if err != nil {
			s.respondErr(w, 500, codes.Internal, err)
			return
		}
		response := &v1beta1.InterceptorResponse{
			Extensions: map[string]interface{}{
				"filesChanged": filesChanged,
			},
			Continue: true,
			Status: v1beta1.Status{
				Code: codes.OK,
			},
		}

		b, err = json.Marshal(response)
		if err != nil {
			s.respondErr(w, 500, codes.Internal, err)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		_, err = w.Write(b)
		if err != nil {
			s.respondErr(w, 500, codes.Internal, err)
			return
		}
	}
}

func (s *Server) respondErr(w http.ResponseWriter, statusCode int, code codes.Code, err error) {
	s.logger.Error("Error While serving request", zap.Error(err))

	b, err := json.Marshal(v1beta1.InterceptorResponse{
		Continue: false,
		Status: v1beta1.Status{
			Code:    code,
			Message: err.Error(),
		},
	})
	if err != nil {
		s.logger.Error("Unable to marshall error response", zap.Error(err))
	}

	w.WriteHeader(statusCode)
	_, err = w.Write(b)
	if err != nil {
		s.logger.Error("Unable to write error response", zap.Error(err))
	}
}
