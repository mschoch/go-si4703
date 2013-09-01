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
const AGC uint16 = 10
const BLNDADJ uint16 = 7

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
	d.registers[POWERCFG] = 0x0001
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

func (d *Device) Close() error {
	fmt.Printf("turning off chip")
	// read
	d.readRegisters()
	// enable the IC
	d.registers[POWERCFG] = 0x0000
	d.updateRegisters()

	// do some manual GPIO to initialize the device
	// err := rpio.Open()
	// if err != nil {
	// 	return err
	// }

	// pin23 := rpio.Pin(23)
	// pin23.Output()
	// pin23.Low()
	// rpio.Close()
	return nil
}

func (d *Device) DisableSoftMute() {
	d.readRegisters()
	d.registers[POWERCFG] = d.registers[POWERCFG] | (1 << SMUTE)
	d.updateRegisters()
}

func (d *Device) DisableMute() {
	d.readRegisters()
	d.registers[POWERCFG] = d.registers[POWERCFG] | (1 << DMUTE)
	d.updateRegisters()
}

func (d *Device) EnableMute() {
	d.readRegisters()
	d.registers[POWERCFG] = d.registers[POWERCFG] & 0xBFFF
	d.updateRegisters()
}

func (d *Device) readRegisters() {

	// with i2c we first write an address we want to read
	// however, this device interprets that address
	// as the first byte of the register at 0x2
	// so in order to use the ReadByteBlock method
	// without destroying our data, we have to write the
	// correct value back there

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, d.registers[0x2])
	bufbytes := buf.Bytes()

	var data []byte
	var err error
	if data, err = d.bus.ReadByteBlock(d.addr, bufbytes[0], 32); err != nil {
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

	//log.Printf("self: %v", d)
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

	d.readRegisters()
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
	// disable mute, it seems to not stick
	d.registers[POWERCFG] = d.registers[POWERCFG] | (1 << DMUTE)
	d.updateRegisters()
}

