//// One server Accept()ing on a StreamListener, and one client that Dials
//// through I2P to the server. Server writes "Hello world!" through a SAMConn
//// (which implements net.Conn) and the client prints the message.
// https://pkg.go.dev/github.com/eyedeekay/sam3#example-StreamListener

package main

import (
	"sync"
	"unsafe"

	"github.com/eyedeekay/i2pkeys"
	"github.com/eyedeekay/sam3"
)

const samBridge = "127.0.0.1:7656"

var mux sync.Mutex
var message string

func main() {
	var sam, _ = sam3.NewSAM(samBridge)
	defer func(sam *sam3.SAM) {
		_ = sam.Close()
	}(sam)
	var serverKeys, _ = sam.NewKeys()
	var serverSession, _ = sam.NewStreamSession("serverWASI", serverKeys, sam3.Options_Wide)
	start := make(chan bool)

	//quit := make(chan bool)
	go func(server i2pkeys.I2PAddr) { //// Start client session

		sam, _ := sam3.NewSAM(samBridge)
		defer func(clientSAM *sam3.SAM) {
			_ = clientSAM.Close()
		}(sam)
		keys, _ := sam.NewKeys()

		<-start // stop

		clientSession, _ := sam.NewStreamSession("clientWASI", keys, sam3.Options_Wide)
		var connection *sam3.SAMConn
		for { // may fail, depending on the I2P network
			if conn, err := clientSession.DialI2P(server); err != nil {
				Print(err.Error())
			} else {
				connection = conn
				break
			}
		}
		buf := make([]byte, 256)
		n, _ := connection.Read(buf)
		Print(string(buf[:n]))
	}(serverKeys.Addr()) //// end of client

	var streamListener, _ = serverSession.Listen() // STREAM STATUS RESULT=OK
	start <- true
	conn, _ := streamListener.Accept()

	mux.Lock()
	_, _ = conn.Write([]byte(message))
	mux.Unlock()
	for {
	}
	//<- quit // keep main alive https://wazero.io/languages/tinygo/#concurrency
}

// https://k33g.hashnode.dev/wazero-cookbook-part-two-host-functions#heading-create-a-new-wasm-module
// // export hostPrintString
func hostPrintString(pos, sisze uint32) uint32

// // Print a string
func Print(message string) {
	buffer := []byte(message)
	bufferPtr := &buffer[0]
	unsafePtr := uintptr(unsafe.Pointer(bufferPtr))

	pos := uint32(unsafePtr)
	size := uint32(len(buffer))

	hostPrintString(pos, size)
}

// // export hello
func hello(valuePosition *uint32, length uint32) uint64 {

	mux.Lock()
	//// read the memory to get the parameter
	message = string(readBufferFromMemory(valuePosition, length))
	mux.Unlock()

	//// copy the value to memory
	posSizePairValue := copyBufferToMemory([]byte(message))

	//// return the position and size
	return posSizePairValue
}

// // readBufferFromMemory returns a buffer from WebAssembly
func readBufferFromMemory(bufferPosition *uint32, length uint32) []byte {
	subjectBuffer := make([]byte, length)
	pointer := uintptr(unsafe.Pointer(bufferPosition))
	for i := 0; i < int(length); i++ {
		s := *(*int32)(unsafe.Pointer(pointer + uintptr(i)))
		subjectBuffer[i] = byte(s)
	}
	return subjectBuffer
}

// // copyBufferToMemory returns a single value
// // (a kind of pair with position and length)
func copyBufferToMemory(buffer []byte) uint64 {
	bufferPtr := &buffer[0]
	unsafePtr := uintptr(unsafe.Pointer(bufferPtr))

	ptr := uint32(unsafePtr)
	size := uint32(len(buffer))

	return (uint64(ptr) << uint64(32)) | uint64(size)
}
