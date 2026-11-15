package redisconfig

import "testing"

func TestParseRedisURL(t *testing.T) {
	options, err := Parse("redis://:secret@localhost:6379/2")
	if err != nil {
		t.Fatalf("Parse() returned an error: %v", err)
	}
	if options.Address != "localhost:6379" || options.Password != "secret" || options.DB != 2 {
		t.Fatalf("Parse() = %+v, want URL fields to be converted", options)
	}
}

func TestParsePlainRedisAddress(t *testing.T) {
	options, err := Parse("localhost:6379")
	if err != nil {
		t.Fatalf("Parse() returned an error: %v", err)
	}
	if options.Network != "tcp" || options.Address != "localhost:6379" {
		t.Fatalf("Parse() = %+v, want a TCP host:port address", options)
	}
}
