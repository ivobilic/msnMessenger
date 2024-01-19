package main

import (
	"flag"
	"os"

	"github.com/eyedeekay/i2pkeys"
	"github.com/eyedeekay/sam3"
	_ "github.com/stealthrocket/net/wasip1"
)

func main() {
	flag.Parse()
	word := flag.Arg(0)
	_, _ = os.Stdout.WriteString(word + "\n")

	const samBridge = "127.0.0.1:7656"

	var sam, _ = sam3.NewSAM(samBridge)
	defer sam.Close()

	var serverKeys, _ = sam.NewKeys()
	var serverSession, _ = sam.NewStreamSession("serverWASI", serverKeys, sam3.Options_Wide)

	start := make(chan bool)
	quit := make(chan bool)

	go func(server i2pkeys.I2PAddr) { //// Start client session
		sam, _ := sam3.NewSAM(samBridge)
		defer func(clientSAM *sam3.SAM) {
			_ = clientSAM.Close()
		}(sam)
		keys, _ := sam.NewKeys()
		session, _ := sam.NewStreamSession("clientWASI", keys, sam3.Options_Wide)

		<-start // wait for the server to start listening

		var connection *sam3.SAMConn
		for { // may fail, depending on the I2P network
			if conn, err := session.DialI2P(server); err != nil {
				_, _ = os.Stdout.WriteString(err.Error() + "\n") // Can not reach peer
			} else {
				connection = conn
				break
			}
		}
		buf := make([]byte, 256)
		n, _ := connection.Read(buf)
		_, _ = os.Stdout.WriteString("\n" + string(buf[:n]) + "\n") // prints received message

		quit <- true //// waits for client to die, for example only
	}(serverKeys.Addr()) //// end of client

	var streamListener, _ = serverSession.Listen() // STREAM STATUS RESULT=OK
	start <- true
	conn, _ := streamListener.Accept()

	_, _ = conn.Write([]byte(word))

	<-quit
}
