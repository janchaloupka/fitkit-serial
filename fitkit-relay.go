package main;

import(
	"go.bug.st/serial.v1"
	"go.bug.st/serial.v1/enumerator"
	"fmt"
	"log"
	"flag"
) 

func main() {
	flagListPorts := flag.Bool("list", false, "List all connected FITkits")
	flagOpen := flag.String("open", "", "Open serial port communication with FITkit")

	flag.Parse()

	if *flagListPorts{
		listPorts()
	}else if *flagOpen != "" {
		fmt.Println("Connecting to: " + *flagOpen)
		openDevice(*flagOpen)
	}
}

func openDevice(portName string){
	mode := serial.Mode{
		BaudRate: 460800,
	}

	port, err := serial.Open(portName, &mode);
	
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Println("Connected");
	
	buff := make([]byte, 100)
	for {
		n, err := port.Read(buff)
		if err != nil {
			log.Fatal(err)
			break
		}

		if n == 0 || string(buff[:n]) == ">" {
			break
		}
		fmt.Printf("%v", string(buff[:n]))
	}
	
	fmt.Println("Disconnected");
	port.Close()
}

func listPorts(){
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		log.Fatal(err)
	}
	if len(ports) == 0 {
		fmt.Println("No serial ports found!")
		return
	}
	for _, port := range ports {
		fmt.Printf("Found port: %s\n", port.Name)
		if port.IsUSB {
			fmt.Printf("   USB ID     %s:%s\n", port.VID, port.PID)
			fmt.Printf("   USB serial %s\n", port.SerialNumber)
		}
	}

	return

	//ports, err := serial.GetPortsList()
	
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
