package main;

import(
	"fmt"
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
	}else{
		fmt.Println("Run with -help to show available flags")
	}
}
