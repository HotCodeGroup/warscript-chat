package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/HotCodeGroup/warscript-utils/models"
	"github.com/jackc/pgx"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jcftang/logentriesrus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var logger *logrus.Logger
var authGPRC models.AuthClient

func main() {
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)

	// собираем логи в хранилище
	le, err := logentriesrus.NewLogentriesrusHook(os.Getenv("LOGENTRIESRUS_TOKEN"))
	if err != nil {
		log.Printf("can not create logrus logger %s", err)
		return
	}
	logger.AddHook(le)

	dbPort, err := strconv.ParseInt(os.Getenv("DB_PORT"), 10, 16)
	if err != nil {
		logger.Errorf("incorrect database port: %s", err.Error())
		return
	}

	pgxConn, err = pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig{
			Host:     os.Getenv("DB_HOST"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASS"),
			Database: os.Getenv("DB_NAME"),
			Port:     uint16(dbPort),
		},
	})
	if err != nil {
		logger.Errorf("cant connect to postgresql database: %s", err.Error())
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
