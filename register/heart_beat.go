package register

import (
	"time"
	"puppy/config"
	"log"
	"fmt"
)

func HeartBeat(hcCall func() []string) {
	runReg := true
	for ; ; {
		log.Print("checking heart beat")
		if (runReg) {
			go func() {
				srvs := hcCall()
				if len(srvs) < 1 {
					return
				}
				Reg.RegisterMethod(srvs, fmt.Sprintf("tcp://%s:%d",config.Instance.LocalIpAddr,config.Instance.RpcPort), 1)
				log.Println(Reg.QueryMethod("ha@testSystem"))
			}()
		}
		go func() {
			Reg.TrackingServices()
		}()
		runReg=!runReg
		time.Sleep(time.Duration(config.Instance.RegisterInfoTTL/2) * time.Second)
	}
}
