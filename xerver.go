// xerver 3.0, a tiny and light transparent fastcgi reverse proxy,
// copyright 2016, (c) Mohammed Al Ashaal <http://www.alash3al.xyz>,
// published uner MIT licnese .
// -----------------------------
// *> available options
// >> --root        [only use xerver as static file server],            i.e "/var/www/" .
// >> --backend     [only use xerver as fastcgi reverse proxy],         i.e "[unix|tcp]:/var/run/php5-fpm.sock" .
// >> --controller  [the fastcgi process main file "SCRIPT_FILENAME"],  i.e "/var/www/main.php"
// >> --http        [the local http address to listen on],              i.e ":80"
// >> --https       [the local https address to listen on],             i.e ":443"
// >> --cert        [the ssl cert file path],                           i.e "/var/ssl/ssl.cert"
// >> --key         [the ssl key file path],                            i.e "/var/ssl/ssl.key"
// *> available internals
// >> Xerver-Internal-ServerTokens [off|on]
// >> Xerver-Internal-FileServer [file|directory]
// >> Xerver-Internal-ProxyPass [transparent-http-proxy]
package main

import "fmt"
import "log"
import "net"
import "flag"
import "net/url"
import "net/http"

import (
	"os"
	"os/signal"
	"syscall"
	"puppy/gateway"
	"puppy/config"
	"puppy/register"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// user vars
var (
	cfgfile *string = flag.String("f", "/Users/chris/gopath/src/puppy/pp.ini", "main config file")
)

func init() {
	flag.Parse()
	config.Init(cfgfile)
	fmt.Println("Socket in:  ", config.Instance.Listen)
	fmt.Println("Socket out: ", config.Instance.FCGI_PASS)
	fmt.Println("")
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// let's play :)
func main() {
	rcvr := func() {
		if err := recover(); err != nil {
			log.Println("err> ", err)
		}
	}
	// an error channel to catch any error
	err := make(chan error)

	httpSrv := (&gateway.CgiHandler{}).Init()
	rpcsrv := (&gateway.RpcService{}).Init()
	(&register.RedisRegister{}).Init()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)
	go (func(address string) {
		defer rcvr()
		uri, e := url.Parse(address);
		if e != nil {
			err <- e;
			return
		}

		srv := http.Server{Addr: uri.Host + uri.Path, Handler: httpSrv}
		l, e := net.Listen(uri.Scheme, srv.Addr)
		if e != nil {
			err <- e;
			return
		}
		go srv.Serve(l)

		go func(c chan os.Signal) {
			sig := <-c
			log.Printf("Caught signal %s: shutting down.", sig)
			l.Close()
			os.Exit(0)
		}(sigc)
	})(config.Instance.Listen)

	go func(host string, port string) {
		err <- rpcsrv.StartServer("tcp://0.0.0.0:" + config.Instance.RpcPort)
		//ret, _ := rpcsrv.CallRemote("tcp://127.0.0.1:9999", "lala", "123")
		//fmt.Print(ret)

	}(config.Instance.LocalIpAddr, config.Instance.RpcPort)

	go register.HeartBeat(func() []string {
		return httpSrv.RetriveServices(config.Instance.SrvHealthCheck)
	})

	log.Fatal(<-err)
}
