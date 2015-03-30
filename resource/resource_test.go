//package get_machine_info 
/******************************************
1.get info of machine
2.process info
3.send info to etcd
*******************************************/

package main 

import (
	"testing"
	"wharf/util"

)

func TestSendInfo2Etcd(t *testing.T){
	err := SendInfo2Etcd(util.UP)
	if err !=nil{
		t.Errorf("SendInfo2Etcd failed!")	
		return 
	}
	t.Errorf("SendInfo2Etcd succeed, please check the data in etcd!")	
	
}
