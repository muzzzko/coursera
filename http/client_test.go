package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"github.com/davecgh/go-spew/spew"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	Token = "secret"
)

var (
	OrderFields = map[string]bool{"Id": true, "Name": true, "Age": true}
)

type RespError struct {
	Error string `json:"error"`
}

type XMLUser struct {
	Id     int `xml:"id"`
	FirstName   string `xml:"first_name"`
	LastName   string `xml:"last_name"`
	Age    int `xml:"age"`
	About  string `xml:"about"`
	Gender string `xml:"gender"`
}

type XMLUsers struct {
	Users []XMLUser `xml:"row"`
}

type JSONUser struct {
	Id     int `json:"id"`
	Name   string `json:"name"`
	Age    int `json:"age"`
	About  string `json:"about"`
	Gender string `json:"gender"`
}

type Params struct {
	limit int
	offset int
	orderBy int
	query string
	orderField string
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	var params Params
	var err error
	spew.Dump()

	defer func (w http.ResponseWriter) {
		if err := recover(); nil != err {
			HandlerError(w, "", http.StatusInternalServerError)
		}
	}(w)

	token := r.Header.Get("AccessToken")
	if token != Token {
		HandlerError(w, "", http.StatusUnauthorized)
		return
	}

	params, err = validateParams(r)
	if nil != err {
		HandlerError(w, err.Error(), http.StatusBadRequest)
		return
	}

	users, err := getUsers(params)
	if nil != err {
		HandlerError(w, err.Error(), http.StatusBadRequest)
		return
	}

	body, err := json.Marshal(users)
	if nil != err {
		panic(err)
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(body)
	if nil != err {
		panic(err)
	}
}

func getUsers(params Params) ([]JSONUser, error) {
	var i int

	file, err := os.Open("dataset.xml")
	if nil != err {
		panic(err)
	}
	defer func () {
		err := file.Close()
		if nil != err {
			panic(err)
		}
	}()

	var XMLusers = XMLUsers{}
	content, err := ioutil.ReadAll(file)
	if nil != err {
		panic(err)
	}
	if err := xml.Unmarshal(content, &XMLusers); nil != err {
		panic(err)
	}

	var countUsers = len(XMLusers.Users)
	for i = 0; i < countUsers; {
		name := XMLusers.Users[i].FirstName + XMLusers.Users[i].LastName
		if strings.Contains(name, params.query) || strings.Contains(XMLusers.Users[i].About, params.query) {
			i++
		} else {
			XMLusers.Users = append(XMLusers.Users[:i], XMLusers.Users[i+1:]...)
		}
	}

	countUsers = len(XMLusers.Users)
	if countUsers < params.offset  {
		return []JSONUser{}, errors.New("big offset")
	}
	var users = make([]JSONUser, countUsers)
	for i = 0; i < countUsers; i++ {
		users[i].Id = XMLusers.Users[i].Id
		users[i].Name = XMLusers.Users[i].FirstName + XMLusers.Users[i].LastName
		users[i].Age = XMLusers.Users[i].Age
		users[i].About = XMLusers.Users[i].About
		users[i].Gender = XMLusers.Users[i].Gender
	}

	if params.orderBy != 0 {
		switch params.orderField {
		case "Id":
			sort.Slice(users, func(i, j int) bool {
				if params.orderBy == 1 {
					return users[i].Id < users[j].Id
				} else {
					return users[i].Id > users[j].Id
				}
			})
		case "Age":
			sort.Slice(users, func(i, j int) bool {
				if params.orderBy == 1 {
					return users[i].Age < users[j].Age
				} else {
					return users[i].Age > users[j].Age
				}
			})
		case "Name":
			sort.Slice(users, func(i, j int) bool {
				if params.orderBy == 1 {
					return users[i].Name < users[j].Name
				} else {
					return users[i].Name > users[j].Name
				}
			})
		}
	}

	if countUsers-params.offset < params.limit {
		params.limit = countUsers - params.offset
	}
	users = users[params.offset: params.offset + params.limit]

	return users, nil
}

func validateParams(r *http.Request) (Params, error) {
	params := Params{}
	var err error
	params.limit, err = strconv.Atoi(r.FormValue("limit"))
	if nil != err {
		return params, errors.New("bad limit")
	}
	params.offset, err = strconv.Atoi(r.FormValue("offset"))
	if nil != err {
		return params, errors.New("bad offset")
	}
	params.orderBy, err = strconv.Atoi(r.FormValue("order_by"))
	if nil != err {
		return params, errors.New("bad orderBy")
	}
	params.query = r.FormValue("query")
	params.orderField = r.FormValue("order_field")
	if _, ok := OrderFields[params.orderField]; !ok {
		return params, errors.New("ErrorBadOrderField")
	}

	return params, nil
}

func HandlerError(w http.ResponseWriter, message string, code int) {
	w.WriteHeader(code)
	if message != "" {
		body, err := json.Marshal(RespError{message})
		if nil != err {
			panic(err)
		}
		_, err = w.Write(body)
		if nil != err {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

func newServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(SearchServer))
}

func TestNegativeLimit(t *testing.T) {
	client := SearchClient{}

	searchRequest := SearchRequest{
		Limit: -1,
	}

	if _, err := client.FindUsers(searchRequest); nil == err || err.Error() != "limit must be > 0" {
		t.Error("failed limit error")
	}
}

func TestNegativeOffset(t *testing.T) {
	client := SearchClient{}

	searchRequest := SearchRequest{
		Limit:  1,
		Offset: -1,
		OrderBy: 1,
		OrderField: "Id",
		Query: "",
	}

	if _, err := client.FindUsers(searchRequest); nil == err || err.Error() != "offset must be > 0" {
		t.Error("failed offset error")
	}
}

func TestUnauthorizedRequest(t *testing.T) {
	ts := newServer()
	defer ts.Close()

	client := SearchClient{
		"wrongSecret",
		ts.URL,
	}

	searchRequest := SearchRequest{
		Limit:  1,
		Offset: 1,
		OrderBy: 1,
		OrderField: "Id",
		Query: "",
	}

	if _, err := client.FindUsers(searchRequest); nil == err || err.Error() != "Bad AccessToken" {
		t.Error("failed unauthorized error")
	}
}

func TestBadOrderByField(t *testing.T) {
	ts := newServer()
	defer ts.Close()

	client := SearchClient{
		Token,
		ts.URL,
	}

	searchRequest := SearchRequest{
		Limit:  1,
		Offset: 1,
		OrderBy: 1,
		OrderField: "wrong",
		Query: "",
	}

	if _, err := client.FindUsers(searchRequest); nil == err || err.Error() != "OrderFeld wrong invalid" {
		t.Error("failed bad order by field")
	}
}

func TestUnknownBadRequest(t *testing.T) {
	ts := newServer()
	defer ts.Close()

	client := SearchClient{
		Token,
		ts.URL,
	}

	searchRequest := SearchRequest{
		Limit:  1,
		Offset: 100,
		OrderBy: 1,
		OrderField: "Id",
		Query: "",
	}

	if _, err := client.FindUsers(searchRequest); nil == err || err.Error() != "unknown bad request error: big offset" {
		t.Error("failed unknown bad request")
	}
}

func TestPartData(t *testing.T) {
	ts := newServer()
	defer ts.Close()

	client := SearchClient{
		Token,
		ts.URL,
	}

	searchRequest := SearchRequest{
		Limit:  1,
		Offset: 1,
		OrderBy: 1,
		OrderField: "Id",
		Query: "",
	}

	resp, err := client.FindUsers(searchRequest)
	if nil != err {
		t.Error("failed get part data")
	}

	if len(resp.Users) != 1 || !resp.NextPage {
		t.Error("wrong return value")
	}
}

func TestLastData(t *testing.T) {
	ts := newServer()
	defer ts.Close()

	client := SearchClient{
		Token,
		ts.URL,
	}

	searchRequest := SearchRequest{
		Limit:  26,
		Offset: 30,
		OrderBy: 1,
		OrderField: "Id",
		Query: "",
	}

	resp, err := client.FindUsers(searchRequest)
	if nil != err {
		t.Error("failed get last data")
	}

	if len(resp.Users) != 5 || resp.NextPage {
		t.Error("wrong return value")
	}
}

func TestInternalServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := SearchClient{
		Token,
		ts.URL,
	}

	searchRequest := SearchRequest{
		Limit:  26,
		Offset: 30,
		OrderBy: 1,
		OrderField: "Id",
		Query: "",
	}

	_, err := client.FindUsers(searchRequest)
	if nil == err || err.Error() != "SearchServer fatal error" {
		t.Error("failed internal server error")
	}
}

func TestErrorJsonUnpackBadRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ =w.Write([]byte("err"))
	}))
	defer ts.Close()

	client := SearchClient{
		Token,
		ts.URL,
	}

	searchRequest := SearchRequest{
		Limit:  26,
		Offset: 30,
		OrderBy: 1,
		OrderField: "Id",
		Query: "",
	}

	_, err := client.FindUsers(searchRequest)
	if nil == err || err.Error() != "cant unpack error json: invalid character 'e' looking for beginning of value" {
		t.Error("failed unpack error")
	}
}

