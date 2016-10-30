package main

import (
	"regexp"
	"strings"
)

type SshLogin struct {
	IPaddr string
	User   string
}

var (
	rx *regexp.Regexp
)

func init() {
	// This is the /var/log/secure version of the regexp
	// rx = regexp.MustCompile(`sshd\[\d+\]:\s+Failed password for (?:invalid\s+user\s+)?(.*) from (\d+\.\d+\.\d+\.\d+)\s+port`)

	// This is the systemd version of the regexp
	rx = regexp.MustCompile(`MESSAGE=Failed password for (?:invalid\s+user\s+)?(.*) from (\d+\.\d+\.\d+\.\d+)\s+port`)
}

func LineMatch(line string) (login SshLogin, found bool) {
	if m := rx.FindAllStringSubmatch(line, -1); m != nil {
		login.IPaddr = m[0][2]
		login.User = strings.ToLower(m[0][1])
		return login, true
	}

	return login, false
}
