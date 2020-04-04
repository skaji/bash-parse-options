package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
)

const code = `
echo "option_url=${option_url[@]}"
echo "option_url_len=${#option_url[@]}"
echo "option_timeout=$option_timeout"
echo "option_retry=$option_retry"
echo "option_z=$option_z"
echo "argv=${argv[@]}"
`

func createBashFile() string {
	buf := new(bytes.Buffer)
	buf.WriteString("#!/bin/bash\n")
	writer = buf
	os.Args = []string{"main_test", "url|u=s@", "timeout|t=i", "retry|r", "z"}
	main()
	str := buf.String()
	str = strings.Replace(str, "# WRITE YOUR CODE\n", code, 1)
	file, err := ioutil.TempFile("", "test")
	if err != nil {
		panic(err)
	}
	file.WriteString(str)
	file.Close()
	return file.Name()
}

func runBash(file string, args ...string) (string, int) {
	args = append([]string{file}, args...)
	out, err := exec.Command("bash", args...).Output()
	exitCode := 0
	var exitError *exec.ExitError
	if err != nil {
		if errors.As(err, &exitError) {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return string(out), exitCode
}

func TestBasic(t *testing.T) {
	file := createBashFile()
	defer os.Remove(file)

	var (
		out         string
		code        int
		contains    []string
		notContains []string
	)

	runtest := func() {
		for _, c := range contains {
			if !strings.Contains(out, c) {
				t.Error("not contains " + c)
			}
		}
		for _, c := range notContains {
			if strings.Contains(out, c) {
				t.Error("contains " + c)
			}
		}
	}
	out, code = runBash(file, "-r", "-u=a", "-u=b", "-t=10")
	contains = []string{
		"option_url=a b",
		"option_retry=1",
		"option_timeout=10",
	}
	notContains = nil
	if code != 0 {
		t.Fail()
	}
	runtest()

	out, _ = runBash(file, "foo", "-r", "--url", "a", "-u", "b", "-t", "10")
	contains = []string{
		"option_url=a b",
		"option_retry=1",
		"option_timeout=10",
		"argv=foo",
	}
	notContains = nil
	runtest()

	_, code = runBash(file, "-t=a")
	if code == 0 {
		t.Fail()
	}

	out, code = runBash(file, "-zr")
	contains = []string{
		"option_retry=1",
		"option_z=1",
	}
	notContains = nil
	runtest()

	out, _ = runBash(file, "-u=a b c")
	contains = []string{
		"option_url=a b c",
		"option_url_len=1",
	}
	notContains = nil
	runtest()
	out, _ = runBash(file, "-u", "a b c")
	contains = []string{
		"option_url=a b c",
		"option_url_len=1",
	}
	notContains = nil
	runtest()
}
