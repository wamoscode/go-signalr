package signalr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (

	// Invocation Indicates a request to invoke a particular method(the Target) with provided
	// Arguments on the remote endpoint
	Invocation = iota + 1

	// StreamItem Indicates individual Items of streamed response data from a previous StreamInvocation message
	StreamItem

	// Completion Indicates a previous Invocation or StreamInvocation has Completed
	// Contains an error if the invocation concluded with an error or result of non-streaming method invocation
	// The result will be absent for void methods.
	// In case of streaming invocations no further StreamItem messages will be received
	Completion

	// StreamInvocation Indicates a request to invoke a streaming method (the target)
	// with provided Arguments on the remote endpoint
	StreamInvocation

	// CancelInvocation Sent by the client to cancel a streaming invocation on the server
	CancelInvocation

	// Ping Sent by either party to check if connection is active
	Ping

	// Close Sent by the server when a connection is closed.
	// Contains an error if the connection was closed because of an error
	Close

	// HandshakeRequest Sent by the client to agree on the message format
	HandshakeRequest

	// HandshakeResponse Sent by the server as an acknowledgement of teh previous HandshakeRequest message.
	// Contains an error if the handshake failed
	HandshakeResponse

	// Time allowed to write message to the peer
	waitWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512

	maxRedirects = 100
)

var signalrClient *Client

// InvokeMessage ...
type InvokeMessage struct {
	// Data consumed by the server method
	Arguments []interface{} `json:"arguments"`

	// Unique ID to represent the invocation
	// In not set, It indicates that the invocation is "non-blocking" and thus the caller does not expect a response.
	InvocationID string `json:"invocationId"`

	// Server method being called
	Target string `json:"target"`

	// type of message being sent
	Type int `json:"type"`
}

// CompletionMessage ...
type CompletionMessage struct {
	Type int `json:"type"`

	// Unique ID to represent the invocation
	InvocationID string `json:"invocationId"`

	// Result of the invocation
	Result string `json:"result"`

	Error string `json:"error"`
}

// PingMessage ...
type PingMessage struct {
	Type int `json:"type"`
}

// CloseMessage ...
type CloseMessage struct {
	Type  int    `json:"type"`
	Error string `json:"error"`
}

// HandshakeRequestMessage ...
type HandshakeRequestMessage struct {
	// protocol - the name of the protocol to be used for messages exchanged between the server and the client
	// can be messagepack, json,
	Protocol string `json:"protocol"`

	// version - the value must always be 1, for both MessagePack and Json protocols
	Version int `json:"version"`
}

// HandshakeResponseMessage ...
type HandshakeResponseMessage struct {
	// error - the optional error message if the server does not supoort the requested protocol
	Error string `json:"error"`
}

// NegotiationRequestPayload ...
type NegotiationRequestPayload struct {
	// URL the client should connect to
	URL string `json:"url"`

	// Optional bearer  token for accessing the specified url
	AccessToken string `json:"accessToken"`
}

// NegotiationError ...
type NegotiationError struct {
	Error string `json:"error"`
}

// NegotiationResponse ...
type NegotiationResponse struct {
	ConnectionID        string               `json:"connectionId"`
	AvailableTransports []AvailableTransport `json:"availableTransports"`
	URL                 string               `json:"url"`
	AccessToken         string               `json:"acccessToken"`
	Error               string               `json:"error"`
}

// AvailableTransport ...
type AvailableTransport struct {
	Transport        string   `json:"transport"`
	TransportFormats []string `json:"transferFormats"`
}

// Client ...
type Client struct {
	Conn   *websocket.Conn
	Params NegotiationResponse

	mutex *sync.Mutex
}

func init() {
	signalrClient = New()
}

// New ...
func New() *Client {
	return &Client{
		Conn:   &websocket.Conn{},
		Params: NegotiationResponse{},
		mutex:  &sync.Mutex{},
	}
}

// Start ...
func Start(p NegotiationRequestPayload) error {
	return signalrClient.start(p)
}

// Stop disables socket connection
func Stop() error {
	return signalrClient.stop()
}

// Read retrieves response on the socket
func Read(f func(d InvokeMessage)) (n int, p []byte, err error) {
	return signalrClient.read(f)
}

// Send writes data on the socket
func Send(m []byte) error {
	return signalrClient.send(m)
}

// Conn returns Websocket connection
func Conn() *websocket.Conn {
	return signalrClient.Conn
}

func (c *Client) start(p NegotiationRequestPayload) error {

	c.mutex.Lock()
	defer c.mutex.Unlock()

	nRes, err := negotiate(p)
	if err != nil {
		return err
	}

	c.Params = nRes
	conn, err := connect(p, nRes)
	if err != nil {
		return err
	}

	c.Conn = conn

	var hsm HandshakeRequestMessage
	hsm.Protocol = "json"
	hsm.Version = 1

	data, err := json.Marshal(hsm)
	if err != nil {
		return err
	}

	err = c.send(data)
	if err != nil {
		return err
	}

	return nil

}

// This request is used to establish a connection betwen the client and the server
// Connection type of the response is application/json
func negotiate(p NegotiationRequestPayload) (NegotiationResponse, error) {
	var response NegotiationResponse
	connectionURL, err := url.Parse(p.URL)
	if err != nil {
		return response, err
	}

	connectionURL.Path = connectionURL.Path + "/negotiate"
	req, err := http.NewRequest("POST", connectionURL.String(), nil)
	if err != nil {
		return response, err
	}

	req.Header.Add("Content-Type", "application/json")
	if p.AccessToken != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", p.AccessToken))
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return response, err
	}

	defer res.Body.Close()

	if body, err := ioutil.ReadAll(res.Body); err != nil {
		return response, err
	} else if err := json.Unmarshal(body, &response); err != nil {
		return response, err
	} else {
		return response, nil
	}
}

func connect(p NegotiationRequestPayload, params NegotiationResponse) (*websocket.Conn, error) {
	var urlParams = url.Values{}

	connectionURL, err := url.Parse(p.URL)
	if err != nil {
		return nil, err
	}

	urlParams.Set("id", params.ConnectionID)

	connectionURL.Scheme = "ws"
	connectionURL.RawQuery = fmt.Sprintf("%s&%s", urlParams.Encode(), connectionURL.RawQuery)

	reqHeaders := http.Header{}
	if p.AccessToken != "" {
		reqHeaders.Add("Authorization", fmt.Sprintf("Bearer %s", p.AccessToken))
	}

	conn, _, err := websocket.DefaultDialer.Dial(connectionURL.String(), reqHeaders)
	if err != nil {
		return nil, err
	}

	return conn, nil

}

func (c *Client) stop() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.Conn.Close()
}

func (c *Client) send(m []byte) error {
	message := mFormat.write(string(m))
	return c.Conn.WriteMessage(websocket.TextMessage, []byte(message))
}

func (c *Client) read(f func(d InvokeMessage)) (n int, p []byte, err error) {
	_, p, err = c.Conn.ReadMessage()
	if err != nil {
		return
	}

	formatedMessage := mFormat.parse(p)
	var data InvokeMessage
	_ = json.Unmarshal(formatedMessage, &data)

	n = data.Type

	switch n {
	case Invocation:
		go f(data)
		break
	case Completion:
		// TODO add handler
		break
	}

	return
}
