package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/janch32/fitkit-serial/discover"
)

func main() {
	flagListPorts := flag.Bool("list", false, "List all connected FITkits. Cannot be used with any other argument")
	flagTerm := flag.Bool("term", false, "Connect to FITkit and open terminal. If this argument is used with --flash, then the terminal is opened after successful flashing")
	flagFlash := flag.Bool("flash", false, "Flash FITkit MCU and FPGA")
	flagForce := flag.Bool("force", false, "Force MCU and FPGA flash. Use with --flash")
	flagPort := flag.String("port", "", "Specify which serial port should be used (optional)")
	flagMcu1Hex := flag.String("hex1x", "", "Path to HEX file for v1.x MCU. Use with --flash")
	flagMcu2Hex := flag.String("hex2x", "", "Path to HEX file for v2.x MCU. Use with --flash")
	flagFpgaBin := flag.String("bin", "", "Path to FPGA bin file. Use with --flash")

	flag.Parse()

	if *flagListPorts {
		found := discover.AllDevices()

		b, err := json.MarshalIndent(found, "", "    ")
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(b))
		os.Exit(0)
	}

	if *flagFlash {
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

			fitkit, err := discover.FirstDevice()
			if err != nil {
				log.Fatal(err)
			}
			*flagPort = fitkit.Port
		}

		fmt.Println("Connecting to " + *flagPort)
		Flash(*flagPort, *flagMcu1Hex, *flagMcu2Hex, *flagFpgaBin, *flagForce)

		if !*flagTerm {
			os.Exit(0)
		}
	}

	if *flagTerm {
		if *flagPort == "" {
			fmt.Println("Port not specified, running port autodiscovery...")

			fitkit, err := discover.FirstDevice()
			if err != nil {
				log.Fatal(err)
			}
			*flagPort = fitkit.Port
		}

		fmt.Println("Connecting to " + *flagPort)
		OpenTerminal(*flagPort)

		os.Exit(0)
	}

	fmt.Println("Run with -help to show available flags")
}
