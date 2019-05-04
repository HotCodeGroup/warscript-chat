package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/HotCodeGroup/warscript-utils/balancer"
	"github.com/HotCodeGroup/warscript-utils/logging"
	"github.com/HotCodeGroup/warscript-utils/middlewares"
	"github.com/HotCodeGroup/warscript-utils/models"
	"github.com/HotCodeGroup/warscript-utils/postgresql"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	consulapi "github.com/hashicorp/consul/api"
	vaultapi "github.com/hashicorp/vault/api"
)

var logger *logrus.Logger
var authGPRC models.AuthClient

//nolint: gocyclo
func main() {
	var err error
	logger, err = logging.NewLogger(os.Stdout, os.Getenv("LOGENTRIESRUS_TOKEN"))
	if err != nil {
		log.Printf("can not create logger: %s", err)
		return
	}

	config := consulapi.DefaultConfig()
	config.Address = os.Getenv("CONSUL_ADDR")
	consul, err := consulapi.NewClient(config)
	if err != nil {
		logger.Errorf("can not connect consul service: %s", err)
		return
	}

	vaultConfig := vaultapi.DefaultConfig()
	vaultConfig.Address = os.Getenv("VAULT_ADDR")
	vault, err := vaultapi.NewClient(vaultConfig)
	if err != nil {
		logger.Errorf("can not connect vault service: %s", err)
		return
	}
	vault.SetToken(os.Getenv("VAULT_TOKEN"))

	httpPort, _, err := balancer.GetPorts("warscript-chat/bounds", "warscript-bots", consul)
	if err != nil {
		logger.Errorf("can not find empry port: %s", err)
		return
	}

	// получаем конфиг на постгрес и стартуем
	postgreConf, err := vault.Logical().Read("warscript-chat/postgres")
	if err != nil || postgreConf == nil || len(postgreConf.Warnings) != 0 {
		logger.Errorf("can read warscript-chat/postges key: %+v; %+v", err, postgreConf)
		return
	}
	pgxConn, err = postgresql.Connect(postgreConf.Data["user"].(string), postgreConf.Data["pass"].(string),
		postgreConf.Data["host"].(string), postgreConf.Data["port"].(string), postgreConf.Data["database"].(string))
	if err != nil {
		logger.Errorf("can not connect to postgresql database: %s", err.Error())
		return
	}
	defer pgxConn.Close()

	authGPRCConn, err := balancer.ConnectClient(consul, "warscript-users-grpc")
	if err != nil {
		logger.Errorf("can not connect to auth grpc: %s", err.Error())
		return
	}
	defer authGPRCConn.Close()
	authGPRC = models.NewAuthClient(authGPRCConn)

	h := NewHub()
	go h.run()

	httpServiceID := fmt.Sprintf("warscript-chat-http:%d", httpPort)
	err = consul.Agent().ServiceRegister(&consulapi.AgentServiceRegistration{
		ID:      httpServiceID,
		Name:    "warscript-chat-http",
		Port:    httpPort,
		Address: "127.0.0.1",
	})
	defer func() {
		err = consul.Agent().ServiceDeregister(httpServiceID)
		if err != nil {
			logger.Errorf("can not derigister http service: %s", err)
		}
		logger.Info("successfully derigister http service")
	}()

	r := mux.NewRouter().PathPrefix("/v1").Subrouter()
	r.HandleFunc("/chat/connect", func(w http.ResponseWriter, r *http.Request) {
		ConnectChat(h, w, r)
	}).Methods("GET")

	logger.Infof("Chat HTTP service successfully started at port %d", httpPort)
	err = http.ListenAndServe(":"+strconv.Itoa(httpPort),
		middlewares.RecoverMiddleware(middlewares.AccessLogMiddleware(r, logger), logger))
	if err != nil {
		logger.Errorf("cant start main server. err: %s", err.Error())
		return
	}
}
