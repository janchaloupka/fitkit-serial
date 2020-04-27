package main

import (
	"flag"
	"fmt"
	"log"

	"./discover"
	"./terminal"
)

func main() {
	flagListPorts := flag.Bool("list", false, "List all connected FITkits")
	flagTerm := flag.Bool("term", false, "Connect to FITkit and open terminal")
	flagFlash := flag.Bool("flash", false, "Flash FITkit MCU and FPGA")
	flagPort := flag.String("port", "", "Specify which serial port should be used (optional)")

	flag.Parse()

	if *flagListPorts {
		discover.PrintDevices()
	} else if *flagTerm {
		if *flagPort == "" {
			fmt.Println("Port not specified, running auto port discovery...")
			*flagPort = discover.FirstDevice().Port
		}

		fmt.Println("Connecting to: " + *flagPort)
		terminal.Open(*flagPort)
	} else if *flagFlash {
		log.Fatalln("Not implemented")
	} else {
		fmt.Println("Run with -help to show available flags")
	}
}
