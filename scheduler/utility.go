package scheduler

import (
	"fmt"
	"regexp"
	"strconv"
)

var host_parse = regexp.MustCompile(`^((?P<user>.+)@)?(?P<hostname>[^:]+)(:(?P<port>\d+))?`)

func qualifyHost(unqualified string, default_user string, default_port int) (qualified string) {
	m := FindNamedStringCaptures(host_parse, unqualified)

	host := m["hostname"]
	user := m["user"]
	port := m["port"]

	if user == "" {
		user = default_user
	}
	if port == "" {
		port = strconv.Itoa(default_port)
	}

	return fmt.Sprintf("%s@%s:%s", user, host, port)

}

func FindNamedStringCaptures(re *regexp.Regexp, x string) map[string]string {
	matches := make(map[string]string)
	parts := re.FindStringSubmatch(x)
	if parts == nil {
		return nil
	}

	for index, key := range host_parse.SubexpNames() {
		if key != "" {
			matches[key] = parts[index]
		}
	}
	return matches
}
