package web

import (
	"fmt"
	"github.com/ZhongAnTech/BlockDB/brefactor/core_interface"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"sync"
)

type HttpListenerConfig struct {
	Port             int
	MaxContentLength int64
}

type HttpListener struct {
	JsonCommandParser       core_interface.JsonCommandParser
	BlockDBCommandProcessor core_interface.BlockDBCommandProcessor
	Config                  HttpListenerConfig

	wg      sync.WaitGroup
	stopped bool
	router  *mux.Router
}

func (l *HttpListener) Name() string {
	return "HttpListener"
}

func (l *HttpListener) Setup() {
	if l.Config.MaxContentLength == 0 {
		l.Config.MaxContentLength = 1e7
	}

	l.router = mux.NewRouter()
	l.router.Methods("POST").Path("/audit").HandlerFunc(l.Handle)
	//l.router.Methods("GET", "POST").Path("/query").HandlerFunc(l.Query)
	//l.router.Methods("GET", "POST").Path("/queryGrammar").HandlerFunc(l.QueryGrammar)
	l.router.Methods("GET", "POST").Path("/health").HandlerFunc(l.Health)
}

func (l *HttpListener) Start() {
	go l.doListen()
	logrus.Info("HttpListener started")
}

func (l *HttpListener) Stop() {
	l.stopped = true
	logrus.Info("HttpListener stopped")
}

func (l *HttpListener) Handle(rw http.ResponseWriter, req *http.Request) {
	if req.ContentLength > l.Config.MaxContentLength {
		http.Error(rw, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		return
	}

	data, err := ioutil.ReadAll(req.Body)
	if err != nil || len(data) == 0 {
		http.Error(rw, "miss content", http.StatusBadRequest)
		return
	}
	logrus.Tracef("get audit request data: %s", string(data))

	command, err := l.JsonCommandParser.FromJson(string(data))

	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := l.BlockDBCommandProcessor.Process(command)
	if err != nil {
		logrus.WithError(err).Warn("failed to process command")
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	logrus.WithField("result", result).Info("process result")
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("{}")) // TODO: write result of BlockDBCommandProcessor.Process

}

func (l *HttpListener) Health(rw http.ResponseWriter, req *http.Request) {
	// TODO: do real health check
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("ok"))
}

func (l *HttpListener) doListen() {
	logrus.WithField("port", l.Config.Port).Info("RPC server listening")
	logrus.Fatal(http.ListenAndServe(":"+fmt.Sprintf("%d", l.Config.Port), l.router))
}