func TestErrorJsonUnpackOk(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ =w.Write([]byte("body"))
	}))
	defer ts.Close()

	client := SearchClient{
		Token,
		ts.URL,
	}

	searchRequest := SearchRequest{
		Limit:  26,
		Offset: 30,
		OrderBy: 1,
		OrderField: "Id",
		Query: "",
	}

	_, err := client.FindUsers(searchRequest)
	if nil == err || err.Error() != "cant unpack result json: invalid character 'b' looking for beginning of value" {
		t.Error("failed unpack error")
	}
}

func TestUnknownErrorWhileRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Location", "")
		_, _ = w.Write([]byte("body"))
	}))
	defer ts.Close()

	client := SearchClient{
		Token,
		"http://localhost:8888",
	}

	searchRequest := SearchRequest{
		Limit:  26,
		Offset: 30,
		OrderBy: 1,
		OrderField: "Id",
		Query: "",
	}

	_, err := client.FindUsers(searchRequest)
	if nil == err {
		t.Error("failed request error")
	}
}

func TestTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := SearchClient{
		Token,
		ts.URL,
	}

	searchRequest := SearchRequest{
		Limit:  26,
		Offset: 30,
		OrderBy: 1,
		OrderField: "Id",
		Query: "",
	}

	_, err := client.FindUsers(searchRequest)
	if nil == err || err.Error() != "timeout for limit=26&offset=30&order_by=1&order_field=Id&query=" {
		t.Error("failed request error")
	}
}