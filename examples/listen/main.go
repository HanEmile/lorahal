// Minimal example: connect, configure EU868, print every received packet.
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/HanEmile/lorahal/hal"
)

var dev = flag.String("dev", "/dev/ttyACM0", "serial device of the concentrator")

func main() {
	flag.Parse()

	h, err := hal.Connect(*dev)
	if err != nil {
		log.Fatal(err)
	}
	defer h.Close(false)

	fmt.Printf("MCU bridge uid %x (bridge fw %#x), SX1301 v%d\n", h.MCUUid, h.MCUFWUsed, h.SX1301Ver)

	// EU868, radio 0 centered on 867.5 MHz, radio 1 on 868.5
	if err := h.BoardSetconf(hal.BoardConf{LorawanPublic: true, ClkSrc: 1}); err != nil {
		log.Fatal(err)
	}
	if err := h.RxrfSetconf(0, hal.RxRFConf{Enable: true, FreqHz: 867500000, RssiOffset: -215.4, Type: hal.RadioTypeSX1257, TxEnable: true}); err != nil {
		log.Fatal(err)
	}
	if err := h.RxrfSetconf(1, hal.RxRFConf{Enable: true, FreqHz: 868500000, RssiOffset: -215.4, Type: hal.RadioTypeSX1257}); err != nil {
		log.Fatal(err)
	}
	type ifc struct {
		rf   uint8
		freq int32
	}
	for i, s := range []ifc{{0, -400000}, {0, -200000}, {0, 0}, {0, 200000}, {1, -400000}, {1, -200000}, {1, 0}, {1, 200000}} {
		if err := h.RxifSetconf(uint8(i), hal.RxIFConf{Enable: true, RFChain: s.rf, FreqHz: s.freq, Bandwidth: hal.BW125kHz, Datarate: hal.DRLoraMulti}); err != nil {
			log.Fatal(err)
		}
	}
	if err := h.RxifSetconf(8, hal.RxIFConf{Enable: false}); err != nil {
		log.Fatal(err)
	}
	if err := h.RxifSetconf(9, hal.RxIFConf{Enable: false}); err != nil {
		log.Fatal(err)
	}
	if err := h.TxgainSetconf(hal.DefaultTxGainLUT()); err != nil {
		log.Fatal(err)
	}

	if err := h.Start(); err != nil {
		log.Fatal(err)
	}
	log.Println("listening...")
	for {
		pkts, err := h.Receive(16)
		if err != nil {
			log.Printf("receive: %v", err)
			time.Sleep(time.Second)
			continue
		}
		for _, p := range pkts {
			fmt.Printf("%s %8.3f MHz ch%d rf%d sf%d rssi=%.1f snr=%.1f crc=%02x len=%d %x\n",
				time.Now().Format("15:04:05.000"), float64(p.FreqHz)/1e6, p.IFChain, p.RFChain,
				sfOfDR(p.Datarate), p.Rssi, p.Snr, p.Crc, p.Size, p.Payload)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func sfOfDR(dr uint32) int {
	return map[uint32]int{hal.DRLoraSF7: 7, hal.DRLoraSF8: 8, hal.DRLoraSF9: 9, hal.DRLoraSF10: 10, hal.DRLoraSF11: 11, hal.DRLoraSF12: 12}[dr]
}
