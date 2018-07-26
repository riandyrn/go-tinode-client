package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	// initialize client
	tClient := NewTinodeClient()
	err := tClient.OpenSocket()
	if err != nil {
		log.Fatalf("unable to open socket due: %v", err)
	}
	// try to create new user
	username := fmt.Sprintf("test_%v", time.Now().UnixNano())
	password := username
	email := fmt.Sprintf("%v@test.com", username)
	if err = tClient.CreateNewUser(username, password, email); err != nil {
		log.Fatalf("unable to create user due: %v", err)
	}
	log.Printf("successfully create username: %v, with password: %v, with email: %v", username, password, email)
	// close session since we want to try login api
	tClient.CloseSocket()

	// start new client
	tClient = NewTinodeClient()
	if err = tClient.OpenSocket(); err != nil {
		log.Fatalf("unable to open socket due: %v", err)
	}
	loginData, err := tClient.BasicLogin(username, password)
	if err != nil {
		log.Fatalf("unable to login due: %v", err)
	}
	log.Printf("successfully login, loginData: %+v", loginData)
	// close session
	tClient.CloseSocket()
}
