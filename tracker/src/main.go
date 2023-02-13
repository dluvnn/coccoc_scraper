package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"scraper/libs"
	"scraper/tracker/src/tracker"
	"strconv"
	"time"

	"github.com/alexflint/go-arg"
)

type appArgs struct {
	Port   int    `arg:"-p,--port" default:"8091" help:"the server listening port."`
	APIKey string `arg:"-k,--key" default:"" help:"the API key to access this service"`
	DBFile string `arg:"-d,--db" default:"db" help:"the database file"`
}

var (
	tk          *tracker.Tracker
	checkAPIKey libs.CheckAPIKeyFn
)

var (
	ErrInvalidTimeFrom = errors.New("invalid time from")
)

func one(w http.ResponseWriter, r *http.Request) {
	if !checkAPIKey(w, r) {
		return
	}

	q := r.URL.Query()

	user_id := q.Get("user")

	s := q.Get("from")
	from, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		libs.BadRequest(w, err)
		return
	}
	var to int64
	s = q.Get("to")
	if s == "" {
		to = time.Now().Unix()
	} else {
		to, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			libs.BadRequest(w, err)
			return
		}
	}

	n, err := tk.QueryOne(r.Context(), user_id, from, to)
	if err != nil {
		libs.InternalServerError(w, err)
		return
	}
	s = strconv.FormatInt(n, 10)
	w.Write([]byte(s))
}

func all(w http.ResponseWriter, r *http.Request) {
	if !checkAPIKey(w, r) {
		return
	}

	q := r.URL.Query()
	s := q.Get("from")
	from, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		libs.BadRequest(w, err)
		return
	}
	var to int64
	s = q.Get("to")
	if s == "" {
		to = time.Now().Unix()
	} else {
		to, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			libs.BadRequest(w, err)
			return
		}
	}

	log.Printf("admin query all from %d to %d", from, to)

	n, err := tk.QueryAll(r.Context(), from, to)
	s = strconv.FormatInt(n, 10)
	w.Write([]byte(s))
}

func update(w http.ResponseWriter, r *http.Request) {
	if !checkAPIKey(w, r) {
		return
	}
	var info map[string]int64
	err := libs.JSONParse(r, &info)
	if err != nil {
		libs.InternalServerError(w, err)
		return
	}
	err = tk.Update(r.Context(), info)
	if err != nil {
		libs.InternalServerError(w, err)
		return
	}
}

func startup() {
	var a appArgs
	arg.MustParse(&a)

	checkAPIKey = libs.MakeCheckAPIKey(a.APIKey)

	tk = new(tracker.Tracker)

	log.Print("open database folder: ", a.DBFile)
	err := tk.Init(a.DBFile)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/admin_query_one", one)
	http.HandleFunc("/admin_query_all", all)
	http.HandleFunc("/update", update)

	go libs.Serve(a.Port)
}

func exec() {
	libs.WaitCtrlC()
	err := tk.Close()
	if err != nil {
		log.Print(err)
	}
	fmt.Println("bye bye!")
}

func main() {
	startup()
	exec()
}
