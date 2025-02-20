package log

import (
	"fmt"
	"log"
	"os"

	"github.com/fatih/color"
)

var (
	green = color.New(color.FgGreen).SprintFunc()
	red   = color.New(color.FgRed).SprintFunc()
)

func Info(v ...interface{}) {
	m := fmt.Sprintln(v...)
	log.Printf("%s %s", green("[info]"), m)
}

func Fail(v ...interface{}) {
	m := fmt.Sprintln(v...)
	log.Printf("%s %s", red("[error]"), m)
}

func Failx(v ...interface{}) {
	Fail(v...)
	os.Exit(1)
}
