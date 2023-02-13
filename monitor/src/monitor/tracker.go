package monitor

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Tracker struct {
	service_address string
	proxy           *httputil.ReverseProxy
	period          time.Duration
	counter_man     CounterManager
	wg              sync.WaitGroup
	running         atomic.Bool
	evt             chan bool
}

func (tk *Tracker) error(err error) {
	log.Printf("Tracker Error: %v\n", err)
}

func (tk *Tracker) update() {
	m := tk.counter_man.ChangedInfo()
	n := len(m)
	log.Printf("Tracker::update %d item changed", n)
	if n == 0 {
		return
	}

	data, err := json.Marshal(m)
	if err != nil {
		tk.error(err)
		return
	}

	r, err := http.Post(tk.service_address+"/update", "application/json", bytes.NewBuffer(data))
	if err != nil {
		tk.error(err)
		return
	}
	if r.StatusCode != 200 {
		log.Printf("Tracker update returns code %d", r.StatusCode)
	}
}

func (tk *Tracker) Stop() {
	if tk.running.Load() {
		tk.running.Store(false)
		tk.evt <- false
		tk.wg.Wait()
	}
}

func (tk *Tracker) Run() {
	tk.Stop()

	log.Printf("Tracker::Run period = %v", tk.period)

	tk.running.Store(true)
	tk.wg.Add(1)
	go func(tk *Tracker) {
		for tk.running.Load() {
			tk.update()
			select {
			case <-tk.evt:
				break
			case <-time.After(tk.period):
			}
		}
		tk.wg.Done()
	}(tk)
}

func (tk *Tracker) Trigger(user_id string) {
	tk.counter_man.Update(user_id)
}

func (tk *Tracker) parseRespond(r *http.Response) (int64, error) {
	data, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		return 0, err
	}

	return strconv.ParseInt(string(data), 10, 64)
}

func (tk *Tracker) Forward(w http.ResponseWriter, r *http.Request) {
	tk.proxy.ServeHTTP(w, r)
}

func NewTracker(service_address string, period time.Duration) (*Tracker, error) {
	tk := new(Tracker)
	u, err := url.Parse(service_address)
	if err != nil {
		return nil, err
	}

	tk.service_address = service_address
	tk.proxy = httputil.NewSingleHostReverseProxy(u)
	tk.period = period
	tk.evt = make(chan bool)
	tk.counter_man.Init()
	return tk, nil
}
