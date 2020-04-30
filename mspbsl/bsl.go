package mspbsl

import (
	"encoding/binary"
	"errors"
	"log"

	"github.com/janch32/fitkit-serial/memory"
)

const (
	massEraseCycles = 1
)

// TxPasswd - Transmit password, default if nil is given
func (b *Instance) TxPasswd(passwd []byte, wait bool) error {
	if passwd == nil {
		log.Print("Transmitting default password...")
		passwd = make([]byte, 32)
		for i := 0; i < len(passwd); i++ {
			passwd[i] = 0xFF
		}
	}

	if len(passwd) != 32 {
		return errors.New("Password has wrong length: " + string(len(passwd)))
	}

	_, err := b.BslTxRx(
		constBslTxPword, //Command: Transmit Password
		0xFFE0,          // Address of interupt vectors
		0x0020,          // Number of bytes
		passwd,          // Password
		wait,            // If wait is 1, try to sync forever
	)

	return err
}

// Prepare to download patch
func (b *Instance) preparePatch() error {
	if b.patchLoaded {
		// Load PC with 0x0220.
		// This will invoke the patched bootstrap loader subroutines.
		_, err := b.BslTxRx(
			constBslLoadPC,
			0x0220, // Address to load into PC
			0,
			nil,
			false,
		)

		if err != nil {
			return err
		}

		b.bslMemAccessWarning = false
	}

	return nil
}

// Setup after the patch is loaded
func (b *Instance) postPatch() {
	if b.patchLoaded {
		b.bslMemAccessWarning = true
	}
}

// Verify memory against data or 0xFF
func (b *Instance) verifyBlk(address uint32, blkout []byte, action byte) error {
	if (action&constActionVerify == 0) && (action&constActionEraseCheck == 0) {
		return nil
	}

	err := b.preparePatch()
	if err != nil {
		return err
	}

	blkin, err := b.BslTxRx(
		constBslRxBLK,
		uint16(address),
		uint16(len(blkout)),
		nil,
		false,
	)
	if err != nil {
		return err
	}

	b.postPatch()

	for i := 0; i < len(blkout); i++ {
		if action&constActionVerify > 0 {
			// Compare data in blkout and blkin
			if blkin[i] != blkout[i] {
				return errors.New(ErrVerifyFailed)
			}
		} else if action&constActionEraseCheck > 0 {
			// Compare data in blkin with erase pattern
			if blkin[i] != 0xFF {
				return errors.New(ErrEraseCheckFailed)
			}
		}
	}

	return nil
}

// Program a memory block
func (b *Instance) programBlk(address uint32, blkout []byte, action byte) error {
	// Check, if specified range is erased
	err := b.verifyBlk(address, blkout, action&constActionEraseCheck)
	if err != nil {
		return err
	}

	if action&constActionProgram > 0 {
		err = b.preparePatch()

		if err == nil {
			_, err = b.BslTxRx(
				constBslTxBLK,
				uint16(address),
				uint16(len(blkout)),
				blkout,
				false,
			)
		}

		if err != nil {
			return err
		}

		b.postPatch()
	}

	return b.verifyBlk(address, blkout, action&constActionVerify)
}

// Program or verify data
func (b *Instance) programData(memory *memory.Memory, action byte) error {
	segments := memory.GetDataSegments()

	for _, seg := range segments {
		currentAddr := seg.Address
		pstart := 0
		count := 0

		for pstart < len(seg.Data) {
			length := constMaxData
			if pstart+length > len(seg.Data) {
				length = len(seg.Data) - pstart
			}

			err := b.programBlk(currentAddr, seg.Data[pstart:pstart+length], action)
			if err != nil {
				return err
			}

			pstart += length
			currentAddr += uint32(length)
			b.byteCtr += length // total sum
			count += length
		}
	}

	return nil
}

// ActionMassErase - Erase the flash memory completely (with mass erase command)
func (b *Instance) ActionMassErase(sendPasswd bool, bslReset bool) error {
	if bslReset {
		b.BslReset(true)
	}

	for i := 0; i < massEraseCycles; i++ {
		_, err := b.BslTxRx(
			constBslMEras, // Command: Mass Erase
			0xFFFE,        // Any address within flash memory.
			0xA506,        // Required setting for mass erase!
			nil,
			false,
		)

		if err != nil {
			return err
		}
	}

	b.Passwd = nil
	if !sendPasswd {
		return nil
	}

	return b.TxPasswd(nil, false)
}

