package proxy

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/kubevirt-ui/kubevirt-proxy-data/util"
)

type Config struct {
	Endpoint        *url.URL
	TLSClientConfig *tls.Config
	Origin          string
}

type Proxy struct {
	Config *Config
}

func (p *Proxy) createUpgrader(subProtocol string) websocket.Upgrader {
	upgrader := &websocket.Upgrader{
		Subprotocols: []string{subProtocol},
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header["Origin"]
			if p.Config.Origin == "" {
				log.Printf("CheckOrigin: Proxy has no configured Origin. Allowing origin %v to %v", origin, r.URL)
				return true
			}
			if len(origin) == 0 {
				log.Printf("CheckOrigin: No origin header. Denying request to %v", r.URL)
				return false
			}
			if p.Config.Origin == origin[0] {
				return true
			}
			log.Printf("CheckOrigin '%v' != '%v'", p.Config.Origin, origin[0])
			return false
		},
	}
	return *upgrader
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Block scripts from running in proxied content for browsers that support Content-Security-Policy.
	w.Header().Set("Content-Security-Policy", "sandbox;")
	// Add `X-Content-Security-Policy` for IE11 and older browsers.
	w.Header().Set("X-Content-Security-Policy", "sandbox;")

	for _, h := range util.HeaderBlacklist {
		r.Header.Del(h)
	}

	// Include `system:authenticated` when impersonating groups so that basic requests that all
	// users can run like self-subject access reviews work.
	if len(r.Header["Impersonate-Group"]) > 0 {
		r.Header.Add("Impersonate-Group", "system:authenticated")
	}

	r.Host = p.Config.Endpoint.Host
	r.URL.Host = p.Config.Endpoint.Host
	r.URL.Scheme = p.Config.Endpoint.Scheme

	if r.URL.Scheme == "https" {
		r.URL.Scheme = "wss"
	} else {
		r.URL.Scheme = "ws"
	}

	proxiedHeader, subProtocol, err := util.CreateProxyHeaders(w, r)

	if err != nil {
		log.Println("Proxy header failed to create: ", err)
	}

	// NOTE (ericchiang): K8s might not enforce this but websockets requests are
	// required to supply an origin.
	dialer := &websocket.Dialer{
		TLSClientConfig: p.Config.TLSClientConfig,
	}

	backend, resp, err := dialer.Dial(r.URL.String(), proxiedHeader)

	if err != nil {
		errMsg := fmt.Sprintf("Failed to dial backend: '%v'", err)
		statusCode := http.StatusBadGateway
		if resp == nil || resp.StatusCode == 0 {
			log.Println(errMsg)
		} else {
			statusCode = resp.StatusCode
			if resp.Request == nil {
				log.Printf("%s Status: '%v' (no request object)", errMsg, resp.Status)
			} else {
				log.Printf("%s Status: '%v' URL: '%v' , r.URL: %v", errMsg, resp.Status, resp.Request.URL, r.URL.String())
			}
		}
		http.Error(w, errMsg, statusCode)
		return
	}

	upgrader := p.createUpgrader(subProtocol)

	frontend, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade websocket to client: '%v'", err)
		return
	}

	var writeMutex sync.Mutex // Needed because ticker & copy are writing to frontend in separate goroutines

	defer func() {
		backend.Close()
		frontend.Close()
	}()

	errc := make(chan error, 2)

	// Can't just use io.Copy here since browsers care about frame headers.
	go func() { errc <- util.CopyMsgs(nil, frontend, backend) }()
	go func() { errc <- util.CopyMsgs(&writeMutex, backend, frontend) }()
	go func() { errc <- util.KeepAlive(&writeMutex, frontend) }()

	for {
		select {
		case <-errc:
			// Only wait for a single error and let the defers close both connections.
			return
		}
	}
}
