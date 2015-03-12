package main 

import(
	"fmt"
	"os"
	/* "wharf/utils" */
	"wharf/server"
)


func main(){
	err := server.MasterConfig.Init()
	if err != nil{
		fmt.Fprintf(os.Stderr, "%s:%s", "main", err)	
		return 
	}

	commandRegAndParse()

	//run sub command
	if *flagDaemon == true{
//		utils.Daemon(0,1)
		server.InitServer()	
	}else{
		//run sub command
		Run()
	}
}

func showVersion() {
	
}
