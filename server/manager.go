package server 

import (
	"fmt"
	"encoding/json"
	"wharf/utils"
	"os"
	"io"
	"net/http"
	"log"
	"strings"
	"strconv"
	/* "net" */
	/*
	"bufio"
	"time"
	*/
	"github.com/coreos/go-etcd/etcd"
)
type Task struct{
	Cmdp	*CreateRequest 
	Set		[]CalUnit
}

type CalUnit struct{
	ContainerIp string
	ContainerDescp *APIContainer
	HostIp string
	HostMachinep	*utils.Machine
}

type Configd struct{
	Etcdnode utils.Etcd
	IpPool utils.IpPool	
	Service   Serve
	Docker	DockerService
	Resource Resource	
}

type DockerService struct{
	Port	string
}

type Resource struct{
	Port	string
}

type Serve struct{
	Ip  string 
	Port string
}

type Res struct {
	Node  utils.Machine
	Docker_nr 	[]int// container num running on each cpu
	/* Filter	[]bool */
}

//node-resouce for cluster
//for Gres, Docker_nr means the num of running containers
//		Rres----the left containers for each cpu
//		Ares----the num containers to run on each cpu
var Gres map[string]Res
var Rres map[string]Res
var Ares map[string]Res

var MasterConfig Configd


//Init  net server, etcd server
func InitServer(){

	Gres = make(map[string]Res, 1)
	Rres = make(map[string]Res, 1)
	http.HandleFunc("/list_task", ListTaskHandler)
	http.HandleFunc("/create", CreateHandler)

	errhttp := http.ListenAndServe(":"+MasterConfig.Service.Port, nil)
	if errhttp != nil{
		log.Fatal("InitServer: ListenAndServe ", errhttp)	
	}
	/* initNetwork() */	
	/* initEtcd() */
	/* initImageServer() */
}

func ListTaskHandler(w http.ResponseWriter, r *http.Request){
	content := r.Body
	fmt.Println(content)	
	return 
}

func CreateHandler( w http.ResponseWriter, r *http.Request){
	var contents []byte
	contents = make([]byte, 1000)
	length, err := r.Body.Read(contents)
	if err != nil && err != io.EOF{
		fmt.Fprintf(os.Stderr, "CreateHandler: can not read from http resposeWriter\n")	
		fmt.Fprintf(os.Stderr, "%s", err)
	}
	var res utils.SendCmd
	if *wharf.FlagDebug {
		fmt.Println("contents:", string(contents))
	}
	//make sure the char in contents should not include '\0'
	contentsRight := contents[:length]
	errunmarshal := json.Unmarshal(contentsRight, &res) 
	if errunmarshal != nil{
		fmt.Fprintf(os.Stderr, "CreateHandler: Unmarshal failed for contents: ")
		fmt.Fprintf(os.Stderr, "%s", errunmarshal)
	}else{
		//now we will create our task here: filter, allocator, (image server) ,scheduler
		var userRequest CreateRequest
		InitCreateRequest( &userRequest )//undefined
		for thisflag, flagvalue := range res.Data{
			switch {
				case strings.EqualFold(thisflag,"i") :	
					userRequest.ImageName = flagvalue
				case strings.EqualFold(thisflag,"t") :	
					userRequest.TypeName = flagvalue
					if  !strings.EqualFold(flagvalue,"mpi") && !strings.EqualFold(flagvalue,"single"){
						fmt.Fprintf(os.Stderr, `the type of the task is not supported yet. Only "single" and "mpi" is supported.`)	
						io.WriteString(w,`the type of the task is not supported yet. Only "single" and "mpi" is supported.`)
						return 
					}
				case strings.EqualFold(thisflag,"n") :	
					userRequest.TaskName = flagvalue
				case strings.EqualFold(thisflag,"s") :	
					if strings.EqualFold(flagvalue, "MEM"){
						userRequest.Stratergy = 2 
					}else if strings.EqualFold(flagvalue, "COM"){
						userRequest.Stratergy = 1 
					}else{
						io.WriteString(w,`Only MEM and COM are valid for -s flag`)
						return 
					}
				case strings.EqualFold(thisflag,"c") :	
					userRequest.TotalCpuNum, _ =  strconv.Atoi(flagvalue) 
				case strings.EqualFold(thisflag,"C") :	
				 	userRequest.ContainerNumMax, _ = strconv.Atoi(flagvalue)	
				case strings.EqualFold(thisflag,"l") :	
					userRequest.OverloadMax ,_ =strconv.ParseFloat(flagvalue,32)
				case strings.EqualFold(thisflag,"f") :	
					filename := flagvalue
					readerme,openerr := os.Open(filename)	
					if openerr != nil{
						fmt.Fprintf(os.Stderr, "CreateHandler:%s", openerr)	
					}	
					unmarshalerr = UnmarshalReader(readerme, &(userRequest.ResNode))
			default:
				fmt.Fprintf(os.Stderr, "CreateHandler: %s flag invalid", flag)
			}
		}
		var err error
		errUpdateEtcd := UpdateEtcdForUse()
		if errUpdateEtcd != nil{
			io.WriteString(w, errUpdateEtcd.Error())	
			return 
		}

		errUpdateGres := UpdateGres()
		if errUpdateGres != nil{
			io.WriteString(w, errUpdateGres.Error())	
			return 
		}

		err = Filter(userRequest)

		err = Allocate()
		if err!= nil{
			io.WriteString(w, err.Error())	
			return 
		}

		err = ImageTransport()
		if err!= nil{
			io.WriteString(w, err.Error())	
			return 
		}
		
		//create container, bind ip
		err = CreateContainer2Ares()
		if err!= nil{
			io.WriteString(w, err.Error())	
			return 
		}

	}
}

