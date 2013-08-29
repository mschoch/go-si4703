package main

import (
	"github.com/mschoch/si4703"
)

func main() {
	var i2cbus byte = 1
	d := new(si4703.Device)
	err := d.Init(i2cbus)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
}
