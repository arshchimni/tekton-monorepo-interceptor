package log

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNew(t *testing.T) {
	if _, err := New("INFO"); err != nil {
		t.Fatal(err)
	}

	if _, err := New("INFOOO"); err == nil {
		t.Fatal("expect to be failed")
	}

	if _, err := New(""); err == nil {
		t.Fatal("expect to be failed")
	}
}

func TestNewDiscard(t *testing.T) {
	logger := NewDiscard()
	logger.Info("test discard", zap.String("output", "discard"))
}

func TestLogLevel(t *testing.T) {
	cases := []struct {
		level   string
		success bool
		want    zapcore.Level
	}{
		{
			"info",
			true,
			zapcore.InfoLevel,
		},
		{
			"DEBUG",
			true,
			zapcore.DebugLevel,
		},

		{
			"FATAL", // not supported (debug or info is enough)
			false,
			zapcore.Level(0),
		},
	}

	for _, tc := range cases {
		got, err := logLevel(tc.level)
		if err != nil {
			if tc.success {
				t.Fatalf("expect to success: %s", err)
			}
			continue
		}

		if !tc.success {
			t.Fatal("expect to be failed")
		}

		if got != tc.want {
			t.Fatalf("got %v, want %v", got, tc.want)
		}
	}
}
