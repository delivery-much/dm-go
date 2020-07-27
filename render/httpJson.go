package render

import (
	"encoding/json"
	"net/http"
)

type infoErr struct {
	Message interface{} `json:"message"`
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
	var mess interface{}
	switch err.(type) {
	case error:
		mess = err.(error).Error()
	case []error:
		var strErrs []string
		for _, v := range err.([]error) {
			strErrs = append(strErrs, v.Error())
		}
		mess = strErrs
	}
	e := responseErr{
		infoErr{
			Message: mess,
		},
	}
	RespondJSON(w, status, e)
}
