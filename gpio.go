/*
 * talkkonnect headless mumble client/gateway with lcd screen and channel control
 * Copyright (C) 2018-2019, Suvir Kumar <suvir@talkkonnect.com>
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 *
 * Software distributed under the License is distributed on an "AS IS" basis,
 * WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
 * for the specific language governing rights and limitations under the
 * License.
 *
 * talkkonnect is the based on talkiepi and barnard by Daniel Chote and Tim Cooper
 *
 * The Initial Developer of the Original Code is
 * Suvir Kumar <suvir@talkkonnect.com>
 * Portions created by the Initial Developer are Copyright (C) Suvir Kumar. All Rights Reserved.
 *
 * Rotary Encoder Alogrithm Inpired By https://www.brainy-bits.com/post/best-code-to-use-with-a-ky-040-rotary-encoder-let-s-find-out
 *
 * Contributor(s):
 *
 * Suvir Kumar <suvir@talkkonnect.com>
 *
 * My Blog is at www.talkkonnect.com
 * The source code is hosted at github.com/talkkonnect
 *
 * gpio.go talkkonnects function to connect to SBC GPIO


 **Edits Made By Andrew Roberts with assistance from Google Gemini
 */

package talkkonnect

import (
	"log"
	"strconv"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
	"github.com/talkkonnect/go-mcp23017"
	"github.com/talkkonnect/max7219"
)

var pioPins = make(map[string]gpio.PinIO)

var (
	Max7219Dev                    max7219.Device
	EnabledRotaryEncoderFunctions []string
	CurrentRotaryEncoderFunction  int
)

// Legacy Hardware Globals
var (
	TxButtonUsed, TxToggleUsed, UpButtonUsed, DownButtonUsed, PanicUsed             bool
	StreamToggleUsed, CommentUsed, ListeningUsed, RotaryUsed, RotaryButtonUsed     bool
	VolUpButtonUsed, VolDownButtonUsed, TrackingUsed, MQTT0ButtonUsed               bool
	MQTT1ButtonUsed, NextServerButtonUsed, RepeaterToneButtonUsed                  bool
	MemoryChannelButton1Used, MemoryChannelButton2Used, MemoryChannelButton3Used    bool
	MemoryChannelButton4Used, ShutdownButtonUsed                                   bool
	VoiceTargetButton1Used, VoiceTargetButton2Used, VoiceTargetButton3Used         bool
	VoiceTargetButton4Used, VoiceTargetButton5Used                                 bool

	TxButtonPin, TxTogglePin, UpButtonPin, DownButtonPin, PanicButtonPin           uint
	StreamButtonPin, CommentButtonPin, ListeningButtonPin, RotaryAPin, RotaryBPin  uint
	RotaryButtonPin, VolUpButtonPin, VolDownButtonPin, TrackingButtonPin           uint
	MQTT0ButtonPin, MQTT1ButtonPin, NextServerButtonPin, RepeaterToneButtonPin     uint
	MemoryChannelButton1Pin, MemoryChannelButton2Pin, MemoryChannelButton3Pin      uint
	MemoryChannelButton4Pin, ShutdownButtonPin                                     uint
	VoiceTargetButton1Pin, VoiceTargetButton2Pin, VoiceTargetButton3Pin            uint
	VoiceTargetButton4Pin, VoiceTargetButton5Pin                                   uint
)

var D [8]*mcp23017.Device

func (b *Talkkonnect) initGPIO() {
	if Config.Global.Hardware.TargetBoard != "rpi" {
		return
	}
	if _, err := host.Init(); err != nil {
		log.Println("error: GPIO Host Init Failed: ", err)
		b.GPIOEnabled = false
		return
	}
	b.GPIOEnabled = true

	for _, io := range Config.Global.Hardware.IO.Pins.Pin {
		if io.Enabled && io.PinNo > 0 && io.Type == "gpio" {
			p := gpioreg.ByName(strconv.Itoa(int(io.PinNo)))
			if p == nil { continue }
			if io.Direction == "input" {
				p.In(gpio.PullUp, gpio.BothEdges)
				b.mapLegacyVariables(io.Name, io.PinNo)
			} else {
				p.Out(gpio.Low)
			}
			pioPins[io.Name] = p
		}
	}
	createEnabledRotaryEncoderFunctions()
	analogCreateZones()
	b.startWatchers()
}

