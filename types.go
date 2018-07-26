package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const socketURL = "wss://api.tinode.co/v0/channels?apikey=AQEAAAABAAD_rAp4DJh05a1HAwFT3A6K"

type TinodeClient struct {
	wsClient *websocket.Conn
}

func NewTinodeClient() *TinodeClient {
	return &TinodeClient{}
}

func (c *TinodeClient) OpenSocket() error {
	// open socket connection
	wsClient, _, err := websocket.DefaultDialer.Dial(socketURL, nil)
	if err != nil {
		return fmt.Errorf("unable to open socket due: %v", err)
	}
	// initialize protocol by say hi to server
	defer func() {
		if err != nil {
			// cleanup ws client if any error occurred
			wsClient.Close()
		}
	}()
	if err = wsClient.WriteJSON(NewHi()); err != nil {
		return fmt.Errorf("unable to say hi to server due: %v", err)
	}
	// wait for server response
	var data WsPacket
	if err = wsClient.ReadJSON(&data); err != nil {
		return fmt.Errorf("unable to read response from server due: %v", err)
	}
	if data.Ctrl.Code != http.StatusCreated {
		err = fmt.Errorf("unable to say hi to server due: %v", data.Ctrl.Text)
		return err
	}
	// set successfully initialized ws client to struct
	c.wsClient = wsClient
	return nil
}

func (c *TinodeClient) CloseSocket() {
	c.wsClient.Close()
}

func (c *TinodeClient) WriteRequest(req *WsRequest) (err error) {
	defer func() {
		if err != nil {
			// close ws client if any error occurred
			c.CloseSocket()
		}
	}()
	if err = c.wsClient.WriteJSON(req); err != nil {
		return fmt.Errorf("unable to write request to server due: %v", err)
	}
	return nil
}

func (c *TinodeClient) WaitPacket() (packet *WsPacket, err error) {
	defer func() {
		if err != nil {
			// close ws client if any error occurred
			c.CloseSocket()
		}
	}()
	if err = c.wsClient.ReadJSON(&packet); err != nil {
		return nil, fmt.Errorf("unable to read packet from server due: %v", err)
	}
	return packet, nil
}

func (c *TinodeClient) CreateNewUser(username string, password string, email string) error {
	// issue account creation
	err := c.WriteRequest(NewAcc(username, password, true, email))
	if err != nil {
		return fmt.Errorf("unable to create account due: %v", err)
	}
	packet, err := c.WaitPacket()
	if err != nil {
		return fmt.Errorf("unable to read packet due: %v", err)
	}
	// confirm account creation
	if packet.Ctrl == nil {
		return fmt.Errorf("packet shouldn't be nil")
	}
	if packet.Ctrl.Code != http.StatusMultipleChoices {
		return fmt.Errorf("unable to create account due: %v", packet.Ctrl.Text)
	}
	loginData := packet.Ctrl.GetLoginData()
	if err = c.WriteRequest(NewTokenLoginWithCred(loginData.Token, email)); err != nil {
		return fmt.Errorf("unable to confirm account creation due: %v", err)
	}
	// wait for response
	packet, err = c.WaitPacket()
	if err != nil {
		return fmt.Errorf("unable to receive confirmation account response due: %v", err)
	}
	if packet.Ctrl.Code != http.StatusOK {
		return fmt.Errorf("unable to confirm account due: %v", packet.Ctrl.Text)
	}
	return nil
}

func (c *TinodeClient) BasicLogin(username string, password string) (data *LoginData, err error) {
	if err = c.WriteRequest(NewBasicLogin(username, password)); err != nil {
		return nil, fmt.Errorf("unable to issue login due: %v", err)
	}
	// wait response
	packet, err := c.WaitPacket()
	if err != nil {
		return nil, fmt.Errorf("unable to wait packet due: %v", err)
	}
	if packet == nil {
		return nil, fmt.Errorf("packet shouldn't be nil")
	}
	if packet.Ctrl.Code != http.StatusOK {
		return nil, fmt.Errorf("unable to login due: %v", packet.Ctrl.Text)
	}
	return packet.Ctrl.GetLoginData(), nil
}

