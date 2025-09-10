package log

import (
	"fmt"
	glog "log"
	"os"
	"path"
	"runtime"

	"github.com/kamrankamilli/gsvnc/pkg/config"
)

var infoLogger, warningLogger, errorLogger, debugLogger *glog.Logger

func init() {
	infoLogger = glog.New(os.Stderr, "INFO: ", glog.Ldate|glog.Ltime)
	warningLogger = glog.New(os.Stderr, "WARNING: ", glog.Ldate|glog.Ltime)
	errorLogger = glog.New(os.Stderr, "ERROR: ", glog.Ldate|glog.Ltime)
	debugLogger = glog.New(os.Stderr, "DEBUG: ", glog.Ldate|glog.Ltime)
}

func formatNormal(args ...interface{}) string {
	_, file, line, _ := runtime.Caller(2)
	out := fmt.Sprintf("%s:%d: ", path.Base(file), line)
	out += fmt.Sprint(args...)
	return out
}

func formatFormat(fstr string, args ...interface{}) string {
	_, file, line, _ := runtime.Caller(2)
	out := fmt.Sprintf("%s:%d: ", path.Base(file), line)
	out += fmt.Sprintf(fstr, args...)
	return out
}

func Info(args ...interface{}) { infoLogger.Println(formatNormal(args...)) }
func Infof(f string, args ...interface{}) {
	infoLogger.Println(formatFormat(f, args...))
}
func Warning(args ...interface{}) { warningLogger.Println(formatNormal(args...)) }
func Warningf(f string, args ...interface{}) {
	warningLogger.Println(formatFormat(f, args...))
}
func Error(args ...interface{}) { errorLogger.Println(formatNormal(args...)) }
func Errorf(f string, args ...interface{}) {
	errorLogger.Println(formatFormat(f, args...))
}
func Debug(args ...interface{}) {
	if config.Debug {
		debugLogger.Println(formatNormal(args...))
	}
}
func Debugf(f string, args ...interface{}) {
	if config.Debug {
		debugLogger.Println(formatFormat(f, args...))
	}
}