func (b *Talkkonnect) findEnabledRotaryEncoderFunction(mode string) bool {
	if len(EnabledRotaryEncoderFunctions) == 0 { return false }
	return EnabledRotaryEncoderFunctions[CurrentRotaryEncoderFunction] == mode
}

func Max7219(device int, x int, y int, char byte, msg string) {}

func createEnabledRotaryEncoderFunctions() {
	EnabledRotaryEncoderFunctions = []string{"volume", "channel"}
	CurrentRotaryEncoderFunction = 0
}

func analogCreateZones() {}

func GPIOOutAll(device string, state string) {
	for name := range pioPins { GPIOOutPin(name, state) }
}

func GPIOOutPin(name string, state string) {
	p, ok := pioPins[name]
	if !ok { return }
	if state == "on" || state == "1" { p.Out(gpio.High) } else { p.Out(gpio.Low) }
}

func GPIOInputPinControl(name string, param string) uint {
	p, ok := pioPins[name]
	if !ok { return 1 }
	if p.Read() == gpio.Low { return 0 }
	return 1
}

func GPIOOutputPinControl(name string, state string) { GPIOOutPin(name, state) }

func (b *Talkkonnect) mapLegacyVariables(name string, pin uint) {
	switch name {
	case "txptt": TxButtonUsed = true; TxButtonPin = pin
	case "channelup": UpButtonUsed = true; UpButtonPin = pin
	case "channeldown": DownButtonUsed = true; DownButtonPin = pin
	case "rotarya": RotaryUsed = true; RotaryAPin = pin
	case "rotaryb": RotaryUsed = true; RotaryBPin = pin
	case "rotarybutton": RotaryButtonUsed = true; RotaryButtonPin = pin
    case "comment": CommentUsed = true; CommentButtonPin = pin
	}
}

func (b *Talkkonnect) startWatchers() {
	if TxButtonUsed { go b.watch("txptt", b.handlePTT) }
	if UpButtonUsed { go b.watch("channelup", func(p bool) { if p { b.ChannelUp() } }) }
	if DownButtonUsed { go b.watch("channeldown", func(p bool) { if p { b.ChannelDown() } }) }
	if RotaryButtonUsed {
		go b.watch("rotarybutton", func(pressed bool) {
			if pressed {
				CurrentRotaryEncoderFunction = (CurrentRotaryEncoderFunction + 1) % len(EnabledRotaryEncoderFunctions)
			}
		})
	}
	if RotaryUsed { go b.watchRotary() }
}

func (b *Talkkonnect) handlePTT(pressed bool) {
	if !IsConnected { return }
	if pressed {
		if !isTx { isTx = true; b.TransmitStart() }
	} else {
		if isTx { isTx = false; b.TransmitStop(true) }
	}
}

func (b *Talkkonnect) watch(name string, action func(bool)) {
	p, ok := pioPins[name]
	if !ok { return }
	last := p.Read()
	for {
		curr := p.Read()
		if curr != last {
			action(curr == gpio.Low)
			last = curr
		}
		time.Sleep(30 * time.Millisecond)
	}
}

func (b *Talkkonnect) watchRotary() {
	pA, okA := pioPins["rotarya"]
	pB, okB := pioPins["rotaryb"]
	if !okA || !okB { return }
	lastA := pA.Read()
	for {
		currA := pA.Read()
		if currA != lastA && currA == gpio.Low {
			direction := "ccw"
			if pB.Read() != currA { direction = "cw" }
			
			currentMode := EnabledRotaryEncoderFunctions[CurrentRotaryEncoderFunction]
			if currentMode == "volume" {
				if direction == "cw" { b.cmdVolumeRXUp() } else { b.cmdVolumeRXDown() }
			} else {
				if direction == "cw" { b.ChannelUp() } else { b.ChannelDown() }
			}
		}
		lastA = currA
		time.Sleep(2 * time.Millisecond)
	}
}
