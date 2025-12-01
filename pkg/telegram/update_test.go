package telegram

import (
	"testing"
)

func TestMessage_ParseCommand(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		wantIsCommand bool
		wantCommand   string
		wantArgs      string
	}{
		{
			name:          "simple command",
			text:          "/start",
			wantIsCommand: true,
			wantCommand:   "start",
			wantArgs:      "",
		},
		{
			name:          "command with args",
			text:          "/invest 1000",
			wantIsCommand: true,
			wantCommand:   "invest",
			wantArgs:      "1000",
		},
		{
			name:          "command with multiple args",
			text:          "/search BTC ETH SOL",
			wantIsCommand: true,
			wantCommand:   "search",
			wantArgs:      "BTC ETH SOL",
		},
		{
			name:          "command with @botname",
			text:          "/start@MyBot",
			wantIsCommand: true,
			wantCommand:   "start",
			wantArgs:      "",
		},
		{
			name:          "command with @botname and args",
			text:          "/invest@PrometheusBot 1000",
			wantIsCommand: true,
			wantCommand:   "invest",
			wantArgs:      "1000",
		},
		{
			name:          "regular text",
			text:          "Hello world",
			wantIsCommand: false,
			wantCommand:   "",
			wantArgs:      "",
		},
		{
			name:          "text starting with /",
			text:          "/",
			wantIsCommand: true,
			wantCommand:   "",
			wantArgs:      "",
		},
		{
			name:          "empty text",
			text:          "",
			wantIsCommand: false,
			wantCommand:   "",
			wantArgs:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{Text: tt.text}
			msg.ParseCommand()

			if msg.IsCommand != tt.wantIsCommand {
				t.Errorf("IsCommand = %v, want %v", msg.IsCommand, tt.wantIsCommand)
			}
			if msg.Command != tt.wantCommand {
				t.Errorf("Command = %q, want %q", msg.Command, tt.wantCommand)
			}
			if msg.Arguments != tt.wantArgs {
				t.Errorf("Arguments = %q, want %q", msg.Arguments, tt.wantArgs)
			}
		})
	}
}
