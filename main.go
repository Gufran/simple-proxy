package main

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/urfave/cli/v2"
	"golang.org/x/sys/unix"
	"inet.af/tcpproxy"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

//go:embed version.txt
var Version string

type Flags struct {
	ConfigFile string
}

func app() *cli.App {
	flags := &Flags{}

	return &cli.App{
		Name:        "simple-proxy",
		Version:     Version,
		Description: "Run TCP proxies",
		Action:      action(flags),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Usage:       "Path to configuration file",
				EnvVars:     []string{"PROXY_CONFIG"},
				Required:    true,
				TakesFile:   true,
				Destination: &flags.ConfigFile,
			},
		},
	}
}

func main() {
	err := app().Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func action(flags *Flags) cli.ActionFunc {
	return func(parent *cli.Context) error {
		var (
			exit    = make(chan os.Signal, 2)
			refresh = make(chan []*tcpproxy.Proxy)
			errc    = make(chan error)
		)

		signal.Notify(exit, os.Interrupt)

		ctx, cancel := context.WithCancel(parent.Context)

		go func() {
			<-exit
			log.Println("attempting to exit gracefully. Send another SIGINT to exit immediately")
			cancel()

			<-exit
			log.Println("second signal received. Ditching...")
			os.Exit(1)
		}()

		go watchForReload(ctx, flags.ConfigFile, refresh, errc)

		return Handle(ctx, refresh, errc)
	}
}

func watchForReload(ctx context.Context, file string, refresh chan<- []*tcpproxy.Proxy, errc chan<- error) {
	reload := make(chan os.Signal, 2)
	signal.Notify(reload, syscall.SIGUSR1, syscall.SIGUSR2)

	config, err := loadConfigFromFile(file)
	if err != nil {
		errc <- err
		return
	}

	refresh <- makeProxies(ctx, config)

	for {
		select {
		case <-ctx.Done():
			return

		case <-reload:
			config, err = loadConfigFromFile(file)
			if err != nil {
				errc <- err
				continue
			}

			refresh <- makeProxies(ctx, config)
		}
	}
}

func Handle(ctx context.Context, refresh <-chan []*tcpproxy.Proxy, errc <-chan error) error {
	var proxies []*tcpproxy.Proxy

	for {
		select {
		case <-ctx.Done():
			log.Println("context is done. closing all proxy listeners and exiting...")
			closeAll(proxies)
			return nil

		case err := <-errc:
			log.Printf("failed to load configuration. %s", err)
			continue

		case newProxies := <-refresh:
			err := startAll(newProxies)
			if err != nil {
				log.Printf("failed to start listeners from new configuration. Keeping last known good configuration. %s", err)
				closeAll(newProxies)
				continue
			}

			log.Printf("all %d listeners from new configuration are online", len(newProxies))
			go closeAll(proxies)

			proxies = newProxies
		}
	}
}

func startAll(proxies []*tcpproxy.Proxy) error {
	var errs []string
	for _, proxy := range proxies {
		err := proxy.Start()
		if err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "\n"))
	}

	return nil
}

func closeAll(proxies []*tcpproxy.Proxy) {
	if proxies == nil {
		return
	}

	log.Printf("shutting down %d old listeners", len(proxies))

	for _, proxy := range proxies {
		// Close is always successful
		_ = proxy.Close()
	}

	for _, proxy := range proxies {
		err := proxy.Wait()
		if err == nil {
			continue
		}

		opErr, ok := err.(*net.OpError)
		if !ok {
			log.Printf("proxy listener closed with an error. %s", err)
			continue
		}

		if opErr.Err.Error() == "use of closed network connection" {
			continue
		}

		log.Printf("proxy listener closed with an error. %s", opErr)
	}
}

func makeProxies(ctx context.Context, config *Config) []*tcpproxy.Proxy {
	var proxies []*tcpproxy.Proxy

	for _, listener := range config.Listener {
		p := &tcpproxy.Proxy{
			ListenFunc: listenFunc(ctx),
		}

		if listener.ProxyTo != "" {
			p.AddRoute(listener.Interface, tcpproxy.To(listener.ProxyTo))
		} else {
			for _, route := range listener.Route {
				if route.Host != "" {
					p.AddHTTPHostRoute(listener.Interface, route.Host, tcpproxy.To(route.To))
				}

				if route.SNI != "" {
					p.AddSNIRoute(listener.Interface, route.SNI, tcpproxy.To(route.To))
				}
			}
		}

		proxies = append(proxies, p)
	}

	return proxies
}

func listenFunc(ctx context.Context) func(string, string) (net.Listener, error) {
	return func(network, address string) (net.Listener, error) {
		lc := net.ListenConfig{
			Control: func(network, address string, c syscall.RawConn) error {
				return c.Control(func(fd uintptr) {
					err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
					if err != nil {
						return
					}

					err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
					if err != nil {
						return
					}
				})
			},
		}

		return lc.Listen(ctx, network, address)
	}
}
