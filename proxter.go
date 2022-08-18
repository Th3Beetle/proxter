package proxter

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type Proxter struct {
	localAddr        string
	defaultLocalAddr string
}

var localAddr string = "127.0.0.1:8000"

const (
	headerDelim = "\r\n\r\n"
	httpPort    = ":80"
	uriStart    = 3
	uriPos      = 1
)

func New(localAddr string) *Proxter {
	return &Proxter{
		localAddr:        localAddr,
		defaultLocalAddr: "127.0.0.1:8000",
	}
}

func (p *Proxter) Start() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	listener := p.getListener()

	for {
		lconn, err := listener.AcceptTCP()
		if err != nil {
			fmt.Println("Failed to accept connection")
			continue
		}

		requestBytes := readAll(lconn)
		request := string(requestBytes)
		prettyPrint("Request string", request)

		raddr := getRemoteAddr(request)
		message := prepareRequest(request)
		prettyPrint("message to send", message)
		messageBytes := []byte(message)

		rconn, err := net.DialTCP("tcp", nil, raddr)
		if err != nil {
			fmt.Println("Failed to establish connection to raddr")
			continue
		}
		rconn.Write(messageBytes)

		response := readAll(rconn)
		rconn.Close()
		prettyPrint("response received", string(response))

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

func getRemoteAddr(request string) *net.TCPAddr {
	queryString := strings.Split(strings.Fields(request)[1], "/")
	remoteAddr := queryString[2]
	if !strings.Contains(remoteAddr, ":") {
		remoteAddr = remoteAddr + httpPort
	}
	raddr, err := net.ResolveTCPAddr("tcp", remoteAddr)
	if err != nil {
		fmt.Println("Failed to resolve remote addr")
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

func readAll(conn *net.TCPConn) []byte {
	reader := bufio.NewReader(conn)
	headers := readHeader(reader)
	headersString := string(headers)
	headers = []byte(headersString)
	clValue, err := extractContentLength(string(headers))
	if err != nil {
		fmt.Println("failed to extract content length")
	}
	body := make([]byte, clValue)
	io.ReadFull(reader, body)
	message := append(headers, body[:]...)
	return message
}

func readHeader(reader *bufio.Reader) []byte {
	var message []byte
	for {
		singleByte, err := reader.ReadByte()
		if err != nil {
			fmt.Println("cant read byte")
		}
		message = append(message, singleByte)
		if len(message) > 4 && bytes.Equal(message[len(message)-4:], []byte(headerDelim)) {
			break
		}
	}
	return message

}

func extractContentLength(headers string) (int, error) {
	contentLength := strings.Split(headers, "Content-Length: ")
	var clValue int
	var err error
	if len(contentLength) > 1 {
		valueString := strings.Split(contentLength[1], "\r\n")[0]
		clValue, err = strconv.Atoi(valueString)
		if err != nil {
			return 0, err
		}
	}
	return clValue, nil
}

func prettyPrint(name string, message string) {
	fmt.Println(name + ": ")
	fmt.Println(message)
	fmt.Println("---------------------")
}
