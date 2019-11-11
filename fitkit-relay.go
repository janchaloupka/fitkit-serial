package main;

import(
	"go.bug.st/serial.v1/enumerator"
	"github.com/tarm/serial"
	"fmt"
	"log"
	"flag"
	"time"
	"regexp"
	"encoding/json"
)

type fitkit struct{
	Port		string
	Version		string
	Revision	string
}

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
	
}

func parseFitkitInfo(portName string, info *fitkit) bool{
	fitkitRegex, _ := regexp.Compile("FITkit (.+) \\$Rev: (\\d+) \\$")
	
	config := serial.Config{
		Name: portName,
		Baud: 460800,
		ReadTimeout: time.Millisecond*500,
	}

	port, err := serial.OpenPort(&config);
	
	if err != nil {
		log.Fatal(err)
	}
	
	buff := make([]byte, 64)
	received := 0
	valid := false

	for {
		n, err := port.Read(buff[received:])
		if err != nil {
			log.Fatal(err)
			break
		}

		received += n
		
		submatch := fitkitRegex.FindSubmatch(buff);
		if len(submatch) >= 3{
			info.Port = portName
			info.Version = string(submatch[1])
			info.Revision = string(submatch[2])
			valid = true
			break
		}

		if n == 0 || received >= len(buff) {
			break
		}
	}
	
	port.Close()
	return valid
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

	found := make([]fitkit, 0)
	for _, port := range ports {
		if port.IsUSB && port.VID == "0403" && port.PID == "6010"{
			info := fitkit{}
			if parseFitkitInfo(port.Name, &info){
				found = append(found, info)
			}
		}
	}
	
	b, err := json.Marshal(found)
	if err != nil {
		log.Fatal(err)
    }

	fmt.Println(string(b))
}
