package config

import (
	"strings"
	"testing"
)

func TestDSNForcesUTCSession(t *testing.T) {
	d := DBConfig{User: "u", Password: "p", Host: "h", Port: 3306, Params: "parseTime=true&loc=UTC"}
	if got := d.DSN("x"); !strings.Contains(got, "time_zone=%27%2B00%3A00%27") {
		t.Fatalf("dsn %s", got)
	}
	d.Params = "parseTime=true&time_zone=%27Africa%2FNairobi%27"
	if got := d.DSN("x"); strings.Count(got, "time_zone=") != 1 {
		t.Fatalf("explicit time_zone overridden: %s", got)
	}
}
