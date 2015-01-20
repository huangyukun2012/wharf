package master

import (
	"fmt"
	"encoding/json"
	"wharf/utils"
	"os"
	"io"
/*	"net"
	"strings"
	"bufio"
	"strconv"
	"time"
	*/
	"github.com/coreos/go-etcd/etcd"
)

type Configd struct{
	Etcdnode utils.Etcd
	IpPool utils.IpPool	
}

type Res struct {
	Node  utils.Machine
	Docker_nr 	[]int// container num running on each cpu
	Filter	[]bool
}


var Gres map[string]Res
var MasterConfig Configd

func main(){
 	err := GetMasterConfig()
	if err != nil{
		fmt.Println(err)	
		return 
	}

	Gres = make(map[string]Res, 1)

	key := MasterConfig.Etcdnode.Key
	machines := []string{`http://`+ MasterConfig.Etcdnode.Ip+":"+MasterConfig.Etcdnode.Port}

	_ = GetMachineResource(machines, key, false, false )
	return 
}

//get resource from the key/value database of etcd, update it for Gres
func GetMachineResource( endpoint []string, key string, sort, recursive bool)( error){
	client := etcd.NewClient(endpoint)	
	res, err := client.Get(key, sort, recursive)
	if err != nil {
		fmt.Println("get ", key, " failed: ", err)
		return err
	}

	ip_nr := res.Node.Nodes.Len()
	for i:= 0; i<ip_nr; i++{
		//skip the "/" to get ip
		key := res.Node.Nodes[i].Key[1:]
		value := res.Node.Nodes[i].Value

		fmt.Println(key, ":", value)	
		var machine_info utils.Machine 
		json.Unmarshal([]byte(value), &machine_info)
		_, found := Gres[key] 
		var temp Res
		temp.Node = machine_info
		if  found {
			temp.Docker_nr = Gres[key].Docker_nr	
		}
		//if notfound, the temp.Docker_nr will be zero
		Gres[key]= temp	
	}
	return nil
}

func GetMasterConfig() ( error ){
	filename := "/etc/wharf/configd"
	reader , err := os.Open(filename)	
	if err != nil{
		fmt.Println(filename, err)	
		return err
	}
	MasterConfig , err = UnmarshalConfigd(reader)	
	if err != nil {
		fmt.Println(err)	
		return err
	}
	return nil
}

//unmarshal configd 
func UnmarshalConfigd(reader io.Reader )( Configd, error){
	decoder := json.NewDecoder(reader)
	var res Configd					
	err := decoder.Decode(&res)
	return res, err
}
