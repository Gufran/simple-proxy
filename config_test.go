package main

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"io/ioutil"
	"path"
	"path/filepath"
	"testing"
)

func TestValidateE2EConfig(t *testing.T) {
	info, err := ioutil.ReadDir("./testing")
	if err != nil {
		t.Error(err)
	}

	for _, p := range info {
		if filepath.Ext(p.Name()) != ".hcl" {
			continue
		}

		_, err := loadConfigFromFile(path.Join("testing", p.Name()))
		if err != nil {
			t.Fatalf("failed to load config from file %s. %s", p.Name(), err)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	cases := map[string]struct {
		config string
		expect *Config
		err    error
	}{
		"direct proxy": {
			config: `
listen ":8899" {
    proxy_to = "somedomain.com:80"
}`,
			expect: &Config{
				Listener: []*Listener{
					{
						Interface: ":8899",
						ProxyTo:   "somedomain.com:80",
					},
				},
			},
		},

		"multiple proxies": {
			config: `
listen ":1234" {
	proxy_to = "domain-one.com:8877"
}

listen ":2345" {
	proxy_to = "domain-two:7657"
}
`,
			expect: &Config{
				Listener: []*Listener{
					{
						Interface: ":1234",
						ProxyTo:   "domain-one.com:8877",
					},
					{
						Interface: ":2345",
						ProxyTo:   "domain-two:7657",
					},
				},
			},
		},

		"routed proxy": {
			config: `
listen ":1234" {
	route {
		host = "domain.com"
		to = "otherdomain.com:6543"
	}

	route {
		sni = "mydomain.com"
		to = "newdomain.com:7890"
	}
}

listen ":2345" {
	proxy_to = "domain-two:7657"
}
`,
			expect: &Config{
				Listener: []*Listener{
					{
						Interface: ":1234",
						Route: []*Route{
							{
								Host: "domain.com",
								To:   "otherdomain.com:6543",
							},
							{
								SNI: "mydomain.com",
								To:  "newdomain.com:7890",
							},
						},
					},
					{
						Interface: ":2345",
						ProxyTo:   "domain-two:7657",
					},
				},
			},
		},

		"invalid listener": {
			config: `
listen "2345" {
	proxy_to = "domain-two:7657"
}
`,
			err: fmt.Errorf("listener 1 (2345): invalid listener address. Address must be in host:port or :port format. address 2345: missing port in address"),
		},

		"invalid route": {
			config: `
listen ":2345" {
	proxy_to = "domain-two:7657"
	route {
		host = "somehost"
		to = "abcd:9876"
	}
}
`,
			err: fmt.Errorf("listener 1 (:2345): either proxy_to or route can be set, not both"),
		},

		"missing route": {
			config: `
listen ":2345" {}
`,
			err: fmt.Errorf("listener 1 (:2345): either proxy_to or route must be set"),
		},

		"malformed direct route": {
			config: `
listen ":2345" {
	proxy_to = "somehost"
}
`,
			err: fmt.Errorf("listener 1 (:2345): invalid proxy_to address \"somehost\". Address must be host:port or :port format. address somehost: missing port in address"),
		},

		"malformed route destination": {
			config: `
listen ":2345" {
	route {
		host = "somehost"
		to = "another"
	}
}
`,
			err: fmt.Errorf("listener 1 (:2345): Route (1): invalid destination address. Address must be host:port or :port format. address another: missing port in address"),
		},

		"missing route matcher": {
			config: `
listen ":2345" {
	route {
		to = "another:6543"
	}
}
`,
			err: fmt.Errorf("listener 1 (:2345): Route (1): at least sni or host attribute must be set"),
		},
	}

	for name, test := range cases {
		config, err := loadConfig(name+".hcl", []byte(test.config))
		if err == nil && test.err != nil {
			t.Errorf("%s: expected an error, got nil", name)
		}

		if err != nil && test.err == nil {
			t.Errorf("%s: unexpected error: %s", name, err)
		}

		if err != nil && !cmp.Equal(err.Error(), test.err.Error()) {
			t.Errorf("%s: unexpected error. %s", name, cmp.Diff(err.Error(), test.err.Error()))
		}

		if !cmp.Equal(config, test.expect) {
			t.Errorf("%s: unexpected result. %s", name, cmp.Diff(config, test.expect))
		}
	}
}
