package onectechcommon

import (
	"log"
	"os"
)

var defLog Logger = NewConsoleLogger()

func SetLogger(l Logger) {
	if l != nil {
		defLog = l
	}
}

func GetLogger() Logger {
	return defLog
}

type Logger interface {
	Infof(format string, args ...any)
	Critf(format string, args ...any)
	Errf(format string, args ...any)
	Warningf(format string, args ...any)
	Debugf(format string, args ...any)
	Tracef(format string, args ...any)
}

type logger struct {
	logger *log.Logger
}

func NewConsoleLogger() *logger {
	return &logger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

func (lg *logger) Infof(format string, args ...any) {
	lg.logger.Printf("[INFO] "+format, args...)
}

func (lg *logger) Critf(format string, args ...any) {
	lg.logger.Printf("[CRIT] "+format, args...)
}

func (lg *logger) Errf(format string, args ...any) {
	lg.logger.Printf("[ERROR] "+format, args...)
}

func (lg *logger) Warningf(format string, args ...any) {
	lg.logger.Printf("[WARN] "+format, args...)
}

func (lg *logger) Debugf(format string, args ...any) {
	lg.logger.Printf("[DEBUG] "+format, args...)
}

func (lg *logger) Tracef(format string, args ...any) {
	lg.logger.Printf("[TRACE] "+format, args...)
}
