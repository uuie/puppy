package config

import (
	"github.com/go-ini/ini"
	"reflect"
	"strings"
	"strconv"
)

type FcgiParam struct {
	SCRIPT_FILENAME   string
	QUERY_STRING      string //$query_string;
	REQUEST_METHOD    string //$request_method;
	CONTENT_TYPE      string //$content_type;
	CONTENT_LENGTH    string //$content_length;
	SCRIPT_NAME       string //$fastcgi_script_name;
	REQUEST_URI       string //$request_uri;
	DOCUMENT_URI      string //$document_uri;
	DOCUMENT_ROOT     string //$document_root;
	SERVER_PROTOCOL   string //$server_protocol;
	REQUEST_SCHEME    string //$scheme;
	HTTPS             string //$https if_not_empty;
	GATEWAY_INTERFACE string //CGI/1.1;
	SERVER_SOFTWARE   string //nginx/$nginx_version;
	REMOTE_ADDR       string //$remote_addr;
	REMOTE_PORT       string //$remote_port;
	SERVER_ADDR       string //$server_addr;
	SERVER_PORT       string //$server_port;
	SERVER_NAME       string //$server_name;
	REDIRECT_STATUS   int    //200;
}

type RegisterCfg struct{
	Host string
	Port int
}
type Config struct {
	Listen           string
	SO_TIMEOUT       int
	FCGI_SPLIT_PATH  string // ^(.+\.php)(/.+)$;
	DOCUMENT_ROOT    string
	FCGI_PASS        string // unix:/var/run/php/php7.1-fpm.sock;
	FCGI_INDEX       string // /index.php;
	Params           FcgiParam
	InboundIpPattern string
	RpcPort          string
	SrvHealthCheck   string
	LocalIpAddr      string
	EnableFileLookup	bool
	RegisterInfoTTL int
	Register         RegisterCfg
}

var Instance Config

func Init(cfgfile *string) {
	file, _ := ini.Load(*cfgfile)
	file.MapTo(&Instance)
	Instance.LocalIpAddr = getLocalIp()
	env := make(map[string]string)
	v := reflect.ValueOf(Instance)
	t := reflect.TypeOf(Instance)
	for i := 0; i < v.NumField(); i++ {
		name := t.Field(i).Name
		value := v.Field(i).String()
		env["$"+strings.ToLower(name)] = value
	}
	p := &(Instance.Params)
	v = reflect.ValueOf(p).Elem()
	for i := 0; i < v.NumField(); i++ {
		value := v.Field(i).String()
		for key, ev := range env {
			value = strings.Replace(value, key, ev, -1)
		}
		switch v.Field(i).Kind() {
		case reflect.String:
			v.Field(i).SetString(value)
		case reflect.Int:
			ival, _ := strconv.Atoi(value)
			v.Field(i).SetInt(int64(ival))
		}

	}
}
