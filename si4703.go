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

func (d *Device) String() string {
	rv := ""
	fmt.Sprintf("Part Number: %x\n", d.registers[DEVICEID]>>12)
	fmt.Sprintf("Manufacturer: %x\n", d.registers[DEVICEID]&0x8F)
	return rv
}

func (d *Device) Init(busNum byte) (err error) {
	return d.InitCustomAddr(I2C_ADDR, busNum)
}

func (d *Device) InitCustomAddr(addr, busNum byte) (err error) {
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
	d.registers[POWERCFG] = d.registers[POWERCFG] | (1 << SMUTE) | (1 << DMUTE)
	// enable the RDS
	d.registers[SYSCONFIG1] = d.registers[SYSCONFIG1] | (1 << RDS)
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

	for {
		d.readRegisters()
		if d.registers[STATUSRSSI]&(1<<STC) != 0 {
			log.Printf("Tuning Complete")
			break
		}
	}
}
