package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

const HOST_INFO_PATH = "/var/run/scylla/endpoint"

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

func run(host, jobname string) {
	doPut(host, fmt.Sprintf("run/%s", jobname), "")
	fmt.Println("run requested")
}
func fail(host, jobname string) {
	doPut(host, fmt.Sprintf("fail/%s", jobname), "")
	fmt.Println("failed")
}

func update_pool(host, pool string) {
	hosts := make([]string, 0, 3)
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if scanner.Text() != "" {
			h := fmt.Sprintf("\"%s\"", scanner.Text())
			hosts = append(hosts, h)
		}
	}
	data := fmt.Sprintf("[%s]", strings.Join(hosts, ","))
	doPut(host, fmt.Sprintf("pool/%s", pool), data)

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
	case "run":
		if len(os.Args) <= 2 {
			err_exit("Syntax: sysctl run <jobname>")
		}
		run(host, os.Args[2])
	case "fail":
		if len(os.Args) <= 2 {
			err_exit("Syntax: sysctl fail <jobname>")
		}
		fail(host, os.Args[2])
	case "update_pool":
		if len(os.Args) <= 2 {
			err_exit("Syntax: sysctl update_pool <pool>")
		}
		update_pool(host, os.Args[2])
	case "test":
		test(host)
	default:
		err_exit("Invalid command: " + cmd)
	}
}
