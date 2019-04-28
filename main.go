package main

import (
	"log"
	"net/http"
	"os"

	"github.com/HotCodeGroup/warscript-utils/logging"
	"github.com/HotCodeGroup/warscript-utils/models"
	"github.com/HotCodeGroup/warscript-utils/postgresql"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var logger *logrus.Logger
var authGPRC models.AuthClient

func main() {
	var err error
	logger, err = logging.NewLogger(os.Stdout, os.Getenv("LOGENTRIESRUS_TOKEN"))
	if err != nil {
		log.Printf("can not create logger: %s", err)
		return
	}

	pgxConn, err = postgresql.Connect(os.Getenv("DB_USER"), os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))
	if err != nil {
		logger.Errorf("can not connect to postgresql database: %s", err.Error())
		return
	}
	defer pgxConn.Close()

	authGPRCConn, err := grpc.Dial(
		os.Getenv("AUTH_ADDRESS"),
		grpc.WithInsecure(),
	)
	if err != nil {
		logger.Errorf("can not connect to auth grpc")
		return
	}
	defer authGPRCConn.Close()

	authGPRC = models.NewAuthClient(authGPRCConn)

	h := NewHub()
	go h.run()

	r := mux.NewRouter().PathPrefix("/v1").Subrouter()
	r.HandleFunc("/chat/connect", func(w http.ResponseWriter, r *http.Request) {
		ConnectChat(h, w, r)
	}).Methods("GET")

	corsMiddleware := handlers.CORS(
		handlers.AllowedOrigins([]string{os.Getenv("CORS_HOST")}),
		handlers.AllowedMethods([]string{"POST", "GET", "PUT", "DELETE"}),
		handlers.AllowedHeaders([]string{"Content-Type"}),
		handlers.AllowCredentials(),
	)

	httpPort := os.Getenv("PORT")
	logger.Infof("Chat HTTP service successfully started at port %s", httpPort)
	err = http.ListenAndServe(":"+httpPort, corsMiddleware(r))
	if err != nil {
		logger.Errorf("cant start main server. err: %s", err.Error())
		return
	}
}
