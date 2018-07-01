
package register

type Register interface{
	QueryMethod(methodSign string) ([]string,error)
	TrackingServices()
	RegisterMethod(methods []string,host string,weight int) error
}

var Reg Register