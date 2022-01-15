package main

import (
	"fmt"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"io/ioutil"
	"net"
	"strings"
)

type Config struct {
	Listener []*Listener `hcl:"listen,block"`
}

type Listener struct {
	Interface string   `hcl:"interface,label"`
	ProxyTo   string   `hcl:"proxy_to,optional"`
	Route     []*Route `hcl:"route,block"`
}

type Route struct {
	Host string `hcl:"host,optional"`
	SNI  string `hcl:"sni,optional"`
	To   string `hcl:"to"`
}

func (p *Config) validate() error {
	if len(p.Listener) == 0 {
		return fmt.Errorf("at least one listen block must be provided")
	}

	var errs []string
	for ln, listen := range p.Listener {
		prefix := fmt.Sprintf("listener %d (%s):", ln+1, listen.Interface)

		_, _, err := net.SplitHostPort(listen.Interface)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s invalid listener address. Address must be in host:port or :port format. %s", prefix, err))
		}

		if listen.ProxyTo != "" && len(listen.Route) > 0 {
			errs = append(errs, fmt.Sprintf("%s either proxy_to or route can be set, not both", prefix))
		}

		if listen.ProxyTo == "" && len(listen.Route) == 0 {
			errs = append(errs, fmt.Sprintf("%s either proxy_to or route must be set", prefix))
		}

		if listen.ProxyTo != "" {
			_, _, err := net.SplitHostPort(listen.ProxyTo)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s invalid proxy_to address %q. Address must be host:port or :port format. %s", prefix, listen.ProxyTo, err))
			}
		}

		for rn, route := range listen.Route {
			rx := fmt.Sprintf("%s Route (%d):", prefix, rn+1)
			if route.To == "" {
				errs = append(errs, fmt.Sprintf("%s to attribute must be set", rx))
			}

			_, _, err := net.SplitHostPort(route.To)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s invalid destination address. Address must be host:port or :port format. %s", rx, err))
			}

			if route.SNI == "" && route.Host == "" {
				errs = append(errs, fmt.Sprintf("%s at least sni or host attribute must be set", rx))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "\n"))
	}

	return nil
}

func loadConfigFromFile(file string) (*Config, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return loadConfig(file, b)
}

func loadConfig(fileName string, content []byte) (*Config, error) {
	config := new(Config)
	err := hclsimple.Decode(fileName, content, nil, config)
	if err != nil {
		return nil, err
	}

	err = config.validate()
	if err != nil {
		return nil, err
	}

	return config, err
}
