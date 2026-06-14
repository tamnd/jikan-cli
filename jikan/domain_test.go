package jikan

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
)

// These tests are offline: they exercise the URI driver's pure string functions
// and the host wiring, which need no network. The client's HTTP behaviour is
// covered in jikan_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "jikan" {
		t.Errorf("Scheme = %q, want jikan", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s, ...]", info.Hosts, Host)
	}
	if info.Identity.Binary != "jikan" {
		t.Errorf("Identity.Binary = %q, want jikan", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct{ in, typ, id string }{
		{"52991", "anime", "52991"},
		{"attack on titan", "anime", "attack on titan"},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if err != nil || typ != tc.typ || id != tc.id {
			t.Errorf("Classify(%q) = (%q, %q, %v), want (%q, %q, nil)",
				tc.in, typ, id, err, tc.typ, tc.id)
		}
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("anime", "52991")
	want := "https://myanimelist.net/anime/52991"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("unknown", "52991")
	if err == nil {
		t.Error("Locate with unknown type should return error")
	}
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("Classify with empty input should return error")
	}
}

// TestHostWiring mounts the driver in a kit Host and checks the round trip:
// ResolveOn maps a bare id to a URI.
func TestHostWiring(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}

	got, err := h.ResolveOn("jikan", "52991")
	if err != nil || got.String() != "jikan://anime/52991" {
		t.Errorf("ResolveOn = (%q, %v), want jikan://anime/52991", got.String(), err)
	}
}
