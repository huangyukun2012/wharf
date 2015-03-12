package server 

import (
	"bufio"
	"fmt"
	"os"
	"io"
	"log"
	"strings"
	"strconv"
	"time"
	"encoding/json"
	"github.com/coreos/go-etcd/etcd"
	"io/ioutil"
	"net/http"
	"os/exec"
	"wharf/util"
)

var flagDebug *bool
var MasterConfig util.Config
//==============task
const(
	TaskNumMax=6000
	RUN=0
	DOWN=1
)
var TasksIndex int32
type Task struct{
	Cmd		CreateRequest 
	Set		[]CalUnit
	Status	int32	
	CreatedTime time.Time	
}
func (t *Task)Init(cmd CreateRequest, setnum int){
	t.Cmd=cmd
	t.Set= make([]CalUnit, setnum)
}

var Tasks map[string]Task

type CalUnit struct{
	ContainerIp string
	ContainerDesc APIContainer
	HostIp string
	HostMachine	util.Machine
}


//==============res
type Res struct {
	Node  util.Machine
	Docker_nr 	[]int// container num running on each cpu
}

//node-resouce for cluster
//for Gres, Docker_nr means the num of running containers
//		Rres----the left containers for each cpu
//		Ares----the num containers to run on each cpu
var Gres map[string]Res
var Rres map[string]Res
var Ares map[string]Res

//=============response
type httpResponse struct{
	Succeed  bool
	Warning	 string
}
func (h *httpResponse)Init(){
	h.Succeed=false
}

type CreateResponse struct{
	Succeed  bool
	Warning	 string
}

func  (c *CreateResponse)String() string{
	if c==nil{
		return "nil"
	}
	data ,_:= json.Marshal(*c)
	return string(data)
}

//Init  net server, etcd server
func InitServer(){
	err := initEtcd()
	if err != nil{
		util.PrintErr(err.Error())
		os.Exit(1)
	}
	initNetwork()	
	Gres = make(map[string]Res, 1)
	Rres = make(map[string]Res, 1)
	Tasks = make(map[string]Task, 1)

	http.HandleFunc("/list_task", ListTaskHandler)
	http.HandleFunc("/create_task", CreateTaskHandler)
	http.HandleFunc("/transport_image", TransportImageHandler)

	errhttp := http.ListenAndServe(":"+MasterConfig.Server.Port, nil)
	if errhttp != nil{
		log.Fatal("InitServer: ListenAndServe ", errhttp)	
	}
}


func initEtcd()error{
	fmt.Println("Etcd is starting...")	

	//test if etcd has started yet
	isEtcdStartedCmd := exec.Command("pgrep", "etcd")
	isEtcdStarted,_ := isEtcdStartedCmd.Output()
	reader := bufio.NewReader(os.Stdin)

	if len(isEtcdStarted)>0{
		//etcd is started
		for{
			fmt.Println("Etcd is running.Do you want to restart it?(y/n)")		
			input, _ := reader.ReadBytes('\n')
			if input[0]!='y'{
				stopEtcdCmd:= exec.Command("pkill", "etcd")
				stoperr := stopEtcdCmd.Run()
				if stoperr!=nil{
					return stoperr
				}
				break
			}else if input[0]!='n'{
				return nil
			}
		}
	}
	//etcd is not started


	_, existsErr := exec.LookPath("etcd")
	if existsErr != nil{
		return existsErr 
	}
	HOME := os.Getenv("HOME")
	path := HOME +`/.wharf.etcd`
	cmd := exec.Command("etcd", "-data-dir="+path )
	err := cmd.Start()
	fmt.Println("Etcd is stared!")	
	return err
}

			

func TransportImageHandler( w http.ResponseWriter, r *http.Request){

}

func ListTaskHandler(w http.ResponseWriter, r *http.Request){
	return 
}

