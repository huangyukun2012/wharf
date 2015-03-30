package main 

import(
	"fmt"
	"os"
	"wharf/util"
	"wharf/server"
)


func main(){
	err := server.MasterConfig.Init()
	if err != nil{
		fmt.Fprintf(os.Stderr, "%s:%s", "main", err)	
		fmt.Fprintf(os.Stderr, "Please check if the config file /etc/wharf/wharf.conf is correct.")	
		return 
	}

	commandRegAndParse()
	server.FlagDebug = FlagDebug
	//run sub command
	if matchingCmd==nil{
		server.InitServer()	
		if *flagDaemon == true{
			util.Daemon(0,1)
		}
	}else{
		//run sub command
		Run()
	}
}

