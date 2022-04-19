package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/go-github/v43/github"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

func (s *Server) InterceptGitPayload() http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {
		s.logger.Info("Request received by the interceptor")
		b, err := io.ReadAll(req.Body)
		defer req.Body.Close()
		if err != nil {
			s.respondErr(w, 500, codes.Internal, err)
			return
		}

		event, err := extractPushEvent(b)
		if err != nil {
			s.respondErr(w, 400, codes.InvalidArgument, err)
			return
		}

		filesChanged, err := s.diff.GetChangedFiles(req.Context(), &event)

		s.logger.Debug("The list of changed files in the commit", zap.Strings("files", filesChanged))
		if err != nil {
			s.respondErr(w, 500, codes.Internal, err)
			return
		}

		response, err := sendResponse(filesChanged)
		if err != nil {
			s.respondErr(w, 500, codes.Internal, err)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		_, err = w.Write(response)
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

func extractPushEvent(b []byte) (github.PushEvent, error) {

	var interceptorRequest v1beta1.InterceptorRequest

	var event github.PushEvent
	err := json.Unmarshal(b, &interceptorRequest)
	if err != nil {
		return event, err
	}
	//the webhook pyaload is wrapped in the body field of the interceptor response struct
	fmt.Println(interceptorRequest)
	err = json.Unmarshal([]byte(interceptorRequest.Body), &event)
	if err != nil {
		return event, err
	}

	return event, nil

}

func sendResponse(filesChanged []string) ([]byte, error) {

	//add the string array that contains the files changed in the given commit to the extensions of the
	//interceptor response
	response := &v1beta1.InterceptorResponse{
		Extensions: map[string]interface{}{
			"filesChanged": filesChanged,
		},
		Continue: true,
		Status: v1beta1.Status{
			Code: codes.OK,
		},
	}

	b, err := json.Marshal(response)
	if err != nil {

		return nil, err
	}
	return b, nil
}