/*function:create task from the user command. 
	we will create the task and store it in Tasks
param:
	r: r.Body include the string of the user command
	w:	w.Body include the result of the create.response will be marshal to w.
*/
func CreateTaskHandler( w http.ResponseWriter, r *http.Request){
	var response CreateResponse
	var thisTask Task
	response.Succeed = false

	var contents []byte
	contents = make([]byte, 1000)
	length, err := r.Body.Read(contents)
	if err != nil && err != io.EOF{
		fmt.Fprintf(os.Stderr, "CreateHandler: can not read from http resposeWriter\n")	
		fmt.Fprintf(os.Stderr, "%s", err)
		response.Warning = "CreateHandler: can not read from http resposeWriter\n" + err.Error()
		io.WriteString(w, response.String())
		return 
	}
	var res util.SendCmd
	if *flagDebug {
		fmt.Println("contents:", string(contents))
	}
	//make sure the char in contents should not include '\0'
	contentsRight := contents[:length]
	errunmarshal := json.Unmarshal(contentsRight, &res) 
	if errunmarshal != nil{
		fmt.Fprintf(os.Stderr, "CreateHandler: Unmarshal failed for contents: ")
		fmt.Fprintf(os.Stderr, "%s", errunmarshal)
		response.Warning = "CreateHandler: Unmarshal failed for contents: " + errunmarshal.Error()
		io.WriteString(w, response.String())
		return 
	}else{
		//now we will create our task here: filter, allocator, (image server) ,scheduler
		var userRequest CreateRequest
		userRequest.Init()
		for thisflag, flagvalue := range res.Data{
			switch {
				case strings.EqualFold(thisflag,"i") :	
					userRequest.ImageName = flagvalue
				case strings.EqualFold(thisflag,"t") :	
					userRequest.TypeName = flagvalue
					if  !strings.EqualFold(flagvalue,"mpi") && !strings.EqualFold(flagvalue,"single"){
						fmt.Fprintf(os.Stderr, `the type of the task is not supported yet. Only "single" and "mpi" is supported.`)	
						response.Warning =`the type of the task is not supported yet. Only "single" and "mpi" is supported.` 
						io.WriteString(w,response.String())
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
						response.Warning = `Only MEM and COM are valid for -s flag`
						io.WriteString(w,response.String())
						return 
					}
				case strings.EqualFold(thisflag,"c") :	
					userRequest.TotalCpuNum, _ =  strconv.Atoi(flagvalue) 
				case strings.EqualFold(thisflag,"C") :	
				 	userRequest.ContainerNumMax, _ = strconv.Atoi(flagvalue)	
				case strings.EqualFold(thisflag,"l") :	
					value, _ := strconv.ParseFloat(flagvalue,32) 
					userRequest.OverLoadMax = float32(value)
				case strings.EqualFold(thisflag,"f") :	
					filename := flagvalue
					readerme,openerr := os.Open(filename)	
					if openerr != nil{
						fmt.Fprintf(os.Stderr, "CreateHandler:%s", openerr)	
						response.Warning = "CreateHandler" + openerr.Error() 
						io.WriteString(w, response.String())
						return 
					}	
					unmarshalerr := util.UnmarshalReader(readerme, &(userRequest.ResNode))
					if unmarshalerr != nil{
						response.Warning = unmarshalerr.Error()	
						io.WriteString(w, response.String())
					}
			default:
				fmt.Fprintf(os.Stderr, "CreateHandler: %s flag invalid", thisflag)
				response.Warning = "CreateHandler: " + thisflag + "flag invalid"
				io.WriteString(w, response.String())
				return 
			}
		}

		var err error
		endpoint := []string{"http://" + MasterConfig.EtcdNode.Ip +":" + MasterConfig.EtcdNode.Port}
		err = UpdateEtcdForUse(endpoint, MasterConfig.EtcdNode.Key, true, true)
		if err!= nil{
			response.Warning = err.Error()
			io.WriteString(w, response.String())
			return 
		}

		err = UpdateGres()
		if err!= nil{
			response.Warning = err.Error()
			io.WriteString(w, response.String())
			return 
		}

		err = Filter(userRequest)

		err = Allocate( userRequest )
		if err!= nil{
			io.WriteString(w, err.Error())	
			io.WriteString(w, response.String())
			return 
		}

		err = ImageTransport()
		if err!= nil{
			io.WriteString(w, err.Error())	
			io.WriteString(w, response.String())
			return 
		}
		
		//create container,start it,  bind ip
		thisTask.Init(userRequest,len(Ares) )
		err = CreateContainer2Ares(&thisTask )
		if err!= nil{
			io.WriteString(w, err.Error())	
			io.WriteString(w, response.String())
			return 
		}
		thisTask.CreatedTime = time.Now()
		Tasks[userRequest.TaskName]=thisTask
	}
}

func ImageTransport()error{
	return nil
}

//update Gres from etcd server
func UpdateGres() (error){
	key := MasterConfig.EtcdNode.Key
	machines := []string{`http://`+ MasterConfig.EtcdNode.Ip+":"+MasterConfig.EtcdNode.Port}

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
		if *(flagDebug){
			fmt.Println("What we get in etcd is:")	
			fmt.Println(ip, ":", machineValue)	
		}

		var machine_info util.Machine 
		json.Unmarshal([]byte(machineValue), &machine_info)
		_, found := Gres[ip] 
		var temp Res
		temp.Node = machine_info
		if  found {
			if  machine_info.Status != util.UP{
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
		if *(flagDebug){
			fmt.Println("What we get in etcd is:", ip, ":", machineValue)	
		}

		var machine_info util.Machine 
		json.Unmarshal([]byte(machineValue), &machine_info)
		if machine_info.Status == util.DOWN{
			client.Delete(key, false)	
		}else{//not down
			getResourceUrl := `http://` + ip +`/` +`get_resource`
			resp, err := http.Get(getResourceUrl)	
			var getSucceed bool//the default is false
			getSucceed = false
			if err == nil{
				defer resp.Body.Close()	
				body, _:= ioutil.ReadAll(resp.Body)
				var resp util.HttpResponse
				jsonerr := json.Unmarshal(body, &resp)
				if jsonerr != nil{
					return jsonerr	
				}
				getSucceed = resp.Succeed 
			}
			if !getSucceed {
				machine_info.FailTime++
				if machine_info.FailTime >= util.MaxFailTime{
					machine_info.Status = util.DOWN	
				}else{
					machine_info.Status = util.ALIVE
				}

				machineValueBytes, errmarshal := json.Marshal(machine_info)
				machineValue=string(machineValueBytes)
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

