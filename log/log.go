package log

import (
	"fmt"
	stdlog "log"
	"os"
)

func init() {
	stdlog.SetFlags(stdlog.Ldate | stdlog.Ltime | stdlog.LUTC | stdlog.Lshortfile | stdlog.Lmsgprefix)
}

func Tracef(format string, v ...any) {
	stdlog.Output(2, fmt.Sprintf(format, v...))
}

func Debugf(format string, v ...any) {
	stdlog.Output(2, fmt.Sprintf(format, v...))
}

func Infof(format string, v ...any) {
	stdlog.Output(2, fmt.Sprintf(format, v...))
}

func Warningf(format string, v ...any) {
	stdlog.Output(2, fmt.Sprintf(format, v...))
}

func Errorf(format string, v ...any) {
	stdlog.Output(2, fmt.Sprintf(format, v...))
}

func Panicf(format string, v ...any) {
	s := fmt.Sprintf(format, v...)
	stdlog.Output(2, s)
	panic(s)
}

func Fatalf(format string, v ...any) {
	stdlog.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}
