package monitor

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"scraper/sampler/src/sampler"
	"sync"
	"sync/atomic"
	"time"
)

type Sampler struct {
	service_address        string
	url_request_update_all string
	url_request_query      string
	period                 time.Duration
	cache                  SafeStringMap[sampler.Status]
	force                  SafeStringMap[bool]
	wg                     sync.WaitGroup
	running                atomic.Bool
	update_evt             chan bool
	force_evt              chan bool
	min                    SafeValue[*sampler.SampleData]
	max                    SafeValue[*sampler.SampleData]
}

func (sm *Sampler) Init() {
	sm.cache.Clear()
	sm.force.Clear()
	sm.url_request_update_all = sm.service_address + "/all"
	sm.url_request_query = sm.service_address + "/query"
	sm.update_evt = make(chan bool)
	sm.force_evt = make(chan bool)
}

func (sm *Sampler) error(err error) {
	log.Printf("Sampler Error: %v\n", err)
}

func (sm *Sampler) update_data(r *http.Response) {
	data, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		sm.error(err)
		return
	}

	var v []sampler.SampleData
	err = json.Unmarshal(data, &v)
	if err != nil {
		sm.error(err)
		return
	}

	n := len(v)
	if n > 0 {
		m := map[string]sampler.Status{}
		var pmin, pmax *sampler.SampleData
		var i int
		for i = 0; i < n; i++ {
			p := &v[i]
			m[p.Address] = p.Status
			if p.Availability {
				pmin = p
				pmax = p
				break
			}
		}

		for ; i < n; i++ {
			p := &v[i]
			m[p.Address] = p.Status
			if p.Availability {
				if p.AccessTime > pmax.AccessTime {
					pmax = p
				}
				if p.AccessTime < pmin.AccessTime {
					pmin = p
				}
			}
		}

		sm.cache.SetMany(m)
		if pmin != nil {
			x := new(sampler.SampleData)
			*x = *pmin
			sm.min.Set(x)
		}

		if pmax != nil {
			x := new(sampler.SampleData)
			*x = *pmax
			sm.max.Set(x)
		}
	}
}

func (sm *Sampler) update_all() {
	log.Printf("Sampler::update_all")
	r, err := http.Get(sm.url_request_update_all)
	if err != nil {
		sm.error(err)
		return
	}
	sm.update_data(r)
}

func (sm *Sampler) update_force() {
	m := sm.force.Clear()
	n := len(m)
	if n == 0 {
		return
	}
	vaddresses := make([]string, n)
	i := 0
	for s := range m {
		vaddresses[i] = s
		i++
	}

	data, err := json.Marshal(vaddresses)
	if err != nil {
		sm.error(err)
		return
	}

	r, err := http.Post(sm.url_request_query, "application/json", bytes.NewBuffer(data))
	if err != nil {
		sm.error(err)
		return
	}
	sm.update_data(r)
}

func (sm *Sampler) Min() *sampler.SampleData {
	return sm.min.Get()
}

func (sm *Sampler) Max() *sampler.SampleData {
	return sm.max.Get()
}

func (sm *Sampler) Stop() {
	if sm.running.Load() {
		sm.running.Store(false)
		sm.update_evt <- false
		sm.force_evt <- false
		sm.wg.Wait()
	}
}

func (sm *Sampler) Run() {
	sm.Stop()

	sm.running.Store(true)

	sm.wg.Add(1)
	go func(sm *Sampler) {
		for sm.running.Load() {
			t := time.Now()
			sm.update_all()
			dt := time.Since(t)
			if dt < sm.period {
				select {
				case <-sm.update_evt:
					break
				case <-time.After(sm.period - dt):
				}
			}
		}
		sm.wg.Done()
	}(sm)

	sm.wg.Add(1)
	go func(sm *Sampler) {
		for sm.running.Load() {
			sm.update_force()
			select {
			case <-sm.force_evt:
				break
			case <-time.After(time.Second):
			}
		}
		sm.wg.Done()
	}(sm)
}

func (sm *Sampler) ForceUpdate(target_address string) {
	sm.force.Set(target_address, true)
}

func (sm *Sampler) Query(targets []string) map[string]sampler.Status {
	return sm.cache.GetMany(targets)
}

func NewSampler(service_address string, period time.Duration) *Sampler {
	sm := new(Sampler)
	sm.period = period
	sm.service_address = service_address
	sm.Init()
	return sm
}
