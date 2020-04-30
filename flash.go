package main

import (
	"fmt"
	"log"

	"github.com/janch32/fitkit-serial/fitkitbsl"
	"github.com/janch32/fitkit-serial/memory"
	"github.com/janch32/fitkit-serial/mspbsl"
)

// Flash Spustí programování FITkitu na daném portu port
//
// Je nutné uvést cestu k binárnímu soubory pro programování FPGA
// a hex souboru pro naprogramování MCU. Pro MCU je nutné uvést minimálně
// jeden soubor (musí odpovídat připojenímu FITkitu).
// Lepší je uvést obě varianty MCU
func Flash(port string, mcuV1path string, mcuV2path string, fpgaPath string, force bool) {
	b, err := mspbsl.New(port)

	if err != nil {
		log.Fatal(err)
	}

	defer b.ComDone()

	mcu1mem, err := memory.LoadHexFile(mcuV1path)
	if err != nil {
		log.Fatal(err)
	}

	mcu2mem, err := memory.LoadHexFile(mcuV2path)
	if err != nil {
		log.Fatal(err)
	}

	pass1 := mcu1mem.GetMemRange(0xFFE0, 0xFFFF)
	pass2 := mcu2mem.GetMemRange(0xFFE0, 0xFFFF)

	err = b.BslReset(true)
	if err != nil {
		log.Fatal(err)
	}

	massErase := !tryPassword(b, pass1) && !tryPassword(b, pass2)
	if force {
		massErase = true
	}

	if massErase {
		fmt.Println("Mass erase")
		b.ActionMassErase(true, false)
	}

	err = b.ActionStartBSL(true, true, 38400, false, false)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("CPU: %s; Device: %X; FITkitBSLrev: %s\n", b.CPUFamily, b.DevID, fitkitbsl.BSLHexRevision)

	var bslMem *memory.Memory
	mcuHexPath := mcuV1path

	if b.DevID == 0xF169 {
		bslMem, err = memory.LoadTIText(fitkitbsl.BSLHexF1xx)
	} else if b.DevID == 0xF26F {
		bslMem, err = memory.LoadTIText(fitkitbsl.BSLHexF2xx)
		mcuHexPath = mcuV2path
	}

	if err != nil {
		log.Fatal(err)
	}

	if bslMem == nil {
		log.Fatal("Unsupported CPU device")
	}

	fmt.Println("Uploading FITkit bootloader...")
	err = b.ActionProgram(bslMem)
	if err != nil {
		log.Fatal(err)
	}

	err = b.ActionRun(0x228)
	if err != nil {
		log.Fatal(err)
	}

	err = b.SetBaudRate(460800)
	if err != nil {
		log.Fatal(err)
	}

	err = b.FlushInput()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("FITkit bootloader running")

	fkbsl := fitkitbsl.NewFITkitBsl(b)
	err = fkbsl.Program(mcuHexPath, fpgaPath, massErase, force)

	if err != nil {
		log.Fatal(err)
	}

	err = b.ComDone()
	if err != nil {
		log.Fatal(err)
	}
}

func tryPassword(b *mspbsl.Instance, passwd []byte) bool {
	err := b.TxPasswd(passwd, false)

	if err != nil {
		if err.Error() != mspbsl.ErrRxNAK {
			log.Fatal(err)
		}
		return false
	}

	return true
}
