package server
import(
	"testing"
	"fmt"
)

func Test_searchImage(t *testing.T){
	fmt.Println("ubuntu:latest on 122.10")
	have,_:= searchImageOnHost(`ubuntu:latest`, `192.168.122.10`)
	if !have{
		fmt.Println("do not found the iamge")
	}else{
		fmt.Println("found this image")	
	}

	fmt.Println("ubuntu:latest on 122.1")
	haves,_:= searchImageOnHost(`ubuntu:latest`, `192.168.122.1`)
	if haves{
		fmt.Println("do not found the iamge")
	}else{
		fmt.Println("found this image")	
	}
	return 
}
/*
func Test_rmImage(t *testing.T){
	err := removeImageOnHost(`ubuntu:latest`, `192.168.122.1`)
	if err!=nil{
		t.Error(err)	
	}else{
		fmt.Println("remove this image")	
	}
	return 

}
*/
