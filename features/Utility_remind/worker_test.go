package remind

import (
	"context"
	"github.com/bwmarrin/discordgo"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeTransport func(*http.Request) (*http.Response, error)

func (f fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestDeliveryAndBlockedDM(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		r, store := openRepo(t, filepath.Join(t.TempDir(), "bot.db"))
		ctx := context.Background()
		if err := r.add(ctx, job{ID: "1", Guild: "g", User: "u", Message: "private", Due: time.Now().Unix() - 1}); err != nil {
			t.Fatal(err)
		}
		session, _ := discordgo.New("Bot test")
		sent := 0
		session.Client = &http.Client{Transport: fakeTransport(func(req *http.Request) (*http.Response, error) {
			code := 200
			body := `{"id":"dm"}`
			if strings.HasSuffix(req.URL.Path, "/messages") {
				if !strings.Contains(req.URL.Path, "/channels/dm/") {
					t.Fatal("not a DM", req.URL.Path)
				}
				sent++
				body = `{"id":"message"}`
				if blocked {
					code = 403
					body = `{"code":50007,"message":"Cannot send messages to this user"}`
				}
			}
			return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
		})}
		w := &service{repo: r, guild: "g"}
		w.deliverBatch(ctx, session)
		if sent != 1 {
			t.Fatal("unexpected sends", sent)
		}
		list, err := r.list(ctx, "g", "u")
		if err != nil {
			t.Fatal(err)
		}
		if blocked {
			if len(list) != 1 || list[0].Status != "failed" {
				t.Fatal(list)
			}
		} else if len(list) != 0 {
			t.Fatal("delivered job not removed")
		}
		store.Close()
	}
}
func TestWorkerShutdown(t *testing.T) {
	r, s := openRepo(t, filepath.Join(t.TempDir(), "bot.db"))
	defer s.Close()
	ctx, cancel := context.WithCancel(context.Background())
	w := &service{repo: r, guild: "g"}
	if err := w.start(ctx, nil); err != nil {
		t.Fatal(err)
	}
	cancel()
	done := make(chan struct{})
	go func() { w.stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
}
