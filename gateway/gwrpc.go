package gateway

import (
	"net/rpc"
	"net"
	"net/url"
	"net/http"
)

type RpcServicable interface{CallRemote(address string, method string, payload string) (string, error) }

var RpcSrv RpcServicable

type RpcProxy struct {

}

type RpcArgs struct {
	Method  string
	Payload string
}
type Resp struct{
	Code int
	Resp string
}
func (r *Resp) Assign( code int,content string){
	r.Code=code
	r.Resp=content
}

func (r *RpcProxy) Invoke(arg *RpcArgs, resp *Resp) error {
	if(HttpSrv==nil){
		panic("HttpService not set!")
	}
	ret,e:=HttpSrv.CallHttp("","")
	resp.Resp=ret.Resp
	resp.Code=ret.Code
	return e
}

type RpcService struct {
	//Service
}

func (r *RpcService) StartServer(address string) error {
	uri, e := url.Parse(address);
	if (e != nil) {
		return e
	}
	p := &RpcProxy{}
	rpc.Register(p)
	rpc.HandleHTTP()
	l, e := net.Listen(uri.Scheme, uri.Host+uri.Path)
	if (e != nil) {
		return e
	}
	return http.Serve(l, nil)
}

func (r *RpcService) CallRemote(address string, method string, payload string) (string, error) {
	uri, e := url.Parse(address);
	if (e != nil) {
		return "", e
	}
	client, e := rpc.DialHTTP(uri.Scheme, uri.Host+uri.Path+":"+uri.Port())
	if (e != nil) {
		return "", e
	}
	var ret Resp;
	e = client.Call("RpcProxy.Invoke", &RpcArgs{method, payload}, &ret)

	if (e != nil) {
		panic(e)
		return "", e
	}
	return ret.Resp, nil
}


func (r *RpcService) Init() *RpcService{

	RpcSrv=r
	return r;
}