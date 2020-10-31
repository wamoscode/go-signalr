package signalr

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

var ngp NegotiationRequestPayload

func TestSignalrConnection(t *testing.T) {
	ngp.URL = "YOUR SOCKET URL TO SIGNALR SERVER"
	err := Start(ngp)
	if err != nil {
		t.Error(err)
	}

	errchan := make(chan error, 1)
	done := make(chan bool)

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			t, p, err := Read(func(d InvokeMessage) { fmt.Println("Received Message: ", d) })

			if err != nil {
				errchan <- err
				break
			}

			// handshake successful
			if strings.Contains(string(p), "{}") {
				errchan <- nil
				signalrClient.stop()
				return
			} else if !strings.Contains(string(p), "{}") {
				errchan <- errors.New("Handshake failed")
				return
			}

			if t == Close {
				done <- true
				return
			}

		}

	}()

	wg.Wait()
	err = <-errchan
	if err != nil {

		t.Error(err)
	}

}

func TestSignalrSendInvocation(t *testing.T) {
	var getbroadcastMsg InvokeMessage
	getbroadcastMsg.InvocationID = "1"
	getbroadcastMsg.Target = "GetBroadCastMessage"
	getbroadcastMsg.Arguments = []interface{}{"walugembeamos@gmail.com"}

	b, err := json.Marshal(getbroadcastMsg)
	if err != nil {
		t.Error(err)
	}

	err = Conn().WriteMessage(Invocation, b)
	if err != nil {
		t.Error(err)
	}

}

func TestSignalrReceiveInvocation(t *testing.T) {

}
