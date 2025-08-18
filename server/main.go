package main
func main() {
	go runTcp()
	go runUdp()
	go runRudp()
	select {} // Keep the main goroutine running
}