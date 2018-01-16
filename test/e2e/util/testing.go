package util

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var mu sync.Mutex

type T struct {
	hasError      bool
	isNonBuffered bool
	t             *testing.T
	messages      []string
}

func NewT(t *testing.T) *T {
	t.Parallel()
	value, exists := os.LookupEnv("E2E_NON_BUFFERED_LOGS")
	return &T{
		hasError:      false,
		isNonBuffered: exists && (value == "true"),
		t:             t,
		messages:      make([]string, 0),
	}
}

func (t *T) Log(args ...interface{}) {
	t.log(decorate(fmt.Sprint(args...)))
}

func (t *T) Logf(format string, args ...interface{}) {
	t.log(decorate(fmt.Sprintf(format, args...)))
}

func (t *T) Error(args ...interface{}) {
	t.log(decorate(fmt.Sprint(args...)))
	t.t.Fail()
}

func (t *T) Errorf(format string, args ...interface{}) {
	t.log(decorate(fmt.Sprintf(format, args...)))
	t.t.Fail()
}

func (t *T) Fatal(args ...interface{}) {
	t.log(decorate(fmt.Sprint(args...)))
	t.Report()
	t.t.FailNow()
}

func (t *T) Fatalf(format string, args ...interface{}) {
	t.log(decorate(fmt.Sprintf(format, args...)))
	t.Report()
	t.t.FailNow()
}

func (t *T) Failed() bool {
	return t.t.Failed()
}

func (t *T) log(message string) {
	if t.isNonBuffered {
		fmt.Print(message)
	} else {
		t.messages = append(t.messages, message)
	}
}

func (t *T) Report() {
	mu.Lock()
	defer mu.Unlock()
	if len(t.messages) > 0 {
		fmt.Printf("--------------------------------------------------\n")
	}
	for _, message := range t.messages {
		fmt.Print(message)
	}
}

func decorate(s string) string {
	_, file, line, ok := runtime.Caller(2)
	if ok {
		if index := strings.LastIndex(file, "/"); index >= 0 {
			file = file[index+1:]
		} else if index = strings.LastIndex(file, "\\"); index >= 0 {
			file = file[index+1:]
		}
	} else {
		file = "???"
		line = 1
	}
	return fmt.Sprintf("%30s:%-3d %s\n", file, line, s)
}
