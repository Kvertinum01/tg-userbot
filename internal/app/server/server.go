package server

import (
	"log"
	"net/http"

	"github.com/Kvertinum01/userbot-api/internal/app/bot"
	"github.com/gorilla/mux"
)

type server struct {
	sessions *bot.Sessions
}

func SetupServer(config *Config, sessions *bot.Sessions) error {
	r := mux.NewRouter()
	s := &server{
		sessions: sessions,
	}

	r.HandleFunc("/api/v1/create_chat", s.handleCreateChat).Methods("POST")
	r.HandleFunc("/api/v1/send_message", s.handleSendMessage).Methods("POST")
	r.HandleFunc("/api/v1/auth", s.handleAuth).Methods("POST")
	r.HandleFunc("/api/v1/code", s.handleCode).Methods("POST")
	r.HandleFunc("/api/v1/password", s.handlePassword).Methods("POST")

	log.Println("Server started at", config.Addr)

	return http.ListenAndServe(config.Addr, r)
}
