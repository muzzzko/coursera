package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
)

type post struct {
	ID int `json:"id"`
	Param string `json:"param"`
}

func handlerTest(w http.ResponseWriter, r *http.Request) {
	s := ""
	for i := 1; i < 1000; i++ {
		p := &post{ID: 1, Param: "param"}
		s += fmt.Sprintf("%#v", p)
	}

	if _, err := w.Write([]byte(s)); nil != err {
		panic(err)
	}
}

func main() {
	http.HandleFunc("/", handlerTest)
	if err := http.ListenAndServe(":8080", nil); nil != err {
		panic(err)
	}
}
