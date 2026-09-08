package onectechcommon

import (
	"log"
	"os"
)

// Определяем уровни логирования (чем больше число, тем выше важность)
const (
	TraceLevel = iota // 0
	DebugLevel        // 1
	InfoLevel         // 2
	WarnLevel         // 3
	ErrorLevel        // 4
	CritLevel         // 5
)

var defLog Logger = NewConsoleLogger()

// SetLogger заменяет глобальный логгер (если передан не nil)
func SetLogger(l Logger) {
	if l != nil {
		defLog = l
	}
}

// GetLogger возвращает текущий глобальный логгер
func GetLogger() Logger {
	return defLog
}

// SetGlobalLevel устанавливает уровень для глобального логгера,
// если он поддерживает интерфейс LevelSetter.
func SetGlobalLevel(level int) {
	if ls, ok := defLog.(LevelSetter); ok {
		ls.SetLevel(level)
	}
}

// Logger — основной интерфейс логгирования
type Logger interface {
	Infof(format string, args ...any)
	Critf(format string, args ...any)
	Errf(format string, args ...any)
	Warningf(format string, args ...any)
	Debugf(format string, args ...any)
	Tracef(format string, args ...any)
}

// LevelSetter позволяет динамически менять уровень логирования
type LevelSetter interface {
	SetLevel(level int)
}

// logger — реализация Logger с поддержкой уровней
type logger struct {
	logger *log.Logger
	level  int
}

// NewConsoleLogger создаёт новый логгер с уровнем по умолчанию InfoLevel
func NewConsoleLogger() *logger {
	return &logger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
		level:  InfoLevel,
	}
}

// SetLevel устанавливает текущий уровень логирования
func (lg *logger) SetLevel(level int) {
	lg.level = level
}

// вспомогательная проверка: выводить ли сообщение
func (lg *logger) shouldLog(msgLevel int) bool {
	return msgLevel >= lg.level
}

// Реализация методов интерфейса с проверкой уровня

func (lg *logger) Infof(format string, args ...any) {
	if lg.shouldLog(InfoLevel) {
		lg.logger.Printf("[INFO] "+format, args...)
	}
}

func (lg *logger) Critf(format string, args ...any) {
	if lg.shouldLog(CritLevel) {
		lg.logger.Printf("[CRIT] "+format, args...)
	}
}

func (lg *logger) Errf(format string, args ...any) {
	if lg.shouldLog(ErrorLevel) {
		lg.logger.Printf("[ERROR] "+format, args...)
	}
}

func (lg *logger) Warningf(format string, args ...any) {
	if lg.shouldLog(WarnLevel) {
		lg.logger.Printf("[WARN] "+format, args...)
	}
}

func (lg *logger) Debugf(format string, args ...any) {
	if lg.shouldLog(DebugLevel) {
		lg.logger.Printf("[DEBUG] "+format, args...)
	}
}

func (lg *logger) Tracef(format string, args ...any) {
	if lg.shouldLog(TraceLevel) {
		lg.logger.Printf("[TRACE] "+format, args...)
	}
}
