package main

import (
	"fmt"

	"github.com/Th3Beetle/proxter"
)

func main() {
	requests := make(chan string)
	responses := make(chan string)
	control := make(chan bool)
	errors := make(chan error)
	proxter := proxter.New("", requests, responses, control, errors)
	go proxter.Start()
	for {
		request := <-requests
		fmt.Println("request: ")
		fmt.Println(request)
		response := <-responses
		fmt.Println("response: ")
		fmt.Println(response)
	}

}