type WsRequest struct {
	Hi    *WsCmdHi    `json:"hi"`
	Acc   *WsCmdAcc   `json:"acc"`
	Login *WsCmdLogin `json:"login"`
	Sub   *WsCmdSub   `json:"sub"`
}

type WsCmdHi struct {
	ID        string `json:"id"`
	Version   string `json:"ver"`
	UserAgent string `json:"ua"`
}

type WsCmdAcc struct {
	ID     string          `json:"id"`
	User   string          `json:"user"`
	Scheme string          `json:"scheme"`
	Tags   []string        `json:"tags"`
	Secret string          `json:"secret"`
	Login  bool            `json:"login"`
	Desc   interface{}     `json:"desc"`
	Cred   []*VerifyMethod `json:"cred"`
}

type VerifyMethod struct {
	Method   string `json:"meth"`
	Value    string `json:"val"`
	Response string `json:"resp"`
}

type WsCmdLogin struct {
	ID     string          `json:"id"`
	Scheme string          `json:"scheme"`
	Secret string          `json:"secret"`
	Cred   []*VerifyMethod `json:"cred"`
}

type WsCmdSub struct {
	ID    string    `json:"id"`
	Topic string    `json:"topic"`
	Get   *WsCmdGet `json:"get"`
}

type WsCmdGet struct {
	What string `json:"what"`
}

type WsPacket struct {
	Ctrl *WsCtrlPayload `json:"ctrl"`
}

type LoginData struct {
	Token   string     `json:"token"`
	UserID  string     `json:"user"`
	Expires *time.Time `json:"expires"`
}

type WsCtrlPayload struct {
	ID        string      `json:"id"`
	Params    interface{} `json:"params"`
	Code      int         `json:"code"`
	Text      string      `json:"text"`
	Timestamp *time.Time  `json:"ts"`
}

func (p *WsCtrlPayload) GetLoginData() (data *LoginData) {
	b, err := json.Marshal(p.Params)
	if err != nil {
		return nil
	}
	if err = json.Unmarshal(b, &data); err != nil {
		return nil
	}
	return data
}

func getPacketID() string {
	return fmt.Sprintf("%v", time.Now().UnixNano()) // could be anything, but we use nano timestamp here
}

func NewHi() *WsRequest {
	return &WsRequest{
		Hi: &WsCmdHi{
			ID:        getPacketID(),
			Version:   "0.15",                                                  // could be anything
			UserAgent: "TinodeWeb/0.15 (Chrome/67.0; MacIntel); tinodejs/0.15", // could be anything
		},
	}
}

func NewAcc(username string, password string, login bool, email string) *WsRequest {
	return &WsRequest{
		Acc: &WsCmdAcc{
			ID:     getPacketID(),
			User:   "new", // we need to set it like this if we want to issue new user,
			Scheme: "basic",
			Secret: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v:%v", username, password))),
			Login:  login,
			Desc: map[string]interface{}{
				"public": map[string]interface{}{
					"fn": username,
				},
			},
			Tags: []string{email}, // set tags so user is discoverable by another user
			Cred: []*VerifyMethod{&VerifyMethod{
				Method: "email",
				Value:  email,
			}},
		},
	}
}

func NewBasicLogin(username string, password string) *WsRequest {
	return &WsRequest{
		Login: &WsCmdLogin{
			ID:     getPacketID(),
			Scheme: "basic", // we use basic scheme for login with username password
			Secret: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v:%v", username, password))),
		},
	}
}

func NewTokenLogin(token string) *WsRequest {
	return &WsRequest{
		Login: &WsCmdLogin{
			ID:     getPacketID(),
			Scheme: "token", // use token for token login
			Secret: token,
		},
	}
}

func NewTokenLoginWithCred(token string, email string) *WsRequest {
	return &WsRequest{
		Login: &WsCmdLogin{
			ID:     getPacketID(),
			Scheme: "token", // use token for token login
			Secret: token,
			Cred: []*VerifyMethod{&VerifyMethod{
				Method:   "email",
				Value:    email,
				Response: "123456",
			}},
		},
	}
}

func NewSub(topic string, getQuery string) *WsRequest {
	return &WsRequest{
		Sub: &WsCmdSub{
			ID:    getPacketID(),
			Topic: topic,
			Get: &WsCmdGet{
				What: getQuery,
			},
		},
	}
}
