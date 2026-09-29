package main

import "testing"

func TestGateDrainsAndRejects(t *testing.T) {
	var g gate
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	go g.run(func() { close(started); <-release; close(done) })
	<-started
	stopped := make(chan struct{})
	go func() { g.stop(); close(stopped) }()
	close(release)
	<-stopped
	select {
	case <-done:
	default:
		t.Fatal("shutdown did not drain handler")
	}
	g.run(func() { t.Fatal("accepted work after shutdown") })
}
