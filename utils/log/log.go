package log

import (
	"context"

	"github.com/sirupsen/logrus"

	"prabogo/utils/activity"
)

const (
	MAX_LOG_ENTRY_SIZE = 8 * 1024
)

type Trail struct {
	Label   string
	Payload interface{}
}

func WithContext(ctx context.Context) *logrus.Entry {
	fields := activity.GetFields(ctx)

	return logrus.WithFields(fields)
}

func LogOrmer(obj interface{}, prefix string) {
	logrus.Debug(prefix, obj)
}

func LogTrail(obj interface{}, prefix string) {
	logrus.WithFields(logrus.Fields{
		"data": obj,
	}).Info(prefix)
}

func LogTrails(trails []Trail) {
	for _, trail := range trails {
		go LogTrail(trail.Payload, trail.Label)
	}
}
