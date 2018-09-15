package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"github.com/godbus/dbus"
	"log"
)

const (
	Black   = iota + 30
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

var logger *log.Logger

func format(msg string, color int) (string) {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color, msg)
}

func main() {

	logger = log.New(os.Stdout, "", log.Lshortfile)

	pactl, err := exec.LookPath("pactl")
	if err != nil {
		logger.Println(format("Not Supported!", Red))
		os.Exit(1)
	}

	cmd := exec.Command(pactl, "list", "sink-inputs")
	out, err := cmd.Output()
	if err != nil {
		logger.Println(format(err.Error(), Red))
		os.Exit(1)
	}
	sinks := strings.Split(string(out), "Sink Input")

	sinkNO := ""
	for _, sink := range sinks {
		if strings.Index(sink, "spotify") > 0 {
			index := strings.Index(sink, "#") + 1
			endIndex := strings.Index(sink, "\n")
			if index == -1 {
				logger.Println(format("Unknown Error", Red))
				os.Exit(1)
			}
			sinkNO = sink[index:endIndex]
			break
		}
	}
	conn, err := dbus.SessionBus()
	if err != nil {
		logger.Println(format(err.Error(), Red))
		os.Exit(1)
	}

	conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',path=/org/mpris/MediaPlayer2,interface='org.freedesktop.DBus.Properties'")

	c := make(chan *dbus.Signal, 10)
	conn.Signal(c)
	for change := range c {
		body := change.Body
		if len(body) > 1 {
			m, ok := body[1].(map[string]dbus.Variant)
			if !ok {
				logger.Println(format("Unexpected Result: "+fmt.Sprintf("%v", body), Yellow))
			} else {
				metaData := m["Metadata"]
				o, ok := metaData.Value().(map[string]dbus.Variant)
				if !ok {
					logger.Println(format("Unexpected Metadata: "+metaData.String(), Yellow))
				} else {
					logger.Println(format("Spotify: "+o["mpris:trackid"].String(), White))
					if sinkNO != "" {
						if strings.Contains(o["mpris:trackid"].String(), "spotify:ad") {
							cmd = exec.Command(pactl, "set-sink-input-mute", sinkNO, "1")
						} else {
							cmd = exec.Command(pactl, "set-sink-input-mute", sinkNO, "0")
						}
						_, err = cmd.Output()
						if err != nil {
							logger.Println(format(err.Error(), Yellow))
						}
					}
				}
			}
		}
	}
}
