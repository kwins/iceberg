package icelog

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/kwins/iceberg/frame/util"
)

var hostname = util.GetHostname()

// 定义日志级别
const (
	DEBUG = iota
	INFO
	WARNING
	ERROR
	FATAL
)

// 定义日志级别
var (
	levleFlags        = [...]string{"DEBUG", " INFO", " WARN", "ERROR", "FATAL"}
	levelFlagsReverse = map[string]int{
		"DEBUG": DEBUG,
		"INFO":  INFO,
		"WARN":  WARNING,
		"ERROR": ERROR,
		"FATAL": FATAL,
	}
)

// default
var (
	defaultLogger *Logger
)

// Logger logger
type Logger struct {
	console  *log.Logger
	file     *FileWriter
	level    int
	layout   string
	showLine bool
}

// NewLogger new logger
func NewLogger() *Logger {
	defaultLogger = new(Logger)
	defaultLogger.console = log.New(os.Stdout, "", log.Ldate|log.Lmicroseconds)
	defaultLogger.level = DEBUG
	defaultLogger.layout = "2006-01-02 15:04:05.999"
	defaultLogger.showLine = true
	return defaultLogger
}

// FormatAndOutput format and out put
func (l *Logger) FormatAndOutput(calldepth, level int, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	var inf, code string
	if format != "" {
		inf = fmt.Sprintf(format, args...)
	} else {
		inf = fmt.Sprint(args...)
	}
	// source code, file and line num
	if l.showLine {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			code = path.Base(file) + ":" + strconv.Itoa(line)
		}
	}
	if l.file != nil {
		l.file.Write(l.formatLog(inf, code, level))
	} else {
		l.console.Output(calldepth, l.formatLog(inf, code, level))
	}
}

func (l *Logger) formatLog(info, code string, level int) string {
	if l.file != nil {
		return fmt.Sprintf("%s %s [%s] %s\n", hostname, time.Now().Format(l.layout), levleFlags[level], info)
	}
	return fmt.Sprintf("%s [%s] %s %s\n", hostname, levleFlags[level], info, code)
}

// SetLayout global SetLayout
func SetLayout(layout string) {
	defaultLogger.layout = layout
}

// SetLog SetLog
func SetLog(filename, level string) {
	if filename == "" {
		filename = os.Args[0] + ".log"
	}
	defaultLogger.file = NewFileWriter(filename)
	if level != "" {
		defaultLogger.level = levelFlagsReverse[strings.ToUpper(level)]
	}
}

// SetLevel SetLevel
func SetLevel(level string) {
	defaultLogger.level = levelFlagsReverse[strings.ToUpper(level)]
}

// Debug global debug
func Debug(args ...interface{}) {
	defaultLogger.FormatAndOutput(3, DEBUG, "", args...)
}

// Warn defalut wawrn
func Warn(args ...interface{}) {
	defaultLogger.FormatAndOutput(3, WARNING, "", args...)
}

// Info default info
func Info(args ...interface{}) {
	defaultLogger.FormatAndOutput(3, INFO, "", args...)
}

// Error default error
func Error(args ...interface{}) {
	defaultLogger.FormatAndOutput(3, ERROR, "", args...)
}

// Fatal default fatal
func Fatal(args ...interface{}) {
	defaultLogger.FormatAndOutput(3, FATAL, "", args...)
}

// Debugf global debug
func Debugf(fmt string, args ...interface{}) {
	defaultLogger.FormatAndOutput(3, DEBUG, fmt, args...)
}

// Warnf defalut wawrn
func Warnf(fmt string, args ...interface{}) {
	defaultLogger.FormatAndOutput(3, WARNING, fmt, args...)
}

// Infof default info
func Infof(fmt string, args ...interface{}) {
	defaultLogger.FormatAndOutput(3, INFO, fmt, args...)
}

// Errorf default error
func Errorf(fmt string, args ...interface{}) {
	defaultLogger.FormatAndOutput(3, ERROR, fmt, args...)
}

// Fatalf default fatal
func Fatalf(fmt string, args ...interface{}) {
	defaultLogger.FormatAndOutput(3, FATAL, fmt, args...)
}

// Default default log
func Default() *Logger {
	return defaultLogger
}

// Close Close
func Close() {
	if defaultLogger.file != nil {
		defaultLogger.file.Close()
	}
}
func init() {
	defaultLogger = NewLogger()
}
