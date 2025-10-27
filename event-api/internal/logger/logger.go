package logger

import (
	"go.uber.org/zap"
)

var Log *zap.Logger

// Init инициализирует логгер
func Init() {
	var err error
	Log, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
}

// Sync синхронизирует логгер (вызывать в defer)
func Sync() error {
	return Log.Sync()
}

// GetLogger возвращает экземпляр логгера
func GetLogger() *zap.Logger {
	if Log == nil {
		Init()
	}
	return Log
}
