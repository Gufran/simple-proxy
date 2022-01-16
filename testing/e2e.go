package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

var (
	conns connTypes
)

const handshake = "Hello from %d\n"

type connType struct {
	kind  string
	ident string
	src   int
	dest  int
}

func (c *connType) String() string {
	var s string
	if c.kind != "" {
		s += c.kind
	}

	if c.ident != "" {
		s += ":" + c.ident + ":"
	}

	return fmt.Sprintf("%s%d:%d", s, c.src, c.dest)
}

type connTypes []*connType

func (c *connTypes) String() string {
	var s []string
	for _, i := range *c {
		s = append(s, i.String())
	}

	return strings.Join(s, ",")
}

func (c *connTypes) Set(val string) error {
	chunks := strings.Split(val, ":")
	if len(chunks) != 2 && len(chunks) != 4 {
		return fmt.Errorf("invalid format. value must be kind:host:src:dest or src:dest format")
	}

	v := new(connType)
	if len(chunks) == 4 {
		v.kind = chunks[0]
		v.ident = chunks[1]
		chunks = chunks[2:]
	}

	i, err := strconv.Atoi(chunks[0])
	if err != nil {
		return fmt.Errorf("source port must be valid integer. %s", err)
	}

	v.src = i

	i, err = strconv.Atoi(chunks[1])
	if err != nil {
		return fmt.Errorf("destination port must be valid integer. %s", err)
	}

	v.dest = i
	*c = append(*c, v)
	return nil
}

type result struct {
	logs []string
	err  error
}

func init() {
	flag.Var(&conns, "conn", "Connection details in kind:host:src:dest format where kind and host are optional")
	flag.Parse()
}

func main() {
	wg := &sync.WaitGroup{}
	results := make(chan result)
	ctx, cancel := context.WithCancel(context.Background())

	for _, c := range conns {
		wg.Add(1)
		go runTest(ctx, c, results, wg)
	}

	go func() {
		wg.Wait()
		close(results)
		cancel()
	}()

	isFailure := false
	for r := range results {
		if r.err != nil {
			isFailure = true
			log.Println(r.err)
		}

		for _, line := range r.logs {
			fmt.Println(line)
		}
	}

	if isFailure {
		log.Fatal("Test failed")
	}
}

func runTest(ctx context.Context, data *connType, results chan<- result, wg *sync.WaitGroup) {
	defer wg.Done()

	sc := make(chan struct{})

	go startListener(ctx, sc, data.src, data.dest)
	<-sc

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", data.dest))
	if err != nil {
		results <- result{err: err}
		return
	}

	var (
		cHello = fmt.Sprintf(handshake, data.src)
		sHello = fmt.Sprintf(handshake, data.dest)
	)

	r := bufio.NewReader(conn)
	_, err = conn.Write([]byte(cHello))
	if err != nil {
		results <- result{err: err}
		return
	}

	line, err := r.ReadString('\n')
	if err != nil {
		results <- result{err: err}
		return
	}

	if line == "error" {
		results <- result{
			logs: []string{
				fmt.Sprintf("reached the wrong server for port pair %d:%d. Expecting handshake message, got error.", data.src, data.dest),
			},
			err: fmt.Errorf("reached wrong server for port pair %d:%d", data.src, data.dest),
		}

		return
	}

	if line != sHello {
		results <- result{
			logs: []string{
				fmt.Sprintf("invalid handshake for port pair %d:%d. got %s", data.src, data.dest, line),
			},
			err: fmt.Errorf("invalid handshake for port pair %d:%d. got %s", data.src, data.dest, line),
		}

		return
	}
}

func startListener(ctx context.Context, sc chan struct{}, src, port int) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(fmt.Sprintf("failed to start listener on port %d. %s", port, err))
	}

	log.Printf("listener started on port %d, expecting connection from port %d", port, src)

	close(sc)

	var (
		cHello = fmt.Sprintf(handshake, src)
		sHello = fmt.Sprintf(handshake, port)
	)

	conn, err := listener.Accept()
	if err != nil {
		panic(fmt.Sprintf("failed to accept connection on port %d. %s", port, err))
	}
	defer conn.Close()

	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		panic(fmt.Sprintf("failed to read line %d:%d. %s", src, port, err))
	}

	if line != cHello {
		_, err = conn.Write([]byte("error\n"))
	} else {
		_, err = conn.Write([]byte(sHello))
	}

	if err != nil {
		panic(fmt.Sprintf("failed to write to client  %d:%d. %s", src, port, err))
	}
}
