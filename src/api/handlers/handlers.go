package handlers

import (
	"fmt"
	"io"
	"moneyman/auth"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type NewVersionRequest struct {
	Notes  string `json:"notes"`
	Number string `json:"number"`
}

type UpdateVersionRequest struct {
	Notes  string `json:"notes"`
	Status string `json:"status"`
}

type ErrorMessage struct {
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

func ReadyHandler(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "OK\n")
}

func AuthenticateHandler(w http.ResponseWriter, req *http.Request) {
	defer func() {
		log.WithField("request_url", req.RequestURI).WithField("remote_addr", req.RemoteAddr).WithField("method", req.Method).Infof("AuthenticateHandler")
	}()

	switch req.Method {
	case "POST":
		username, password, ok := req.BasicAuth()
		log.WithField("user", username).WithField("pw", password).Debug("Auth")
		if !ok {
			http.Error(w, http.StatusText(400), 400)
			return
		}
		token, err := auth.GenerateToken(username, password)
		if err != nil {
			log.Errorf(err.Error())
			http.Error(w, http.StatusText(500), 500)
			return
		}
		if token == "" {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, fmt.Sprintf("{\"token\":\"%s\"}", token))
	default:
		http.Error(w, http.StatusText(400), 400)
		return
	}
}
