package main;

import(
	"go.bug.st/serial.v1"
	"fmt"
	"log"
	"flag"
) 

func main() {
	flagListPorts := flag.Bool("listports", false, "List all available serial ports")
	flagOpen := flag.String("open", "", "Open serial port communication with FITkit")

	flag.Parse()

	if *flagListPorts{
		listPorts()
	}else if *flagOpen != "" {
		fmt.Println("Connecting to: " + *flagOpen)
	}
}

func listPorts(){
	ports, err := serial.GetPortsList()
	
	if err != nil {
		log.Fatal(err)
	}
	
	if len(ports) == 0 {
		log.Fatal("No serial ports found!")
	}
	
	for _, port := range ports {
		fmt.Println(port)
	}
}
