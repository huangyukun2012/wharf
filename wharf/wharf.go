package main 

import(
	"fmt"
	"os"
	/* "wharf/utils" */
	"wharf/server"
)


func main(){
 	err := server.GetMasterConfig()
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
		/* res, err := Run() */
		/* if err != nil{ */
		/* 	fmt.Fprintf(os.Stderr, "%s", err) */
		/* }else{ */
		/* 	fmt.Println(string(res)) */				
		/* } */
	}
}

func showVersion() {
	
}