// ActionStartBSL - Start BSL, download patch if desired and needed, adjust SP
// if desired, download replacement BSL, change baudrate.
func (b *Instance) ActionStartBSL(
	usePatch bool,
	adjsp bool,
	speed int,
	bslReset bool,
	sendPasswd bool,
) error {
	if bslReset {
		err := b.BslReset(true)
		if err != nil {
			return err
		}
	}

	if sendPasswd {
		err := b.TxPasswd(b.Passwd, false)
		if err != nil {
			return err
		}
	}

	// Read actual bootstrap loader version.
	blkin, err := b.BslTxRx(
		constBslRxBLK, // Command: Read/Receive Block
		0x0ff0,        // Start address
		16,            // No. of bytes to read
		nil,
		false,
	)
	if err != nil {
		return err
	}

	blkin = blkin[:len(blkin)-2]
	devID := binary.BigEndian.Uint16(blkin)
	bslVerHi := blkin[10]
	bslVerLo := blkin[11]

	if b.CPUFamily == "" {
		cpu, ok := deviceIDs[devID]
		if !ok {
			log.Printf("Autodetect failed! Unknown ID: %04x. Trying to continue anyway.\n", cpu)
			cpu = F1x
		}

		b.CPUFamily = cpu
		b.DevID = devID
	}

	b.bslVer = (uint16(bslVerHi) << 8) | uint16(bslVerLo)

	// Check if patch is needed. Fixed in newer versions of BSL.
	b.bslMemAccessWarning = b.bslVer <= 0x0110

	if b.bslVer <= 0x0130 && adjsp {
		// only do this on BSL where it's needed to prevent
		// malfunction with F4xx devices/ newer ROM-BSLs

		// Execute function within bootstrap loader
		// to prepare stack pointer for the following patch.
		// This function will lock the protected functions again.
		_, err := b.BslTxRx(
			constBslLoadPC, // Command: Load PC
			0x0C22,         //Address to load into PC
			0,
			nil,
			false,
		)

		// Re-send password to re-gain access to protected functions.
		if err == nil {
			err = b.TxPasswd(b.Passwd, false)
		}

		if err != nil {
			return err
		}
	}

	// Now apply workarounds or patches if BSL in use requires that
	if b.bslVer <= 0x0110 && usePatch {
		b.patchRequired = true
		mem, err := memory.LoadTIText(patch)

		if err == nil {
			err = b.programData(mem, constActionProgram|constActionVerify)
		}

		if err != nil {
			return err
		}
		b.patchLoaded = true
	}

	if speed != 0 {
		b.actionChangeBaudrate(speed)
	}

	return nil
}

// Change baudrate. The command is sent first, then the comm
// port is reprogrammed. Only possible with newer MSP430-BSL versions.
// (ROM >= 1.6, downloadable >= 1.5)
func (b *Instance) actionChangeBaudrate(baudrate int) error {
	var baudEntry baudrateEntry
	ok := false

	switch b.CPUFamily {
	case F1x:
		baudEntry, ok = baudrateF1x[baudrate]
		break
	case F2x:
		baudEntry, ok = baudrateF2x[baudrate]
		break
	case F4x:
		baudEntry, ok = baudrateF4x[baudrate]
		break
	default:
		return errors.New("Unknown CPU type " + b.CPUFamily + ", can't switch baudrate")
	}

	if !ok {
		return errors.New("Baudrate not valid")
	}

	_, err := b.BslTxRx(
		constBslChangeBaud, // Command: change baudrate
		baudEntry.addr,     // Args are coded in adr and len
		baudEntry.len,
		nil,
		false,
	)

	if err != nil {
		return err
	}

	sleepMs(10)
	return b.SetBaudRate(baudrate)

}

// ActionProgram - Program data into flash memory
func (b *Instance) ActionProgram(memory *memory.Memory) error {
	err := b.programData(memory, constActionProgram)
	return err
}

// ActionRun - Start program at specified address
func (b *Instance) ActionRun(address uint16) error {
	_, err := b.BslTxRx(
		constBslLoadPC,
		address,
		0,
		nil,
		false,
	)
	return err
}