//update Gres from etcd server
func UpdateGres() (error){
	key := MasterConfig.Etcdnode.Key
	machines := []string{`http://`+ MasterConfig.Etcdnode.Ip+":"+MasterConfig.Etcdnode.Port}

	err := GetMachineResource(machines, key, false, false )
	return err
}

/*get resource from the key/value database of etcd, update it for Gres
loop all the machine in etcd server:
1. If the machine was not in Gres, just add it
2. If the machine is in Gres, and the status is DOWN, delete it from Gres.
						----, and the status is UP,just update Gres. 
*/
func GetMachineResource( endpoint []string, key string, sort, recursive bool)( error){
	client := etcd.NewClient(endpoint)	
	res, err := client.Get(key, sort, recursive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get %s failed -- %s", key, err)
		return err
	}

	ip_nr := res.Node.Nodes.Len()
	for i:= 0; i<ip_nr; i++{
		//skip the "/" to get ip
		ip:= res.Node.Nodes[i].Key[1:]
		machineValue:= res.Node.Nodes[i].Value
		if *(wharf.FlagDebug){
			fmt.Println("What we get in etcd is:")	
			fmt.Println(ip, ":", machineValue)	
		}

		var machine_info utils.Machine 
		json.Unmarshal([]byte(machineValue), &machine_info)
		_, found := Gres[ip] 
		var temp Res
		temp.Node = machine_info
		if  found {
			if  machine_info.Status != utils.UP{
				delete(Gres,key)	
			}else{
				temp.Docker_nr = Gres[ip].Docker_nr	
				Gres[ip]= temp	
			} 
		}else{
			//if notfound, this node is a new one. the temp.Docker_nr will be zero
			Gres[ip]= temp	
		}
	}
	return nil
}

/* Before we create a task, we should update the contents in etcd:
1.Get the contents for each node in endpoint of etcd
2.If the status of the node is Down, Delete it from etcd
						----is not Down:Update it through module resource
										2.1 if Update succeed, set data, reset Failtime, set Status to UP
										2.2 if Update Failed, Failtime++
																		if Failtime >= MaxFailTime, status= down
																		if Failtime < MaxFailTime, status = alive
*/
func UpdateEtcdForUse( endpoint []string, key string, sort, recursive bool)( error){
	client := etcd.NewClient(endpoint)	
	res, err := client.Get(key, sort, recursive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get %s failed -- %s", key, err)
		return err
	}

	ip_nr := res.Node.Nodes.Len()
	for i:= 0; i<ip_nr; i++{
		//skip the "/" to get ip
		ip := res.Node.Nodes[i].Key[1:]
		machineValue := res.Node.Nodes[i].Value
		if *(wharf.FlagDebug){
			fmt.Println("What we get in etcd is:", ip, ":", machineValue)	
		}

		var machine_info utils.Machine 
		json.Unmarshal([]byte(machineValue), &machine_info)
		if machine_info.Status == utils.DOWN{
			client.Delete(key, false)	
		}else{//not down
			getResourceUrl := `http://` + ip +`/` +`get_resource`
			resp, err := http.Get(getResourceUrl)	
			var getSucceed bool//the default is false
			if err == nil{
				defer resp.Body.Close()	
				body, err := ioutil.ReadAll(resp.Body)
				bodystring := string(body)
				if strings.EqualFold(bodystring, "succeed"){
					getSucceed = true 
				}
			}
			if !getSucceed {
				machine_info.FailTime++
				if machine_info.FailTime >= utils.MaxFailTime{
					machine_info.Status = utils.DOWN	
				}else{
					machine_info.Status = utils.ALIVE
				}
				machineValue, errmarshal= json.Marshal(machine_info)
				if errmarshal != nil{
					fmt.Fprintf(os.Stderr, "UpdateEtcdForUse: marshal machine_info failed--%s", errmarshal)
					return errmarshal	
				}else{
					_, seterr := client.Set(ip,machineValue,0)
					if seterr != nil{
						fmt.Fprintf(os.Stderr, "UpdateEtcdForUse: can not update etcd from serve--", seterr)		
						return seterr
					}
				}
			}
		} 
	}
	return nil
}

//get config info from /etc/wharf/configd to MasterConfig
func GetMasterConfig() ( error ){
	filename := "/etc/wharf/configd"
	reader , err := os.Open(filename)	
	if err != nil{
		fmt.Println(filename, err)	
		return err
	}

	MasterConfig , err = UnmarshalConfigd(reader)	

	return err 
}

//unmarshal configd from Reader  
func UnmarshalConfigd(reader io.Reader )( Configd, error){
	decoder := json.NewDecoder(reader)
	var res Configd					
	err := decoder.Decode(&res)
	return res, err
}

func UnmarshalReader( res interface{}, reader io.Reader)(  error){
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&res)
	return err
}