func (d *Device) SetChannel(channel uint16) {
	newChannel := channel * 10
	newChannel = newChannel - 8750
	newChannel = newChannel / 20

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

func (d *Device) Seek(dir byte) {
	d.readRegisters()
	if dir == 1 {
		d.registers[POWERCFG] = d.registers[POWERCFG] | (1 << SEEKUP)
	}
	d.registers[POWERCFG] = d.registers[POWERCFG] | (1 << SEEK)

	// start seek
	log.Printf("Attempting to seek")
	d.updateRegisters()

	// wait for seek to complete
	for {
		d.readRegisters()
		if d.registers[STATUSRSSI]&(1<<STC) != 0 {
			log.Printf("Seek Complete")
			break
		}
		log.Printf("Trying %s", d.printReadChannel(d.registers[READCHAN]))
		// reset the seek bits (they come back as zero whenever we read)
		// if dir == 1 {
		// 	d.registers[POWERCFG] = d.registers[POWERCFG] | (1 << SEEKUP)
		// }
		// d.registers[POWERCFG] = d.registers[POWERCFG] | (1 << SEEK)
		// d.updateRegisters()
	}

	// clear the seek bit
	d.registers[POWERCFG] = d.registers[POWERCFG] &^ (1 << SEEK)

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
	rv = rv + d.printSysConfig1(d.registers[SYSCONFIG1])
	rv = rv + d.printStatusRSSI(d.registers[STATUSRSSI])
	rv = rv + d.printReadChannel(d.registers[READCHAN])
	rv = rv + d.printRDS("A", d.registers[RDSA])
	rv = rv + d.printRDS("B", d.registers[RDSB])
	rv = rv + d.printRDS("C", d.registers[RDSC])
	rv = rv + d.printRDS("D", d.registers[RDSD])
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
	rv = rv + fmt.Sprintf("Stereo/Mono: %s\n", d.printStereoMonoConfig(byte(powercfg&0x3fff)>>13))
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

func (d *Device) printStereoMonoConfig(mono byte) string {
	switch mono {
	case 0x0:
		return "Stereo"
	default:
		return "Mono"
	}
}

func (d *Device) printStereoMonoActual(mono byte) string {
	switch mono {
	case 0x0:
		return "Mono"
	default:
		return "Stereo"
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
		freq := ((float64(channel) * 20) + 8750) / 100
		return fmt.Sprintf("%fMHz", freq)
	case 1:
		freq := (float64(spacing) * float64(channel)) + 76.0
		return fmt.Sprintf("%fMHz", freq)
	default:
		return "Unknown"
	}
}

func (d *Device) printDeemphasis(de byte) string {
	switch de {
	case 0:
		return fmt.Sprintf("75μs")
	case 1:
		return fmt.Sprintf("50μs")
	default:
		return "Unknown"
	}
}

func (d *Device) printSMBlend(blndadj byte) string {
	switch blndadj {
	case 0:
		return fmt.Sprintf("31–49 RSSI dBµV (default)")
	case 1:
		return fmt.Sprintf("37–55 RSSI dBµV (+6 dB)")
	case 2:
		return fmt.Sprintf("19–37 RSSI dBµV (–12 dB)")
	case 3:
		return fmt.Sprintf("25–43 RSSI dBµV (–6 dB)")
	default:
		return "Unknown"
	}
}

func (d *Device) printSysConfig1(sysconf uint16) string {
	rv := ""
	rv = rv + fmt.Sprintf("RDS Interrupt: %s\n", d.printEnabled(byte(sysconf>>RDSR)))
	rv = rv + fmt.Sprintf("Seek/Tune Complete Interrupt: %s\n", d.printEnabled(byte(sysconf>>STC&0x1)))
	rv = rv + fmt.Sprintf("RDS: %s\n", d.printEnabled(byte(sysconf>>RDS&0x1)))
	rv = rv + fmt.Sprintf("De-emphasis: %s\n", d.printDeemphasis(byte(sysconf>>DE&0x1)))
	rv = rv + fmt.Sprintf("AGC: %s\n", d.printEnabled(byte(sysconf>>AGC&0x1)))
	rv = rv + fmt.Sprintf("Stereo/Mono Blend Adjustment: %s\n", d.printSMBlend(byte(sysconf>>BLNDADJ&0x3)))
	return rv
}

func (d *Device) printRDSReady(rdsr byte) string {
	switch rdsr {
	case 0x0:
		return "No RDS group ready"
	default:
		return "New RDS group ready"
	}
}

func (d *Device) printComplete(com byte) string {
	switch com {
	case 0x0:
		return "Not complete"
	default:
		return "Complete"
	}
}

func (d *Device) printSeekFailBandLimit(sfbl byte) string {
	switch sfbl {
	case 0x0:
		return "Seek successful"
	default:
		return "Seek failure/Band limit reached"
	}
}

func (d *Device) printAFCRail(afcrl byte) string {
	switch afcrl {
	case 0x0:
		return "AFC not railed"
	default:
		return "AFC railed"
	}
}

func (d *Device) printSynchronized(rdss byte) string {
	switch rdss {
	case 0x0:
		return "RDS decoder not synchronized"
	default:
		return "RDS decoder synchronized"
	}
}

func (d *Device) printStatusRSSI(status uint16) string {
	fmt.Printf("raw status rssi: %d - %d\n", status>>8, status&0xFF)
	rv := ""
	rv = rv + fmt.Sprintf("RDS Ready: %s\n", d.printRDSReady(byte(status>>RDSR)))
	rv = rv + fmt.Sprintf("Seek/Tune Complete: %s\n", d.printComplete(byte(status>>STC&0x1)))
	rv = rv + fmt.Sprintf("Seek Fail/Band Limit: %s\n", d.printSeekFailBandLimit(byte(status>>SFBL&0x1)))
	rv = rv + fmt.Sprintf("AFC Rail: %s\n", d.printAFCRail(byte(status>>AFCRL&0x1)))
	rv = rv + fmt.Sprintf("RDS Synchronized: %s\n", d.printSynchronized(byte(status>>RDSS&0x1)))
	rv = rv + fmt.Sprintf("Stereo/Mono: %s\n", d.printStereoMonoActual(byte(status>>STEREO&0x1)))
	rv = rv + fmt.Sprintf("RSSI: %ddBµV\n", status&0x7F)
	return rv
}

func (d *Device) printReadChannel(readChannel uint16) string {
	rv := ""
	rv = rv + fmt.Sprintf("Read Channel: %s\n", d.printChannelNumber(readChannel&0x1FF))
	return rv
}

func (d *Device) printRDS(prefix string, rds uint16) string {
	rv := ""
	rv = rv + fmt.Sprintf("%s: %s%s\n", prefix, string(rds>>8), string(rds&0xFF))
	return rv
}
