# Overview

SignalR go client is built over websockets based on the client implementation of the Javascript client.

The spec documents below give a deeper insight about the implementation

[TransportProtocols](https://github.com/dotnet/aspnetcore/blob/master/src/SignalR/docs/specs/TransportProtocols.md)

[HubProtocoal](https://github.com/dotnet/aspnetcore/blob/master/src/SignalR/docs/specs/HubProtocol.md)



## Example
Typical Usage example:

```go
package yours

import (
  signalr "github.com/wamoscode/go-signalr"
)

func main() {

   p = signalr.NegotiationRequestPayload{
       URL: "YOUR URL",
       AccessToken: "Add token if auth is required!"
   }

   err = signalr.Start(p)
	if err != nil {
		 return
    }

    errchan := make(chan error, 1)
    done := make(chan bool)
    interrupt := make(chan os.Signal, 1)
	restart := make(chan bool)
 
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
    
    go func() {
		defer func() {
			signalr.Conn().Close()
			restart <- true
		}()
		for {
			t, p, err := signalr.Read(processReceivedMessage)

			if err != nil {
				errchan <- err
				return
			}

			if strings.Contains(string(p), "{}") {
				log.Debug("Handshake Successfull")
			}

			if t == signalr.Close {
				done <- true
				return
			}
		}
    }()
    
    go func() {
		defer func() {
			close(done)
			restart <- true
		}()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				pingMessage := signalr.PingMessage{
					Type: signalr.Ping,
				}
				pm, err := json.Marshal(pingMessage)
				if err != nil {
					return
				}
				err = signalr.Send(pm)
				if err != nil {
					return
				}
			case <-interrupt:
				closeMessage := signalr.CloseMessage{
					Type:  signalr.Close,
					Error: "Interrupt occurred",
				}
				cm, err := json.Marshal(closeMessage)
				if err != nil {
					return
				}
				// send ping
				err = signalr.Send(cm)
				if err != nil {
					return
				}

				select {
				case <-done:
				case <-time.After(time.Second):
				}
				return
			}
		}
    }()
    
    // Wait for restart and error message
    for {
		select {
		case <-restart:
			{
				log.Debug("Signaled Restart")
				// Do something
			}
		case err := <-errchan:
			{
				 // Do something
			} 
		default:
		}

	}
  
}

// Signalr Callback
func processReceivedMessage(data signalr.InvokeMessage) {
   switch data.Target {
	case "YOUR EVENT":
        //Process message
    default:
   }
}
```

## Documentation
SignalR: https://dotnet.microsoft.com/apps/aspnet/signalr