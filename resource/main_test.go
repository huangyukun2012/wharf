package main 
import (
	"os/exec"
	"errors"
	"testing"
)

func TestBindIp2Container(t *testing.T){
	ip := "192.168.122.9"
	containerId := "4291"
	_, err0 := exec.LookPath("bindip.sh")
	if err0 != nil{
		t.Error(`You have not install 'bindip.sh' in your computer! Please install it first.`)
	}
	cmd := exec.Command("/bin/bash", "-c", "bindip.sh "+ip+ " "+containerId )
	output,  err := cmd.CombinedOutput()
	var res error
	if err != nil{
		res=errors.New("Fail to bind ip " + ip +" to container " + containerId + ": "  + string(output))	
	}

	if res!=nil{
		t.Error(res.Error())	
	}
}
