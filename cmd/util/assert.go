package util

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func printf(format string, v ...interface{}) {
	if len(format) > 0 && format[0] == '\r' {
		fmt.Fprintf(os.Stderr, format, v...)
	} else {
		log.Printf(format, v...)
	}
}

func Verbosef(format string, v ...interface{}) {
	if FlagVerbose {
		printf(format, v...)
	}
}

func Warnf(format string, v ...interface{}) {
	printf(format, v...)
}

func Warning(err error, v ...interface{}) bool {
	if err != nil {
		if len(v) == 0 {
			Warnf("WARNING: %s.", err)
		} else {
			format := v[0].(string)
			v = v[1:]
			Warnf("%s: %s.", fmt.Sprintf(format, v...), err)
		}
		return true
	}
	return false
}

func Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

func Assert(err error, v ...interface{}) {
	if err != nil {
		if len(v) == 0 {
			Fatalf("ERROR: %s.", err)
		} else {
			format := v[0].(string)
			v = v[1:]
			Fatalf("%s: %s.", fmt.Sprintf(format, v...), err)
		}
	}
}

func AssertNArg(n int) {
	if flag.NArg() != n {
		flag.Usage()
	}
}

func AssertLeastNArg(n int) {
	if flag.NArg() < n {
		flag.Usage()
	}
}

func AssertIsDir(path string) {
	info, err := os.Stat(path)
	Assert(err, "Directory '%s' is not accessible", path)
	if !info.IsDir() {
		Fatalf("'%s' is not a directory.", path)
	}
}
