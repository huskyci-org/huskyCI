package log_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	apiContext "github.com/huskyci-org/huskyCI/api/context"

	"github.com/huskyci-org/huskyCI/api/log"
)

func TestInitLog(t *testing.T) {
	apiContext.APIConfiguration = &apiContext.APIConfig{
		GraylogConfig: &apiContext.GraylogConfig{
			DevelopmentEnv: true,
			AppName:        "log_test",
			Tag:            "log_test",
		},
	}

	log.InitLog(true, "", "", "log_test", "log_test")

	if log.DefaultLogger() == nil {
		t.Error("expected default logger to be initialized, but it wasn't")
	}
}

func TestLog(t *testing.T) {
	testCases := []struct {
		name       string
		action     string
		info       string
		msgCode    int
		message    []interface{}
		logFunc    func(action, info string, msgCode int, message ...interface{})
		wantLevel  string
		wantMsgSub string
	}{
		{
			name:       "Info",
			action:     "action",
			info:       "info",
			msgCode:    11,
			message:    []interface{}{"got some info!"},
			logFunc:    log.Info,
			wantLevel:  "level=INFO",
			wantMsgSub: "Starting HuskyCI. got some info!",
		},
		{
			name:       "Warning",
			action:     "action",
			info:       "warn",
			msgCode:    11,
			message:    []interface{}{"got some warning!"},
			logFunc:    log.Warning,
			wantLevel:  "level=WARN",
			wantMsgSub: "Starting HuskyCI. got some warning!",
		},
		{
			name:       "Error",
			action:     "action",
			info:       "err",
			msgCode:    11,
			message:    []interface{}{"got some error!"},
			logFunc:    log.Error,
			wantLevel:  "level=ERROR",
			wantMsgSub: "Starting HuskyCI. got some error!",
		},
		{
			name:       "Info without variadic message",
			action:     "main",
			info:       "SERVER",
			msgCode:    11,
			message:    nil,
			logFunc:    log.Info,
			wantLevel:  "level=INFO",
			wantMsgSub: "Starting HuskyCI.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
			log.SetLogger(slog.New(h))

			if tc.message != nil {
				tc.logFunc(tc.action, tc.info, tc.msgCode, tc.message...)
			} else {
				tc.logFunc(tc.action, tc.info, tc.msgCode)
			}

			assertLogOutput(t, buf.String(), tc.wantLevel, tc.wantMsgSub, tc.action, tc.info)
		})
	}
}

func assertLogOutput(t *testing.T, out, wantLevel, wantMsgSub, action, info string) {
	t.Helper()
	checks := []struct {
		contains string
		desc     string
	}{
		{wantLevel, "level"},
		{wantMsgSub, "message"},
		{"action=" + action, "action"},
		{"info=" + info, "info"},
		{"msg_code=11", "msg_code"},
	}
	for _, c := range checks {
		if !strings.Contains(out, c.contains) {
			t.Errorf("log output should contain %s %q; got:\n%s", c.desc, c.contains, out)
		}
	}
}
