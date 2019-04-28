package main

import (
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

func logError(logs *logrus.Entry, msg string, err error) {
	if err != nil {
		logs.Errorf("%s: %s", msg, err)
	}
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan WSObject
	info *UserInfo

	conversations []int64
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	logs := logger.WithFields(logrus.Fields{
		"method": "readPump",
	})

	c.conn.SetReadLimit(maxMessageSize)
	err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		logError(logs, "can not set conn read deadline", err)
		return
	}

	c.conn.SetPongHandler(func(string) error {
		if err = c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			logError(logs, "can not set conn read deadline", err)
			return err
		}

		return nil
	})

	for {
		_, jsonMsg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logError(logs, "unexpected close error", err)
			}
			break
		}

		var raw WSObject
		err = json.Unmarshal(jsonMsg, &raw)
		logError(logs, "client msg unmarshal error", err)

		raw.Author = c.info
		raw.client = c
		c.hub.broadcast <- raw
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	logs := logger.WithFields(logrus.Fields{
		"method": "writePump",
	})
	for {
		select {
		case raw, ok := <-c.send:
			err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				logError(logs, "can not set write deadline", err)
				return
			}

			if !ok {
				// The hub closed the channel.
				err = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				logError(logs, "can not write close message", err)
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				logError(logs, "can not get next writer", err)
				return
			}

			msg, _ := json.Marshal(raw)
			_, err = w.Write(msg)
			logError(logs, "can not write message", err)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				_, err = w.Write([]byte{'\n'})
				logError(logs, "can not write newline message", err)

				raw = <-c.send
				msg, _ := json.Marshal(raw)
				_, err = w.Write(msg)
				logError(logs, "can not write message", err)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			logError(logs, "set write deadline", err)

			if err = c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logError(logs, "WriteMessage err", err)
				return
			}
		}
	}
}
