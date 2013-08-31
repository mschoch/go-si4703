//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package si4703

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"bitbucket.org/gmcbay/i2c"
	"github.com/stianeikeland/go-rpio"
)

const I2C_ADDR = 0x10

// register names
const (
	DEVICEID uint16 = iota
	CHIPID
	POWERCFG
	CHANNEL
	SYSCONFIG1
	SYSCONFIG2
	UNUSED6
	UNUSED7
	UNUSED8
	UNUSED9
	STATUSRSSI
	READCHAN
	RDSA
	RDSB
	RDSC
	RDSD
)

// powercfg
const SMUTE uint16 = 15
const DMUTE uint16 = 14
const SKMODE uint16 = 10
const SEEKUP uint16 = 9
const SEEK uint16 = 8

// channel
const TUNE uint16 = 15

// sysconfig1
const RDS uint16 = 12
const DE uint16 = 11

// sysconfig2
const SPACE1 uint16 = 5
const SPACE0 uint16 = 4

// statusrssi
const RDSR uint16 = 15
const STC uint16 = 14
const SFBL uint16 = 13
const AFCRL uint16 = 12
const RDSS uint16 = 11
const STEREO uint16 = 8

type Device struct {
	bus       *i2c.I2CBus
	busNum    byte
	addr      byte
	registers []uint16
}

func (d *Device) Init(busNum byte) (err error) {
	return d.InitCustomAddr(I2C_ADDR, busNum)
}

func (d *Device) InitCustomAddr(addr, busNum byte) (err error) {
	// do some manual GPIO to initialize the device
	log.Printf("starting manual gpio")
	err = rpio.Open()
	if err != nil {
		return err
	}

	pin23 := rpio.Pin(23)
	pin23.Output()

	pin23.Low()
	time.Sleep(1 * time.Second)
	pin23.High()
	time.Sleep(1 * time.Second)

	rpio.Close()
	log.Printf("done manual gpio")

	if d.bus, err = i2c.Bus(busNum); err != nil {
		return
	}

	d.busNum = busNum
	d.addr = addr
	d.registers = make([]uint16, 16)

	// read
	d.readRegisters()
	// enable the oscillator
	d.registers[UNUSED7] = 0x8100
	// update
	d.updateRegisters()

	// wait for clock to settle
	time.Sleep(500 * time.Millisecond)

	// read
	d.readRegisters()
	// enable the IC
	d.registers[POWERCFG] = 0x4001
	// disable mute
	//d.registers[POWERCFG] = d.registers[POWERCFG] | (1 << SMUTE) | (1 << DMUTE)
	// enable the RDS
	d.registers[SYSCONFIG1] = d.registers[SYSCONFIG1] | (1 << RDS)
	d.registers[SYSCONFIG2] = d.registers[SYSCONFIG2] & 0xFFF0 // clear volume
	d.registers[SYSCONFIG2] = d.registers[SYSCONFIG2] | 0x0001 // set to lowest
	// update
	d.updateRegisters()

	// wait max powerup time
	time.Sleep(110 * time.Millisecond)

	return
}

func (d *Device) readRegisters() {
	var data []byte
	var err error
	if data, err = d.bus.ReadByteBlock(d.addr, 0, 32); err != nil {
		return
	}

	log.Printf("read bytes %v", data)

	counter := 0
	for x := 0x0A; ; x++ {
		if x == 0x10 {
			x = 0
		}
		p := bytes.NewBuffer(data[counter : counter+2])
		err = binary.Read(p, binary.BigEndian, &d.registers[x])
		if err != nil {
			log.Printf("error reading: %v", err)
			return
		}
		counter = counter + 2
		if x == 0x09 {
			break
		}
	}

	log.Printf("self: %v", d)
}

func (d *Device) updateRegisters() {
	p := new(bytes.Buffer)
	for x := 0x02; x < 0x08; x++ {
		binary.Write(p, binary.BigEndian, d.registers[x])
	}

	bytes := p.Bytes()
	log.Printf("output bytes is %v", bytes)

	err := d.bus.WriteByteBlock(d.addr, bytes[0], bytes[1:])
	if err != nil {
		log.Printf("error writing: %v")
	}
}

func (d *Device) SetVolume(volume uint16) {
	d.readRegisters()
	if volume < 0 {
		volume = 0
	}
	if volume > 15 {
		volume = 15
	}
	d.registers[SYSCONFIG2] = d.registers[SYSCONFIG2] & 0xFFF0
	d.registers[SYSCONFIG2] = d.registers[SYSCONFIG2] | volume
	d.updateRegisters()
}

func (d *Device) SetChannel(channel uint16) {
	newChannel := channel * 10
	newChannel = newChannel - 8750
	newChannel = newChannel / 10

	d.readRegisters()
	d.registers[CHANNEL] = d.registers[CHANNEL] & 0xFE00
	d.registers[CHANNEL] = d.registers[CHANNEL] | newChannel
	d.registers[CHANNEL] = d.registers[CHANNEL] | (1 << TUNE)

	log.Printf("Attempting to tune")
	d.updateRegisters()

	// wait for tuning to complete
	for {
		d.readRegisters()
		if d.registers[STATUSRSSI]&(1<<STC) != 0 {
			log.Printf("Tuning Complete")
			break
		}
	}

	// clear the tune bit
	d.registers[CHANNEL] = d.registers[CHANNEL] &^ (1 << TUNE)
	d.updateRegisters()

	// now wait for for STC to be cleared
	for {
		d.readRegisters()
		if d.registers[STATUSRSSI]&(1<<STC) == 0 {
			log.Printf("STC Cleared")
			break
		}
	}
}

