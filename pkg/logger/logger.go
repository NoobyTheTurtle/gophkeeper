package logger

import (
	"log"
	"sync"

	"github.com/caarlos0/env/v6"
	"go.uber.org/zap"
)

type logger struct {
	IsProd  bool   `env:"IS_PROD" envDefault:"false"`
	LogPath string `env:"LOG_PATH" envDefault:"./.tmp/log.txt"`
}

var (
	zl   *zap.Logger
	once sync.Once
)

func NewLogger() *zap.Logger {
	once.Do(func() {
		var zlogger logger
		var zlConfig zap.Config

		err := env.Parse(&zlogger)
		if err != nil {
			log.Println(err.Error())
		}

		if zlogger.IsProd {
			zlConfig = zap.NewProductionConfig()
		} else {
			zlConfig = zap.NewDevelopmentConfig()
		}

		zlConfig.OutputPaths = []string{"stderr", zlogger.LogPath}
		zl, err = zlConfig.Build()
		if err != nil {
			log.Println(err.Error())
		}
	})

	return zl
}
