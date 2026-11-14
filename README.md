# lorahal

Pure-Go driver for Semtech SX1301 LoRa concentrator cards behind the classic
"PicoCell GW" STM32 USB↔SPI bridge — notably the **MikroTik R11e-LR8/LR9**,
which until now was only documented to work with user-soldered SPI wiring or
RouterOS. Plugged into an mPCIe→USB adapter it simply shows up as USB
CDC-ACM and works as a full LoRaWAN concentrator from any unix box.

A faithful Go port of the host half of
[Semtech picoGW_hal](https://github.com/Lora-net/picoGW_hal) (BSD), cleanly
split into two packages:

- **`bridge`** — the MCU bridge wire protocol over USB CDC-ACM
  (115200 baud, framed commands, burst chunking, MCU command set).
- **`hal`** — the SX1301 driver on top: version checks, board/radio/IF/TXgain
  config, radio calibration, AGC/arbiter firmware upload (blobs included,
  1:1 from picoGW_hal), start/stop, RX fetch, TX, time-on-air.

## Quickstart

```go
package main

import (
    "fmt"
    "log"

    "github.com/HanEmile/lorahal/hal"
)

func main() {
    h, err := hal.Connect("/dev/ttyACM0")
    if err != nil {
        log.Fatal(err)
    }
    defer h.Close(false)

    fmt.Printf("MCU %x, bridge fw %#x, SX1301 v%d\n", h.MCUUid, h.MCUFWUsed, h.SX1301Ver)

    err = h.BoardSetconf(hal.BoardConf{LorawanPublic: true, ClkSrc: 1})
    err = h.RxrfSetconf(0, hal.RxRFConf{Enable: true, FreqHz: 867500000, RssiOffset: -215.4, Type: hal.RadioTypeSX1257, TxEnable: true})
    err = h.RxrfSetconf(1, hal.RxRFConf{Enable: true, FreqHz: 868500000, RssiOffset: -215.4, Type: hal.RadioTypeSX1257})
    err = h.TxgainSetconf(hal.DefaultTxGainLUT())
    for ch := 0; ch < 10; ch++ {
        if err := h.RxifSetconf(uint8(ch), hal.RxIFConf{Enable: false}); err != nil {
            log.Fatal(err)
        }
    }
    if err := h.Start(); err != nil {
        log.Fatal(err)
    }

    for {
        pkts, _ := h.Receive(16)
        for _, p := range pkts {
            fmt.Printf("%.3fMHz SF%d rssi=%.1f %x\n", float64(p.FreqHz)/1e6, p.Datarate, p.Rssi, p.Payload)
        }
    }
}
```

See `examples/` for a complete configuration incl. channel plan.

## Bridge protocol (reverse-engineered docs)

The only public documentation of this card says "SPI over miniPCIe", which is
outdated: current R11e-LR8 hardware carries an STM32 that implements Semtech's
PicoCell host protocol over USB CDC-ACM:

- Frames: host→MCU `[cmdID][len_hi][len_lo][addr][data...]`,
  MCU→host `[cmdID][len_hi][len_lo][status][data...]`.
- `w`/`r` single-register write/read. Burst: `a` (atomic write), `x,y,z`
  (write burst start/middle/end, 600 B chunks), `p` (atomic read),
  `s,t,u` (read burst start/middle/end, 900 B chunks).
- MCU commands: `i` board config, `c` RF chain config, `d` IF chain config,
  `h` TX gain LUT, `b` receive FIFO, `f` transmit, `q` internal µs counter,
  `j` commit radio calibration, `l` bridge-firmware version check +
  unique ID, `m` reset MCU, `n` jump to bootloader (DFU).
- The `l` command negotiates bridge firmware: host sends its expected
  version as a 4-byte word, the MCU ACKs iff compatible. Known-good values
  on MikroTik R11e-LR8: `0x010a0004`–`0x010a0006`.

## Compatibility

Known working: MikroTik R11e-LR8 (SX1301/rev. 103, bridge fw 0x010a0006). The
R11e-LR9 is the US915 variant and should work identically. Cards behind the
reference Semtech PicoCell GW STM32 bridge should work by design.

## License

BSD-3 (see LICENSE). The register map and MCU firmware blobs in
`hal/regs.go` / `hal/fw/` are derived from Semtech's picoGW_hal (BSD).
