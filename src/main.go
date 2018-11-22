package main

func main() {
	startHTTPServer(":8080")

	//FIXME
	lock := make(chan bool)
	<-lock
}
