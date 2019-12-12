package render

import (
	"encoding/json"
	"net/http"
)

type infoErr struct {
	Message string `json:"message"`
}

type responseErr struct {
	Err infoErr `json:"errors"`
}

// RespondJSON makes the response with payload as json format
func RespondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(response))
}

// RespondError makes the error response with payload as json format
func RespondError(w http.ResponseWriter, status int, err error) {
	e := responseErr{
		infoErr{
			Message: err.Error(),
		},
	}
	RespondJSON(w, status, e)
}
