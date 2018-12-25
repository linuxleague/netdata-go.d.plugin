package portcheck

import (
	"fmt"
	"sort"
	"time"

	"github.com/netdata/go.d.plugin/modules"
	"github.com/netdata/go.d.plugin/pkg/web"
)

func init() {
	creator := modules.Creator{
		UpdateEvery: 5,
		Create:      func() modules.Module { return New() },
	}

	modules.Register("portcheck", creator)
}

// New creates PortCheck with default values
func New() *PortCheck {
	var (
		defHTTPTimeout = time.Second
	)

	return &PortCheck{
		Timeout: web.Duration{Duration: defHTTPTimeout},

		doCh:    make(chan *port),
		doneCh:  make(chan struct{}),
		ports:   make([]*port, 0),
		metrics: make(map[string]int64),
	}

}

type state string

var (
	success state = "success"
	timeout state = "timeout"
	failed  state = "failed"
)

type port struct {
	number      int
	state       state
	inState     int
	updateEvery int
	latency     time.Duration
}

func (p *port) setState(s state) {
	changed := p.state != s

	if changed {
		p.inState = p.updateEvery
		p.state = s
	} else {
		p.inState += p.updateEvery
	}
}

func (p port) stateText() string {
	switch p.state {
	case success:
		return fmt.Sprintf("success_%d", p.number)
	case timeout:
		return fmt.Sprintf("timeout_%d", p.number)
	case failed:
		return fmt.Sprintf("failed_%d", p.number)
	}
	panic("unknown state")
}

func newPort(number, updateEvery int) *port {
	return &port{
		number:      number,
		updateEvery: updateEvery,
	}
}

// PortCheck portcheck module
type PortCheck struct {
	modules.Base

	Host        string       `yaml:"host" validate:"required"`
	Ports       []int        `yaml:"ports" validate:"required,gte=1"`
	Timeout     web.Duration `yaml:"timeout"`
	UpdateEvery int          `yaml:"update_every"`

	doCh   chan *port
	doneCh chan struct{}

	ports   []*port
	workers []*worker

	metrics map[string]int64
}

// Cleanup makes cleanup
func (tc *PortCheck) Cleanup() {
	for _, w := range tc.workers {
		w.stop()
	}
}

// Init makes initialization
func (tc *PortCheck) Init() bool {
	sort.Ints(tc.Ports)

	for _, p := range tc.Ports {
		tc.ports = append(tc.ports, newPort(p, tc.UpdateEvery))
		tc.workers = append(tc.workers, newWorker(tc.Host, tc.Timeout.Duration, tc.doCh, tc.doneCh))
	}

	tc.Debugf("using host %s", tc.Host)
	tc.Debugf("using ports %v", tc.Ports)
	tc.Debugf("using HTTP timeout: %s", tc.Timeout.Duration)

	return true
}

// Check makes check
func (PortCheck) Check() bool {
	return true
}

// Charts creates    charts
func (tc PortCheck) Charts() *Charts {
	var charts modules.Charts

	for _, p := range tc.Ports {
		charts.Add(chartsTemplate(p)...)
	}

	return &charts
}

// Collect collects metrics
func (tc *PortCheck) Collect() map[string]int64 {
	for _, p := range tc.ports {
		tc.doCh <- p
	}

	for i := 0; i < len(tc.ports); i++ {
		<-tc.doneCh
	}

	for _, p := range tc.ports {
		tc.metrics[fmt.Sprintf("success_%d", p.number)] = 0
		tc.metrics[fmt.Sprintf("failed_%d", p.number)] = 0
		tc.metrics[fmt.Sprintf("timeout_%d", p.number)] = 0

		tc.metrics[p.stateText()] = 1
		tc.metrics[fmt.Sprintf("instate_%d", p.number)] = int64(p.inState)
		tc.metrics[fmt.Sprintf("latency_%d", p.number)] = int64(p.latency)
	}

	return tc.metrics
}