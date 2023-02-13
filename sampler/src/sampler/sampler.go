package sampler

import (
	"log"
	"net"
	"scraper/libs"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Status struct {
	Availability bool          `json:"availability"`
	AccessTime   time.Duration `json:"access_time"`
}

type SampleData struct {
	Address string `json:"address"`
	Status  `json:",inline"`
}

type Sampler struct {
	address string
	data    SampleData
	mtx     sync.RWMutex
}

func (sp *Sampler) Update(timeout time.Duration) {
	tstart := time.Now()
	conn, err := net.DialTimeout("tcp", sp.address, timeout)
	dt := time.Since(tstart)
	if err == nil && conn != nil {
		conn.Close()
	}

	sp.mtx.Lock()
	defer sp.mtx.Unlock()
	if err != nil {
		sp.data.Availability = false
		return
	}
	sp.data.AccessTime = dt
	sp.data.Availability = true
}

func (sp *Sampler) CurrentData() SampleData {
	sp.mtx.RLock()
	defer sp.mtx.RUnlock()
	return sp.data
}

func (sp *Sampler) setAddress(value string) {
	sp.data.Address = value
	if !strings.Contains(value, ":") {
		sp.address = value + ":80" // try to check http port in default
	} else {
		sp.address = value
	}
}

func NewSampler(url string) *Sampler {
	return &Sampler{
		data: SampleData{
			Address: url,
		},
	}
}

type Group struct {
	data []Sampler
}

func (gs *Group) Update(timeout time.Duration) {
	for i := range gs.data {
		gs.data[i].Update(timeout)
	}
}

func NewGroupSamplers(addresses []string) *Group {
	gs := new(Group)
	n := len(addresses)
	gs.data = make([]Sampler, n)
	for i := 0; i < n; i++ {
		gs.data[i].setAddress(addresses[i])
	}
	return gs
}

type Manager struct {
	groups  []*Group
	lut     map[string]*Sampler
	period  time.Duration
	timeout time.Duration
	wg      sync.WaitGroup
	running atomic.Bool
	evt     chan bool
}

func (sm *Manager) update() {
	log.Println("manager update sites")
	var wg sync.WaitGroup
	for _, p := range sm.groups {
		wg.Add(1)
		go func(gs *Group) {
			gs.Update(sm.timeout)
			wg.Done()
		}(p)
	}
	wg.Wait()
}

func (sm *Manager) GetAll() []SampleData {
	v := make([]SampleData, len(sm.lut))
	cnt := 0
	for _, p := range sm.lut {
		v[cnt] = p.CurrentData()
		cnt++
	}
	return v
}

func (sm *Manager) GetOne(address string) *SampleData {
	p, ok := sm.lut[address]
	if !ok {
		return nil
	}
	x := p.CurrentData()
	return &x
}

func (sm *Manager) GetMany(addresses []string) []SampleData {
	v := libs.Unique(addresses)
	n := len(v)

	ls := make([]SampleData, 0, n)
	for i := 0; i < n; i++ {
		p, ok := sm.lut[v[i]]
		if !ok {
			continue
		}
		ls = append(ls, p.CurrentData())
	}
	return ls
}

func (sm *Manager) Stop() {
	if sm.running.Load() {
		log.Printf("try to stop sampler manager")
		sm.running.Store(false)
		sm.evt <- false
		sm.wg.Wait()
		log.Printf("stopped sampler manager")
	}
}

func (sm *Manager) Run() {
	sm.Stop()

	sm.running.Store(true)
	go func(sm *Manager) {
		sm.wg.Add(1)
		for sm.running.Load() {
			t := time.Now()
			sm.update()
			dt := time.Since(t)
			if dt < sm.period {
				select {
				case <-sm.evt:
					break
				case <-time.After(sm.period - dt):
				}
			}
		}
		sm.wg.Done()
	}(sm)
}

func NewSamplerManager(period, sampler_timeout time.Duration, addresses []string) *Manager {
	n := len(addresses)

	lut := make(map[string]*Sampler)
	for i := 0; i < n; i++ {
		lut[addresses[i]] = nil
	}

	n = len(lut)
	vurls := make([]string, n)
	cnt := 0
	for url := range lut {
		vurls[cnt] = url
		cnt++
	}

	group_size := (int)(period/(sampler_timeout+time.Millisecond*100) + 1)
	ngroups := (n + group_size - 1) / group_size
	groups := make([]*Group, ngroups)
	n = ngroups - 1
	for i := 0; i < ngroups; i++ {
		var v []string
		if i == n {
			v = vurls[i*group_size:]
		} else {
			v = vurls[i*group_size : (i+1)*group_size]
		}
		g := NewGroupSamplers(v)
		for cnt := range v {
			lut[v[cnt]] = &(g.data[cnt])
		}
		groups[i] = g
	}

	return &Manager{
		period:  period,
		timeout: sampler_timeout,
		lut:     lut,
		groups:  groups,
		evt:     make(chan bool),
	}
}
