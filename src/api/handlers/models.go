package handlers

import (
	"encoding/json"
	"moneyman/auth"
	"moneyman/models"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func ModelHandler[F models.Factory](w http.ResponseWriter, req *http.Request) {
	user, err := auth.ValidateBearerToken(req.Header.Get("Authorization"))
	if err != nil {
		log.WithField("request_url", req.RequestURI).WithField("remote_addr", req.RemoteAddr).WithField("method", req.Method).WithField("error", err.Error()).Error("ModelHandler")
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	defer func() {
		log.WithField("request_url", req.RequestURI).WithField("remote_addr", req.RemoteAddr).WithField("method", req.Method).WithField("user", user).Info("ModelHandler")
	}()

	var f F

	switch req.Method {
	case "GET":
		v := f.All()
		json.NewEncoder(w).Encode(&v)
	case "POST":
		r := f.Creator()
		dec := json.NewDecoder(req.Body)
		dec.DisallowUnknownFields()
		err := dec.Decode(&r)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		u := models.User{}
		u.Email = user.Email
		r.SetOwner(&u)
		m := f.New(r)

		err = m.Add()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			e := ErrorMessage{err.Error(), http.StatusBadRequest}
			json.NewEncoder(w).Encode(&e)
			return
		}
		json.NewEncoder(w).Encode(&m)
	}
}
