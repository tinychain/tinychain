package common

import (
	"github.com/op/go-logging"
	"sync"
)

var (
	logMgr *LoggerMgr
	once   sync.Once
)

func init() {
	once.Do(func() {
		if logMgr == nil {
			logMgr = NewLoggerMgr()
		}
	})
}

type LoggerMgr struct {
	loggers map[string]*Logger
	lock    sync.RWMutex
}

type Logger struct {
	namespace string
	logger    *logging.Logger
}

func NewLoggerMgr() *LoggerMgr {
	loggerMgr := &LoggerMgr{
		loggers: make(map[string]*Logger),
	}
	return loggerMgr
}

func (mgr *LoggerMgr) AddLogger(log *Logger) {
	mgr.lock.Lock()
	mgr.loggers[log.namespace] = log
	mgr.lock.Unlock()
}

func (mgr *LoggerMgr) GetLogger(namespace string) *logging.Logger {
	mgr.lock.RLock()
	defer mgr.lock.RUnlock()
	if mgr.loggers[namespace] == nil {
		mgr.loggers[namespace] = &Logger{
			namespace: namespace,
			logger:    logging.MustGetLogger(namespace),
		}
	}
	return mgr.loggers[namespace].logger
}

func GetLogger(namespace string) *logging.Logger {
	if logMgr == nil {
		logMgr = NewLoggerMgr()
	}
	return logMgr.GetLogger(namespace)
}
