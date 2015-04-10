package server
import(
	"testing"
	"fmt"
)

func Test_searchImage(t *testing.T){
	err := searchImageOnHost(`ubuntu:latest`, `192.168.122.10`)
	if err!=nil{
		t.Error(err)	
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
