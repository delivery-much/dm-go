package render

import (
	"encoding/json"
	"net/http"
)

type infoErr struct {
	Message interface{} `json:"message"`
	Title string `json:"title,omitempty"`
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
func RespondError(w http.ResponseWriter, status int, err interface{}) {
	var m interface{}
	switch err.(type) {
	case error:
		m = err.(error).Error()
	case []error:
		var strErrs []string
		for _, v := range err.([]error) {
			strErrs = append(strErrs, v.Error())
		}
		m = strErrs
	}
	e := responseErr{
		infoErr{
			Message: m,
		},
	}
	RespondJSON(w, status, e)
}

// RespondErrorWithTitle makes the error response with title with payload as json format
func RespondErrorWithTitle(w http.ResponseWriter, status int, message, title string) {
	e := responseErr{
		infoErr{
			Message: message,
			Title: title,
		},
	}
	RespondJSON(w, status, e)
}