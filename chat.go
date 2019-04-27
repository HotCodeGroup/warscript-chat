package main

import (
	"context"
	"net/http"

	"github.com/HotCodeGroup/warscript-utils/models"
	"github.com/HotCodeGroup/warscript-utils/utils"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // мы уже прошли слой CORS
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// SessionInfo достаёт инфу о юзере из контекстаs
func GetSessionInfo(r *http.Request) (*models.SessionPayload, error) {
	cookie, err := r.Cookie("JSESSIONID")
	if err != nil || cookie == nil {
		return nil, errors.Wrap(err, "no cookie")
	}

	session, err := authGPRC.GetSessionInfo(r.Context(), &models.SessionToken{Token: cookie.Value})
	if err != nil {
		return nil, errors.Wrap(err, "can not get session info")
	}

	return session, nil
}

func ConnectChat(hub *Hub, w http.ResponseWriter, r *http.Request) {
	logger := utils.GetLogger(r, logger, "ConnectChat")
	info, err := GetSessionInfo(r)
	if err != nil {
		logger.Warnf("can not get session info from grpc service: %s", err.Error())
	}

	var infoUser *models.InfoUser
	if info != nil {
		var err error
		infoUser, err = authGPRC.GetUserByID(context.Background(), &models.UserID{ID: info.ID})
		if err != nil {
			logger.Warnf("can not get user info from grpc service: %s", err.Error())
		}
	}

	// тут дальше апгрейд до вебсокета
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Errorf("can not upgrade conn: %s", err.Error())
		return
	}

	client := &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan WSObject, 256),
		conversations: []int64{1},
	}

	if infoUser != nil {
		client.info = &UserInfo{
			ID:       infoUser.ID,
			Username: infoUser.Username,
		}
	}

	client.hub.register <- client
	go client.writePump()
	go client.readPump()

	logger.Infof("User: %s connected chat", infoUser.Username)
}
