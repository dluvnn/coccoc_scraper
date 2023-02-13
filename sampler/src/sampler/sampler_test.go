package sampler_test

import (
	sampler "scraper/sampler/src/sampler"
	"testing"
	"time"
)

func TestSampleManager(t *testing.T) {
	m := sampler.NewSamplerManager(time.Second*5, time.Second*3, []string{"jd.com", "jd.com:80/", "live.com", "instagram.com"})
	m.Run()
	time.Sleep(time.Second * 11)
	m.Stop()

	t.Logf("%+v", m.GetAll())
}
