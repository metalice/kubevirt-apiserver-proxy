package handlers

import (
	"crypto/tls"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubevirt-ui/kubevirt-apiserver-proxy/proxy"
	"github.com/kubevirt-ui/kubevirt-apiserver-proxy/util"
)

var API_SERVER_URL string = "kubernetes.default.svc"
var PROTOCOL string = "https"
var ORIGIN = "http://localhost"

func HealthHandler(c *gin.Context) {
	defer c.Request.Body.Close()
	c.String(200, "OK")
}

func RequestHandler(c *gin.Context) {
	if c.Request.URL.Scheme == "" {
		c.Request.URL.Scheme = "https"
	}

	if c.Request.URL.Host == "" {
		c.Request.URL.Host = API_SERVER_URL
	}

	tlsConf := &tls.Config{InsecureSkipVerify: true}

	proxy := &proxy.Proxy{
		Config: &proxy.Config{
			TLSClientConfig: tlsConf,
			Endpoint:        c.Request.URL,
			Origin:          ORIGIN,
		},
	}

	c.Request.Header.Set("Origin", ORIGIN)
	c.Request.Header.Set("Accept-Encoding", "*")
	defer c.Request.Body.Close()

	if c.IsWebsocket() {

		proxy.ServeHTTP(c.Writer, c.Request)

	} else {

		tr := &http.Transport{
			TLSClientConfig: tlsConf, // TODO: add a check for PROD / DEV mode
		}

		cr := func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}

		httpClient := http.Client{Transport: tr, CheckRedirect: cr}

		c.Request.RequestURI = ""
		resp, err := httpClient.Do(c.Request)
		if err != nil {
			log.Println("Failed to initiate call to kube api server: ", err.Error())
		}

		bodyBytes, err := io.ReadAll(resp.Body)

		if err != nil {
			log.Println("Failed to read response body: ", err.Error())
		}

		defer resp.Body.Close()
		filteredJson := util.FilterResponseQuery(bodyBytes, c.Request.URL.Query())

		if err != nil {
			log.Println("Unable to transform response body to json) ", err.Error())
		}

		c.JSON(resp.StatusCode, filteredJson)
	}
}
