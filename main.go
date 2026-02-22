package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[*websocket.Conn]string)
var broadcast = make(chan Message)
var mutex = &sync.Mutex{}

var game = &Game{
	TeamA: Team{
		Name:    "Team A",
		Score:   0,
		Players: make(map[*websocket.Conn]bool),
	},
	TeamB: Team{
		Name:    "Team B",
		Score:   0,
		Players: make(map[*websocket.Conn]bool),
	},
}

type Game struct {
	TeamA Team
	TeamB Team
}

type Message struct {
	Team string
	Text []byte
}

type Team struct {
	Name    string
	Score   int
	Players map[*websocket.Conn]bool
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Upgrade error:", err)
		return
	}

	defer conn.Close()

	mutex.Lock()
	clients[conn] = ""
	mutex.Unlock()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			mutex.Lock()
			delete(clients, conn)
			mutex.Unlock()
			break
		}

		stringMessage := string(message)
		if stringMessage == "A" {
			game.TeamA.Players[conn] = true
			clients[conn] = "A"
			continue
		} else if stringMessage == "B" {
			game.TeamB.Players[conn] = true
			clients[conn] = "B"
			continue
		}

		team := clients[conn]

		if team == "" {
			continue
		}

		msg := Message{
			Team: team,
			Text: message,
		}
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		message := <-broadcast

		mutex.Lock()

		selectedTeam := game.TeamA
		if message.Team == "B" {
			selectedTeam = game.TeamB
		}

		for client := range selectedTeam.Players {
			err := client.WriteMessage(websocket.TextMessage, message.Text)
			if err != nil {
				client.Close()
				delete(selectedTeam.Players, client)
			}
		}
		mutex.Unlock()
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	http.Handle("/", http.FileServer(http.Dir("./static")))

	go handleMessages()

	fmt.Println("WebSocket server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
