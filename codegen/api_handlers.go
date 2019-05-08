package main
import "net/http"
import "strconv"
import "context"
import "encoding/json"
import "errors"

func (in *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/profile":
		if r.Method == "GET" {
			in.handlerProfile(w,r)
			return
		}
		if r.Method == "POST" {
			in.handlerProfile(w,r)
			return
		}
		apiError := ApiError{Err: errors.New("bad method"), HTTPStatus: http.StatusNotAcceptable}
		handleError(w, apiError)
		return
	case "/user/create":
		if r.Method == "POST" {
			in.handlerCreate(w,r)
			return
		}
		apiError := ApiError{Err: errors.New("bad method"), HTTPStatus: http.StatusNotAcceptable}
		handleError(w, apiError)
		return
	}
	apiError := ApiError{Err: errors.New("unknown method"), HTTPStatus: http.StatusNotFound}
	handleError(w, apiError)
}

func (in *MyApi) handlerProfile(w http.ResponseWriter, r *http.Request) {
	params := ProfileParams{}
	params.Login = r.FormValue("login")
	if params.Login == "" {
		apiError := ApiError{Err: errors.New("login must me not empty"), HTTPStatus: http.StatusBadRequest}
		handleError(w, apiError)
		return
	}
	result, err := in.Profile(context.Background(), params)
	if nil != err {
		handleError(w, err)
		return
	}
	handleResult(w, result)
}

func (in *MyApi) handlerCreate(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Auth")
	if token != "100500" {
		apiError := ApiError{Err: errors.New("unauthorized"), HTTPStatus: http.StatusForbidden}
		handleError(w, apiError)
		return
	}
	params := CreateParams{}
	params.Login = r.FormValue("login")
	params.Name = r.FormValue("full_name")
	params.Status = r.FormValue("status")
	Age, err := strconv.Atoi(r.FormValue("age"))
	if nil != err {
		apiError := ApiError{Err: errors.New("age must be int"), HTTPStatus: http.StatusBadRequest}
		handleError(w, apiError)
		return
	}
	params.Age = Age
	if params.Login == "" {
		apiError := ApiError{Err: errors.New("login must me not empty"), HTTPStatus: http.StatusBadRequest}
		handleError(w, apiError)
		return
	}
	if len(params.Login) < 10 {
		apiErr := ApiError{Err: errors.New("login len must be >= 10"), HTTPStatus: http.StatusBadRequest}
		handleError(w, apiErr)
		return
	}
	if params.Status == "" {
		params.Status = "user"
	}
	if params.Status != "user" &&
		params.Status != "moderator" &&
		params.Status != "admin" {
		apiErr := ApiError{Err: errors.New("status must be one of [user, moderator, admin]"), HTTPStatus: http.StatusBadRequest}
		handleError(w, apiErr)
		return
	}
	if params.Age < 0 {
		apiErr := ApiError{Err: errors.New("age must be >= 0"), HTTPStatus: http.StatusBadRequest}
		handleError(w, apiErr)
		return
	}
	if params.Age > 128 {
		apiErr := ApiError{Err: errors.New("age must be <= 128"), HTTPStatus: http.StatusBadRequest}
		handleError(w, apiErr)
		return
	}
	result, err := in.Create(context.Background(), params)
	if nil != err {
		handleError(w, err)
		return
	}
	handleResult(w, result)
}

func (in *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/create":
		if r.Method == "POST" {
			in.handlerCreate(w,r)
			return
		}
		apiError := ApiError{Err: errors.New("bad method"), HTTPStatus: http.StatusNotAcceptable}
		handleError(w, apiError)
		return
	}
	apiError := ApiError{Err: errors.New("unknown method"), HTTPStatus: http.StatusNotFound}
	handleError(w, apiError)
}

func (in *OtherApi) handlerCreate(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Auth")
	if token != "100500" {
		apiError := ApiError{Err: errors.New("unauthorized"), HTTPStatus: http.StatusForbidden}
		handleError(w, apiError)
		return
	}
	params := OtherCreateParams{}
	params.Username = r.FormValue("username")
	params.Name = r.FormValue("account_name")
	params.Class = r.FormValue("class")
	Level, err := strconv.Atoi(r.FormValue("level"))
	if nil != err {
		apiError := ApiError{Err: errors.New("level must be int"), HTTPStatus: http.StatusBadRequest}
		handleError(w, apiError)
		return
	}
	params.Level = Level
	if params.Username == "" {
		apiError := ApiError{Err: errors.New("username must me not empty"), HTTPStatus: http.StatusBadRequest}
		handleError(w, apiError)
		return
	}
	if len(params.Username) < 3 {
		apiErr := ApiError{Err: errors.New("username len must be >= 3"), HTTPStatus: http.StatusBadRequest}
		handleError(w, apiErr)
		return
	}
	if params.Class == "" {
		params.Class = "warrior"
	}
	if params.Class != "warrior" &&
		params.Class != "sorcerer" &&
		params.Class != "rouge" {
		apiErr := ApiError{Err: errors.New("class must be one of [warrior, sorcerer, rouge]"), HTTPStatus: http.StatusBadRequest}
		handleError(w, apiErr)
		return
	}
	if params.Level < 1 {
		apiErr := ApiError{Err: errors.New("level must be >= 1"), HTTPStatus: http.StatusBadRequest}
		handleError(w, apiErr)
		return
	}
	if params.Level > 50 {
		apiErr := ApiError{Err: errors.New("level must be <= 50"), HTTPStatus: http.StatusBadRequest}
		handleError(w, apiErr)
		return
	}
	result, err := in.Create(context.Background(), params)
	if nil != err {
		handleError(w, err)
		return
	}
	handleResult(w, result)
}

func handleError(w http.ResponseWriter, err error) {
	apiError, ok := err.(ApiError)
	if !ok {
		apiError = ApiError{Err: err, HTTPStatus: http.StatusInternalServerError}
	}
	var response = make(map[string]interface{})
	response["error"] = apiError.Err.Error()
	body, err := json.Marshal(response)
	if nil != err {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(apiError.HTTPStatus)
	w.Write(body)
}

func handleResult(w http.ResponseWriter, result interface{}) {
	var response = make(map[string]interface{})
	response["response"] = result
	response["error"] = ""
	body, err := json.Marshal(response)
	if nil != err {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(body)
}
