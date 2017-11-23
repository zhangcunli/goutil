package util

import (
	"fmt"
	"net/http"
)

type HttpSrv struct {
	host string
	port int
}

func NewHttpSrv(host string, port int) *HttpSrv {
	srv := &HttpSrv{host: host, port: port}
	return srv
}

func (srv *HttpSrv) Route(pattern string, f http.HandlerFunc) {
	http.HandleFunc(pattern, f)
}

func (srv *HttpSrv) Run() {
	addr := fmt.Sprintf("%s:%d", srv.host, srv.port)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		panic(err)
	}
}
