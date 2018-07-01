package gateway

import (
	"net/http"
	//import "net/http/httputil"
	//import "github.com/tomasen/fcgi_client"
	"net/url"
	"log"
	"github.com/tomasen/fcgi_client"
	"net"
	"fmt"
	"strings"
	"strconv"
	"puppy/register"
	"puppy/config"
	"encoding/json"
	"io/ioutil"
	"os"
	"puppy/lb"
)

var(
	 HttpSrv HttpServicable
)
func init(){

}
type HttpServicable interface {
	CallHttp(method string, payload string) (*Resp, error)
}


type JsonService struct {
	SystemId  string   `json:"system_id"`
	Endpoints []string `json:"endpoints"`
}

type CgiHandler struct {
	http.Handler
	HttpServicable
	//http.ResponseWriter
	Proto string
	Addr  string
}

func (c *CgiHandler) Init() *CgiHandler {
	url, e := url.Parse(config.Instance.FCGI_PASS)
	if (e != nil) {
		panic(e)
	}
	c.Proto = url.Scheme
	c.Addr = url.Path + url.Host
	HttpSrv = c
	return c;
}

/**
in request, invoke rpc call
 */
func (c *CgiHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	method := req.Header.Get("Remote-Method-Key")

	if providers ,err:= register.Reg.QueryMethod(method); len(providers) < 1 || err!=nil {
		c.renderError(res, "method no provider", method,err)
		return
	}
	payload := req.URL.Query().Encode()
	hosts,err:=register.Reg.QueryMethod(method)
	if(err!=nil || len(hosts)<1){
		c.renderError(res,"error finding host",err)
		return
	}
	host:=lb.Instance.Select(method,hosts)
	RpcSrv.CallRemote(host, method, payload)
}
func (c *CgiHandler) renderError(writer http.ResponseWriter, msg ...interface{}) {

}

/**
out request , in
 */
func (c *CgiHandler) CallHttp(method string, payload string) (*Resp, error) {

	req := http.Request{
		Host:       "127.0.0.1",
		RemoteAddr: "",
		//URL:url.ParseRequestURI(""),
		Method: "POST",
		Proto:  "HTTP/1.1",
	}
	req.URL, _ = url.Parse("http://127.0.0.1/" + method)
	req.URL.RawQuery = payload

	resp := &Resp{}
	c.ServeFCGI(resp, &req)
	//ServeFCGI
	return resp, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (c *CgiHandler) ServeFCGI(res *Resp, req *http.Request) {

	// connect to the fastcgi backend,
	// and check whether there is an error or not .
	fcgi, err := fcgiclient.Dial(c.Proto, c.Addr)
	if err != nil {
		log.Println(err)
		res.Assign(502,"Bad gateway")
		return
	}
	// automatically close the fastcgi connection and the requested body at the end .
	defer fcgi.Close()
	//defer req.Body.Close()
	// prepare some vars :
	// -- http[addr, port]
	// -- https[addr, port]
	// -- remote[addr, host, port]
	// -- edit the request path
	// -- environment variables
	http_addr, http_port, _ := net.SplitHostPort(req.Host)
	//https_addr, https_port, _ := net.SplitHostPort(*HTTPS)
	remote_addr, remote_port, _ := net.SplitHostPort(req.RemoteAddr)
	req.URL.Path = req.URL.ResolveReference(req.URL).Path

	var script string
	if(config.Instance.EnableFileLookup){
	script := config.Instance.DOCUMENT_ROOT + req.URL.Path
		if _, err := os.Stat(script); err != nil && os.IsNotExist(err) { //file not exists
			script = config.Instance.Params.SCRIPT_NAME
		}
	}else{
		script = config.Instance.Params.SCRIPT_NAME
	}
	env := map[string]string{
		"DOCUMENT_ROOT":   config.Instance.DOCUMENT_ROOT,
		"SCRIPT_FILENAME": script,
		"REQUEST_METHOD":  req.Method,
		"REQUEST_URI":     req.URL.RequestURI(),
		"REQUEST_PATH":    req.URL.Path,
		"PATH_INFO":       req.URL.Path,
		"CONTENT_LENGTH":  fmt.Sprintf("%d", req.ContentLength),
		"CONTENT_TYPE":    req.Header.Get("Content-Type"),
		"REMOTE_ADDR":     remote_addr,
		"REMOTE_PORT":     remote_port,
		"REMOTE_HOST":     remote_addr,
		"QUERY_STRING":    req.URL.Query().Encode(),
		//"SERVER_SOFTWARE": VERSION,
		"SERVER_NAME":     req.Host,
		"SERVER_ADDR":     http_addr,
		"SERVER_PORT":     http_port,
		"SERVER_PROTOCOL": req.Proto,
		"FCGI_PROTOCOL":   c.Proto,
		"FCGI_ADDR":       c.Addr,
		"HTTPS":           "",
		"HTTP_HOST":       req.Host,
	}

	// iterate over request headers and append them to the environment varibales in the valid format .
	for k, v := range req.Header {
		env["HTTP_"+strings.Replace(strings.ToUpper(k), "-", "_", -1)] = strings.Join(v, ";")
	}
	// fethcing the response from the fastcgi backend,
	// and check for errors .
	resp, err := fcgi.Request(env, req.Body)
	if err != nil {
		log.Println("err> ", err.Error())
		res.Assign(502,err.Error())
		return
	}
	// parse the fastcgi status .
	resp.Status = resp.Header.Get("Status")
	resp.StatusCode, _ = strconv.Atoi(strings.Split(resp.Status, " ")[0])
	if resp.StatusCode < 100 {
		resp.StatusCode = 200
	}

	// automatically close the fastcgi response body at the end .
	defer resp.Body.Close()
	content,err:=ioutil.ReadAll(resp.Body)
	if(err==nil){
		res.Assign(200,string(content))
		return
	}
	res.Assign(500,err.Error())
}

func (c *CgiHandler) RetriveServices(uri string) []string {
	resp, err := c.CallHttp(config.Instance.SrvHealthCheck, "")
	if (err != nil || resp.Code != 200) {
		log.Println("Error retrive services", err, resp.Code, resp.Resp)
		return []string{}
	}
	info := &JsonService{}
	err = json.Unmarshal([]byte(resp.Resp), info)
	if (err != nil) {
		log.Println("Error decode response json : "+resp.Resp,err)
		return []string{}
	}
	srvs := make([]string, len(info.Endpoints))
	for idx, serviceId := range info.Endpoints {
		srvs[idx] = fmt.Sprintf("%s@%s", serviceId, info.SystemId)
	}
	return srvs
}
