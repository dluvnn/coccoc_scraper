package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"scraper/libs"
	"scraper/monitor/src/monitor"
	"time"

	"github.com/alexflint/go-arg"
)

type appArgs struct {
	Port           int    `arg:"-p,--port" default:"8090" help:"the server listening port."`
	AdminToken     string `arg:"-a,--admin" default:"" help:"the admin's token to use this service"`
	SamplerService string `arg:"-s,--sampler,required" help:"the address of the service Sampler, ex: http://localhost:8092"`
	SamplingPeriod int    `arg:"--period" default:"300" help:"the period in second to update data from Sampler"`
	TrackerService string `arg:"-t,--tracker,required" help:"the address of the service Tracker, ex: http://localhost:8091"`
	TrackerPeriod  int    `arg:"--tracker_period" default:"30" help:"the period in second to update service Tracker"`
}

var (
	ErrInvalidUserID       = errors.New("invalid user id")
	ErrIncorrectAdminToken = errors.New("incorrect admin token")
)

var (
	adminToken string
	tk         *monitor.Tracker
	sm         *monitor.Sampler
)

func startup() {
	var err error
	var a appArgs
	arg.MustParse(&a)

	adminToken = a.AdminToken
	tk, err = monitor.NewTracker(a.TrackerService, time.Duration(a.TrackerPeriod)*time.Second)
	if err != nil {
		panic(err)
	}
	sm = monitor.NewSampler(a.SamplerService, time.Duration(a.SamplingPeriod)*time.Second)

	http.HandleFunc("/force", force)
	http.HandleFunc("/check", check)
	http.HandleFunc("/min", min)
	http.HandleFunc("/max", max)
	http.HandleFunc("/admin_query_one", one)
	http.HandleFunc("/admin_query_all", all)

	go libs.Serve(a.Port)
}

func checkUserID(w http.ResponseWriter, r *http.Request) bool {
	user := r.Header.Get("user_id")
	if len(user) == 0 {
		libs.BadRequest(w, ErrInvalidUserID)
		return false
	}

	tk.Trigger(user)
	return true
}

func force(w http.ResponseWriter, r *http.Request) {
	if !checkUserID(w, r) {
		return
	}

	target := r.URL.Query().Get("target")
	log.Printf("force update target %s", target)
	if target == "" {
		return
	}

	sm.ForceUpdate(target)
}

func check(w http.ResponseWriter, r *http.Request) {
	if !checkUserID(w, r) {
		return
	}

	targets := r.URL.Query()["target"]
	log.Printf("check targets: %v", targets)
	if len(targets) == 0 {
		w.Write([]byte("{}"))
		return
	}

	libs.JSONReply(w, sm.Query(targets))
}

func min(w http.ResponseWriter, r *http.Request) {
	if !checkUserID(w, r) {
		return
	}

	libs.JSONReply(w, sm.Min())
}

func max(w http.ResponseWriter, r *http.Request) {
	if !checkUserID(w, r) {
		return
	}

	libs.JSONReply(w, sm.Max())
}

func one(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("admin_token") != adminToken {
		libs.BadRequest(w, ErrIncorrectAdminToken)
		return
	}

	tk.Forward(w, r)
}

func all(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("admin_token") != adminToken {
		libs.BadRequest(w, ErrIncorrectAdminToken)
		return
	}

	tk.Forward(w, r)
}

func exec() {
	sm.Run()
	tk.Run()

	fmt.Println("Press CTRL+C to exit.")
	libs.WaitCtrlC()

	sm.Stop()
	tk.Stop()
	fmt.Println("bye bye!")
}

func main() {
	startup()
	exec()
}
