package proxter

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"

	utils "github.com/Th3Beetle/thutils"
)

type Proxter struct {
	localAddr        string
	defaultLocalAddr string
	Requests         chan string
	Responses        chan string
	Control          chan bool
	ErrorCh          chan error
}

const (
	httpPort = ":80"
	uriStart = 3
	uriPos   = 1
)

func New(localAddr string, requests chan string, responses chan string, control chan bool, errorCh chan error) *Proxter {
	return &Proxter{
		localAddr:        localAddr,
		defaultLocalAddr: "127.0.0.1:8000",
		Requests:         requests,
		Responses:        responses,
		Control:          control,
		ErrorCh:          errorCh,
	}
}

func (p *Proxter) Start() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	listener := p.getListener()

	for {
		lconn, err := listener.AcceptTCP()
		if err != nil {
			p.ErrorCh <- err
			continue
		}

		requestBytes, err := utils.ReadAll(lconn)
		if err != nil {
			p.ErrorCh <- err
			continue
		}
		request := string(requestBytes)

		raddr := p.getRemoteAddr(request)
		requestPrepared := prepareRequest(request)
		p.Requests <- requestPrepared

		control := <-p.Control

		if control {
			requestPrepared = <-p.Requests
		}

		preparedBytes := []byte(requestPrepared)

		rconn, err := net.DialTCP("tcp", nil, raddr)
		if err != nil {
			p.ErrorCh <- err
			continue
		}
		rconn.Write(preparedBytes)

		response, err := utils.ReadAll(rconn)
		if err != nil {
			p.ErrorCh <- err
			continue
		}
		rconn.Close()
		p.Responses <- string(response)
		lconn.Write(response)
		lconn.Close()
	}
}
func (p *Proxter) getListener() *net.TCPListener {
	var laddr *net.TCPAddr
	var err error
	if p.localAddr != "" {
		laddr, err = net.ResolveTCPAddr("tcp", p.localAddr)
	} else {
		laddr, err = net.ResolveTCPAddr("tcp", p.defaultLocalAddr)
	}
	if err != nil {
		fmt.Println("error resolving local address")
		os.Exit(1)
	}
	listener, err := net.ListenTCP("tcp", laddr)

	if err != nil {
		fmt.Println("Failed to get listener")
		os.Exit(1)
	}
	return listener
}

func (p *Proxter) getRemoteAddr(request string) *net.TCPAddr {
	queryString := strings.Split(strings.Fields(request)[1], "/")
	remoteAddr := queryString[2]
	if !strings.Contains(remoteAddr, ":") {
		remoteAddr = remoteAddr + httpPort
	}
	raddr, err := net.ResolveTCPAddr("tcp", remoteAddr)
	if err != nil {
		p.ErrorCh <- err
	}
	return raddr
}

func prepareRequest(request string) string {
	request = strings.Join(strings.Split(request, "Proxy-Connection: "), "Connection: ")
	uri := "/" + strings.Join(strings.Split(strings.Fields(request)[1], "/")[uriStart:], "/")
	requestSplitted := strings.Split(request, " ")
	requestSplitted[uriPos] = uri
	return strings.Join(requestSplitted, " ")
}
