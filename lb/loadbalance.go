package lb

var Instance LoadBalance

type  LoadBalance interface{
	Select(string,[]string) string
}