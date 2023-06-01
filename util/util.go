package util

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var HeaderBlacklist = []string{"Cookie", "X-CSRFToken"}

// These headers aren't things that proxies should pass along. Some are forbidden by http2.
// This fixes the bug where Chrome users saw a ERR_SPDY_PROTOCOL_ERROR for all proxied requests.
func FilterHeaders(r *http.Response) error {
	badHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Connection",
		"Transfer-Encoding",
		"Upgrade",
		"Access-Control-Allow-Headers",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Origin",
		"Access-Control-Expose-Headers",
	}
	for _, h := range badHeaders {
		r.Header.Del(h)
	}
	return nil
}

func SingleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// decodeSubprotocol decodes the impersonation "headers" on a websocket.
// Subprotocols don't allow '=' or '/'
func DecodeSubprotocol(encodedProtocol string) (string, error) {
	encodedProtocol = strings.Replace(encodedProtocol, "_", "=", -1)
	encodedProtocol = strings.Replace(encodedProtocol, "-", "/", -1)
	decodedProtocol, err := base64.StdEncoding.DecodeString(encodedProtocol)
	return string(decodedProtocol), err
}

func CopyMsgs(writeMutex *sync.Mutex, dest, src *websocket.Conn) error {
	for {
		messageType, msg, err := src.ReadMessage()
		if err != nil {
			return err
		}

		if writeMutex == nil {
			err = dest.WriteMessage(messageType, msg)
		} else {
			writeMutex.Lock()
			err = dest.WriteMessage(messageType, msg)
			writeMutex.Unlock()
		}

		if err != nil {
			return err
		}
	}
}

func KeepAlive(writeMutex *sync.Mutex, dest *websocket.Conn) error {
	websocketTimeout := 30 * time.Second
	websocketPingInterval := 30 * time.Second
	ticker := time.NewTicker(websocketPingInterval)

	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			writeMutex.Lock()
			// Send pings to client to prevent load balancers and other middlemen from closing the connection early
			err := dest.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(websocketTimeout))
			writeMutex.Unlock()
			if err != nil {
				return err
			}
		}
	}
}

func CreateProxyHeaders(w http.ResponseWriter, r *http.Request) (http.Header, string, error) {
	subProtocol := ""
	proxiedHeader := make(http.Header, len(r.Header))
	for key, value := range r.Header {
		if key != "Sec-Websocket-Protocol" {
			// Do not proxy the subprotocol to the API server because k8s does not understand what we're sending
			proxiedHeader.Set(key, r.Header.Get(key))
			continue
		}

		for _, protocols := range value {
			for _, protocol := range strings.Split(protocols, ",") {
				protocol = strings.TrimSpace(protocol)
				// TODO: secure by stripping newlines & other invalid stuff
				// "Impersonate-User" and "Impersonate-Group" and bridge specific (not a k8s thing)
				if strings.HasPrefix(protocol, "Impersonate-User.") {
					encodedProtocol := strings.TrimPrefix(protocol, "Impersonate-User.")
					decodedProtocol, err := DecodeSubprotocol(encodedProtocol)
					if err != nil {
						errMsg := fmt.Sprintf("Error decoding Impersonate-User subprotocol: %v", err)
						http.Error(w, errMsg, http.StatusBadRequest)
						return nil, "", err
					}
					proxiedHeader.Set("Impersonate-User", decodedProtocol)
					subProtocol = protocol
				} else if strings.HasPrefix(protocol, "Impersonate-Group.") {
					encodedProtocol := strings.TrimPrefix(protocol, "Impersonate-Group.")
					decodedProtocol, err := DecodeSubprotocol(encodedProtocol)
					if err != nil {
						errMsg := fmt.Sprintf("Error decoding Impersonate-Group subprotocol: %v", err)
						http.Error(w, errMsg, http.StatusBadRequest)
						return nil, "", err
					}
					proxiedHeader.Set("Impersonate-User", string(decodedProtocol))
					proxiedHeader.Set("Impersonate-Group", string(decodedProtocol))
					subProtocol = protocol
				} else {
					proxiedHeader.Set("Sec-Websocket-Protocol", protocol)
					subProtocol = protocol
				}
			}
		}
	}

	// Filter websocket headers.
	websocketHeaders := []string{
		"Connection",
		"Sec-Websocket-Extensions",
		"Sec-Websocket-Key",
		// NOTE: kans - Sec-Websocket-Protocol must be proxied in the headers
		"Sec-Websocket-Version",
		"Upgrade",
	}
	for _, header := range websocketHeaders {
		proxiedHeader.Del(header)
	}

	return proxiedHeader, subProtocol, nil
}
