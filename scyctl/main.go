package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

const HOST_INFO_PATH = "/var/run/scylla.endpoint"

func err_exit(msg string) {
	fmt.Println(msg)
	os.Exit(-1)
}

func readHost() string {
	var data []byte
	var err error
	if data, err = ioutil.ReadFile(HOST_INFO_PATH); err != nil {
		err_exit("Unable to find host endpoint: " + err.Error())
	}
	return string(data)
}

func doPut(host string, resource string, data string) []byte {
	client := &http.Client{}
	url := fmt.Sprintf("http://%s/api/v1/%s", host, resource)
	request, err := http.NewRequest("PUT", url, strings.NewReader(data))
	if err != nil {
		err_exit(fmt.Sprintf("Request allocation for %s failed: %s", resource, err.Error()))
	}
	request.ContentLength = int64(len(data))
	response, err := client.Do(request)
	if err != nil {
		err_exit(fmt.Sprintf("Request %s failed: %s", resource, err.Error()))
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		err_exit(fmt.Sprintf("Request body read for %s failed: %s", resource, err.Error()))
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		err_exit(fmt.Sprintf("HTTP Failure sending command  %s failed: %d", resource, response.StatusCode))
	}
	return contents
}

func test(host string) {
	doPut(host, "test", "")
	fmt.Println("config ok")
}

func reload(host string) {
	doPut(host, "reload", "")
	fmt.Println("reloaded")
}

func main() {
	if len(os.Args) <= 1 {
		err_exit("Syntax: scyctl <reload|test|run|fail> [job]")
	}
	host := readHost()
	fmt.Printf("Using host: %s\n", host)
	cmd := os.Args[1]
	switch cmd {
	case "reload":
		reload(host)
	case "test":
		test(host)
	default:
		err_exit("Invalid command: " + cmd)
	}
}