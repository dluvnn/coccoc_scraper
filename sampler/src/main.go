package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"scraper/libs"
	"scraper/sampler/src/sampler"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
)

type appArgs struct {
	Port      int    `arg:"-p,--port" default:"8092" help:"the server listening port."`
	SitesFile string `arg:"-f,--file" default:"sites.txt" help:"the file contains list of address"`
	Period    int    `arg:"--period" default:"300" help:"sampling period in second"`
	Timeout   int    `arg:"--timeout" default:"60" help:"sampling timeout in second"`
	APIKey    string `arg:"-k,--key" default:"" help:"the API key to access this service"`
}

var (
	sm          *sampler.Manager
	checkAPIKey libs.CheckAPIKeyFn
)

func startup() {
	var err error
	var a appArgs
	arg.MustParse(&a)

	data, err := os.ReadFile(a.SitesFile)
	if err != nil {
		panic(err)
	}

	v := strings.Split(string(data), "\n")
	n := len(v)
	var vaddress []string
	for i := 0; i < n; i++ {
		s := strings.TrimSpace(v[i])
		if len(s) > 0 {
			vaddress = append(vaddress, s)
		}
	}

	checkAPIKey = libs.MakeCheckAPIKey(a.APIKey)

	sm = sampler.NewSamplerManager(time.Second*time.Duration(a.Period), time.Second*time.Duration(a.Timeout), vaddress)

	http.HandleFunc("/query", query)
	http.HandleFunc("/one", one)
	http.HandleFunc("/all", all)

	go libs.Serve(a.Port)
}

func query(w http.ResponseWriter, r *http.Request) {
	if !checkAPIKey(w, r) {
		return
	}

	var address []string

	err := libs.JSONParse(r, &address)
	if err != nil {
		libs.InternalServerError(w, err)
		return
	}

	data := sm.GetMany(address)
	err = libs.JSONReply(w, &data)
	if err != nil {
		log.Print(err)
	}
}

func one(w http.ResponseWriter, r *http.Request) {
	if !checkAPIKey(w, r) {
		return
	}

	address := r.URL.Query().Get("address")
	p := sm.GetOne(address)
	err := libs.JSONReply(w, p)
	if err != nil {
		log.Print(err)
	}
}

func all(w http.ResponseWriter, r *http.Request) {
	if !checkAPIKey(w, r) {
		return
	}

	err := libs.JSONReply(w, sm.GetAll())
	if err != nil {
		log.Print(err)
	}
}

func exec() {
	sm.Run()

	libs.WaitCtrlC()

	sm.Stop()
	fmt.Println("bye bye!")
}

func main() {
	startup()
	exec()
}
