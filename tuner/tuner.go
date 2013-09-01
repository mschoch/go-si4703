package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mschoch/go-si4703"
)

func main() {
	var i2cbus byte = 1
	d := new(si4703.Device)

	err := d.Init(i2cbus)

	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	defer d.Close()

	reader := bufio.NewReader(os.Stdin)

OUTTER:
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.Replace(line, "\n", "", -1)

		command := strings.SplitN(line, " ", 2)
		switch command[0] {
		case "quit":
			break OUTTER
		case "volume":
			if len(command) > 1 {
				val, err := strconv.Atoi(command[1])
				if err != nil || val < 0 || val > 15 {
					fmt.Printf("Invalid volume level, must be (0-15)\n")
					continue
				}
				d.SetVolume(uint16(val))
			} else {
				fmt.Printf("Specify a volume level (0-15)\n")
			}
		case "mute":
			if len(command) > 1 {
				val := command[1]
				if val == "on" {
					d.EnableMute()
				} else if val == "off" {
					d.DisableMute()
				} else {
					fmt.Printf("Invalid setting, must be (on/off)\n")
				}
			} else {
				fmt.Printf("Specify setting (on/off)\n")
			}
		case "seek":
			if len(command) > 1 {
				val := command[1]
				if val == "up" {
					d.Seek(1)
				} else if val == "down" {
					d.Seek(0)
				} else {
					fmt.Printf("Invalid direction, must be (up/down)\n")
				}
			} else {
				fmt.Printf("Specify direction (up/down)\n")
			}
		case "tune":
			if len(command) > 1 {
				val, err := strconv.ParseFloat(command[1], 64)
				if err != nil {
					fmt.Printf("Invalid frequence\n")
				} else {
					val = val * 10
					freqint := uint16(val)
					d.SetChannel(freqint)
				}
			} else {
				fmt.Printf("Specify frequency in MHz\n")
			}
		case "status":
			fmt.Printf("%v", d)
		case "help":
			fmt.Printf("Valid commands are: quit, volume, mute, seek, tune, status\n")
		default:
			fmt.Printf("Unknown command: `%s`\n", command[0])
		}

	}
}
