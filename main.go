package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/janch32/fitkit-serial/discover"
)

func main() {
	flagListPorts := flag.Bool("list", false, "List all connected FITkits")
	flagTerm := flag.Bool("term", false, "Connect to FITkit and open terminal")
	flagFlash := flag.Bool("flash", false, "Flash FITkit MCU and FPGA")
	flagForce := flag.Bool("force", false, "Force MCU and FPGA flash. Use with --flash")
	flagPort := flag.String("port", "", "Specify which serial port should be used (optional)")
	flagMcu1Hex := flag.String("mcu1hex", "", "Path to HEX file for v1.x MCU. Use with --flash")
	flagMcu2Hex := flag.String("mcu2hex", "", "Path to HEX file for v2.x MCU. Use with --flash")
	flagFpgaBin := flag.String("fpgabin", "", "Path to FPGA bin file. Use with --flash")

	flag.Parse()

	if *flagListPorts {
		discover.PrintDevices()
	} else if *flagTerm {
		if *flagPort == "" {
			fmt.Println("Port not specified, running port autodiscovery...")
			*flagPort = discover.FirstDevice().Port
		}

		fmt.Println("Connecting to " + *flagPort)
		OpenTerminal(*flagPort)
	} else if *flagFlash {
		if *flagMcu1Hex == "" {
			log.Fatal("Must specify MCU v1.x HEX file --mcu1hex")
		}

		if *flagMcu2Hex == "" {
			log.Fatal("Must specify MCU v2.x HEX file --mcu2hex")
		}

		if *flagFpgaBin == "" {
			log.Fatal("Must specify FPGA BIN file --fpgabin")
		}

		if *flagPort == "" {
			fmt.Println("Port not specified, running port autodiscovery...")
			*flagPort = discover.FirstDevice().Port
		}

		fmt.Println("Connecting to " + *flagPort)
		Flash(*flagPort, *flagMcu1Hex, *flagMcu2Hex, *flagFpgaBin, *flagForce)
	} else {
		fmt.Println("Run with -help to show available flags")
	}
}
