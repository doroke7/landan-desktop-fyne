// Package helper prints what docker compose expects to read: JSON message lines and the metadata.
package helper

import (
	"encoding/json"
	"fmt"
)

// Metadata tells docker compose which options `up` and `down` accept (none).
const Metadata = `{
  "description": "在宿主機前景執行 landan 桌面程式,直到視窗關閉",
  "up": {"parameters": []},
  "down": {"parameters": []}
}`

type message struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func send(sType, sMessage string) {
	aLine, _ := json.Marshal(message{Type: sType, Message: sMessage})
	fmt.Println(string(aLine))
}

// Info shows a line in the output of docker compose.
func Info(sMessage string) { send("info", sMessage) }

// Error reports a failure; the command should then exit with a non-zero code.
func Error(sMessage string) { send("error", sMessage) }
