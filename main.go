package main

import (
	"flag"
	"fmt"

	"./discover"
	"./flash"
	"./terminal"
)

func main() {
	flagListPorts := flag.Bool("list", false, "List all connected FITkits")
	flagTerm := flag.Bool("term", false, "Connect to FITkit and open terminal")
	flagFlash := flag.Bool("flash", false, "Flash FITkit MCU and FPGA")
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
		terminal.Open(*flagPort)
	} else if *flagFlash {
		if *flagPort == "" {
			fmt.Println("Port not specified, running port autodiscovery...")
			*flagPort = discover.FirstDevice().Port
		}

		fmt.Println("Connecting to " + *flagPort)
		flash.Flash(*flagPort, *flagMcu1Hex, *flagMcu2Hex, *flagFpgaBin)
	} else {
		fmt.Println("Run with -help to show available flags")
	}
}
