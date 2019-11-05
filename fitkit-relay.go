package main;

import "go.bug.st/serial.v1"
import "log"
import "fmt"

func main() {
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