func (d *Device) String() string {
	rv := "--------------------------------------------------------------------------------\n"
	rv = rv + d.printDeviceID(d.registers[DEVICEID])
	rv = rv + d.printChipID(d.registers[CHIPID])
	rv = rv + d.printPowerCfg(d.registers[POWERCFG])
	rv = rv + d.printChannel(d.registers[CHANNEL])
	rv = rv + "--------------------------------------------------------------------------------\n\n"
	return rv
}

func (d *Device) printDeviceID(deviceid uint16) string {
	rv := ""
	rv = rv + fmt.Sprintf("Part Number: %s\n", d.printPartNumber(byte(deviceid>>12)))
	rv = rv + fmt.Sprintf("Manufacturer: 0x%x\n", deviceid&0xFFF)
	return rv
}

func (d *Device) printPartNumber(num byte) string {
	switch num {
	case 0x01:
		return "Si4702/03"
	default:
		return "Unknown"
	}
}

func (d *Device) printChipID(chipid uint16) string {
	rv := ""
	rv = rv + fmt.Sprintf("Chip Version: %s\n", d.printChipVersion(byte(chipid>>10)))
	rv = rv + fmt.Sprintf("Device: %s\n", d.printDevice(byte((chipid&0x1FF)>>6)))
	rv = rv + fmt.Sprintf("Firmware Version %s\n", d.printFirmwareVersion(byte(chipid&0x1F)))

	return rv
}

func (d *Device) printChipVersion(rev byte) string {
	switch rev {
	case 0x04:
		return "Rev C"
	default:
		return "Unknown"
	}
}

func (d *Device) printDevice(dev byte) string {
	switch dev {
	case 0x0:
		return "Si4702 (off)"
	case 0x1:
		return "Si4702 (on)"
	case 0x8:
		return "Si4703 (off)"
	case 0x9:
		return "Si4703 (on)"
	default:
		return "Unknown"
	}
}

func (d *Device) printFirmwareVersion(rev byte) string {
	switch rev {
	case 0x0:
		return "Off"
	default:
		return fmt.Sprintf("%v", rev)
	}
}

func (d *Device) printPowerCfg(powercfg uint16) string {
	rv := ""
	rv = rv + fmt.Sprintf("Soft Mute: %s\n", d.printMute(byte(powercfg>>15)))
	rv = rv + fmt.Sprintf("Mute: %s\n", d.printMute(byte(powercfg&0x7fff)>>14))
	rv = rv + fmt.Sprintf("Stereo/Mono: %s\n", d.printStereoMono(byte(powercfg&0x3fff)>>13))
	rv = rv + fmt.Sprintf("RDS Mode: %s\n", d.printRDSMode(byte(powercfg&0xfff)>>11))
	rv = rv + fmt.Sprintf("Seek Mode: %s\n", d.printSeekMode(byte(powercfg&0x7ff)>>10))
	rv = rv + fmt.Sprintf("Seek Direction: %s\n", d.printSeekDirection(byte(powercfg&0x3ff)>>9))
	rv = rv + fmt.Sprintf("Seek: %s\n", d.printEnabled(byte(powercfg&0x1ff)>>8))
	rv = rv + fmt.Sprintf("Power-Up Disable: %s\n", d.printPower(byte(powercfg&0x3f)>>6))
	rv = rv + fmt.Sprintf("Power-Up Enable: %s\n", d.printPower(byte(powercfg&0x1)))
	return rv
}

func (d *Device) printMute(mute byte) string {
	switch mute {
	case 0x0:
		return "Enabled"
	default:
		return "Disabled"
	}
}

func (d *Device) printStereoMono(mono byte) string {
	switch mono {
	case 0x0:
		return "Stereo"
	default:
		return "Mono"
	}
}

func (d *Device) printRDSMode(rds byte) string {
	switch rds {
	case 0x0:
		return "Standard"
	default:
		return "Verbose"
	}
}

func (d *Device) printSeekMode(seek byte) string {
	switch seek {
	case 0x0:
		return "Wrap"
	default:
		return "Stop"
	}
}

func (d *Device) printSeekDirection(seek byte) string {
	switch seek {
	case 0x0:
		return "Down"
	default:
		return "Up"
	}
}

func (d *Device) printEnabled(seek byte) string {
	switch seek {
	case 0x0:
		return "Disabled"
	default:
		return "Enabled"
	}
}

func (d *Device) printPower(power byte) string {
	switch power {
	case 0x0:
		return "Default"
	default:
		return "On"
	}
}

func (d *Device) printChannel(tune uint16) string {
	rv := ""
	rv = rv + fmt.Sprintf("Tune: %s\n", d.printEnabled(byte(tune>>15)))
	rv = rv + fmt.Sprintf("Channel: %s\n", d.printChannelNumber(tune&0x1FF))

	return rv
}

func (d *Device) printChannelNumber(channel uint16) string {
	band := 0      // FIXME use actual band
	spacing := 200 // FIXME use actual spacing
	switch band {
	case 0:
		freq := (float64(spacing) * float64(channel)) + 87.5
		return fmt.Sprintf("%fMHz", freq)
	case 1:
		freq := (float64(spacing) * float64(channel)) + 76.0
		return fmt.Sprintf("%fMHz", freq)
	default:
		return "Unknown"
	}
}
