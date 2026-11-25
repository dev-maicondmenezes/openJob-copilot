package browser

import (
	"errors"
	"reflect"
	"testing"
)

func envOf(m map[string]string) Env { return func(k string) string { return m[k] } }

func TestIsHeadless(t *testing.T) {
	cases := []struct {
		name string
		goos string
		env  map[string]string
		want bool
	}{
		{"linux desktop x11", "linux", map[string]string{"DISPLAY": ":0"}, false},
		{"linux desktop wayland", "linux", map[string]string{"WAYLAND_DISPLAY": "wayland-0"}, false},
		{"linux no display", "linux", nil, true},
		{"linux ssh with forwarded display", "linux", map[string]string{"DISPLAY": "localhost:10", "SSH_CONNECTION": "1 2 3 4"}, true},
		{"linux ssh tty", "linux", map[string]string{"DISPLAY": ":0", "SSH_TTY": "/dev/pts/1"}, true},
		{"darwin gui", "darwin", nil, false},
		{"darwin ssh", "darwin", map[string]string{"SSH_CLIENT": "1 2 3"}, true},
		{"windows gui", "windows", nil, false},
		{"windows ssh", "windows", map[string]string{"SSH_CONNECTION": "x"}, true},
	}
	for _, c := range cases {
		if got := IsHeadless(c.goos, envOf(c.env)); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestOpenWith(t *testing.T) {
	url := "http://localhost:8080"
	cases := []struct {
		goos     string
		wantName string
		wantArgs []string
	}{
		{"linux", "xdg-open", []string{url}},
		{"darwin", "open", []string{url}},
		{"windows", "rundll32", []string{"url.dll,FileProtocolHandler", url}},
	}
	for _, c := range cases {
		var gotName string
		var gotArgs []string
		err := OpenWith(c.goos, url, func(name string, args ...string) error {
			gotName, gotArgs = name, args
			return nil
		})
		if err != nil || gotName != c.wantName || !reflect.DeepEqual(gotArgs, c.wantArgs) {
			t.Errorf("%s: %q %v err=%v", c.goos, gotName, gotArgs, err)
		}
	}
}

func TestOpenWithErrors(t *testing.T) {
	if err := OpenWith("plan9", "http://x", func(string, ...string) error { return nil }); err == nil {
		t.Error("unsupported OS must error")
	}
	boom := errors.New("no xdg-open")
	if err := OpenWith("linux", "http://x", func(string, ...string) error { return boom }); !errors.Is(err, boom) {
		t.Errorf("runner error not propagated: %v", err)
	}
}
