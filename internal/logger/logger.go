package logger

import (
	"github.com/sirupsen/logrus"
)

func NewLogger(level string) (*logrus.Logger, error) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return nil, err
	}
	logger.SetLevel(lvl)

	return logger, nil
}
