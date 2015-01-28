package version 

import (
	"fmt"
)
var (
	VERSION="1.0"
	GITCOMMIT string

	DOCKERAPI string
	DOCKERCLIENTAPI string

)

func ShowVersion(){
	fmt.Println(VERSION)
}
