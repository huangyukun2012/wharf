package server 

import (
	/* "bufio" */
	"fmt"
	"errors"
	"os"
	"io"
	"log"
	"reflect"
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

var FlagDebug *bool
var MasterConfig util.Config
var LatestTask string
//==============task
const(
	TaskNumMax=6000
	RUNNING=0
	DOWN=1
)
var TasksIndex int32
type Task struct{
	Cmd		CreateRequest 
	Set		[]CalUnit
	Status	int	
	CreatedTime time.Time	
}


func (t *Task)Init(cmd CreateRequest, setnum int){
	t.Cmd=cmd
	t.Set= make([]CalUnit, setnum)
}

var Tasks map[string]Task//name -- task

type CalUnit struct{
	ContainerIp string
	ContainerDesc APIContainer
	HostIp string
	Nic		string	//virtual nic network name
//	HostMachine	util.Machine
	Docker_nr []int
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
	Warnings	 string
}
func (h *httpResponse)Init(){
	h.Succeed=false
}

type CreateResponse struct{
	Succeed  bool
	Warnings	 string
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
	starterr := util.StartDocker(MasterConfig.Docker.Bridge)
	if starterr!=nil{
		util.PrintErr(starterr)
		os.Exit(1)
	}
	Gres = make(map[string]Res, 1)
	Rres = make(map[string]Res, 1)
	Ares = make(map[string]Res, 1)
	Tasks = make(map[string]Task, 1)

	http.HandleFunc("/ps", ListTaskHandler)
	http.HandleFunc("/create", CreateTaskHandler)
	http.HandleFunc("/inspect", InspectTaskHandler)
	http.HandleFunc("/stop", StopTaskHandler)
	http.HandleFunc("/start", StartTaskHandler)
	http.HandleFunc("/rm", RmTaskHandler)


	errhttp := http.ListenAndServe(":"+MasterConfig.Server.Port, nil)
	if errhttp != nil{
		log.Fatal("InitServer: ListenAndServe ", errhttp)	
	}
}

/*Function: statr etcd server for wharf.
Carefull:
	1)if etcd is stared, we should not close it.
	2)May be etcd is stared, but not for wharf.*/

func initEtcd()error{
	fmt.Println("Etcd is starting...")	

	//test if etcd has started yet
	isEtcdStartedCmd := exec.Command("pgrep", "etcd")
	isEtcdStarted,_ := isEtcdStartedCmd.Output()
			

	if len(isEtcdStarted)>0{
		//etcd is started
		fmt.Println("Etcd is already running.Make sure it is for wharf.")		
		return nil
	}

	//etcd is not started
	_, existsErr := exec.LookPath("etcd")
	if existsErr != nil{
		return existsErr 
	}
	HOME := os.Getenv("HOME")
	path := HOME +`/.wharf.etcd`
	fmt.Println("etcd", "-name wharf -data-dir", path)
	cmd := exec.Command("etcd","-name", "wharf" ,"-data-dir", path)
	err := cmd.Start()
	if err !=nil{
		return err	
	}
	fmt.Println("Etcd is stared!")	
	return err
}

			

/*
function:
	start some tasks.
param:
	flags , task1, task2...
*/
func StartTaskHandler( w http.ResponseWriter, r *http.Request){
	var response util.HttpResponse
	inputInfo , err := GetInfoFromRequest(&w, r)
	if err != nil{
		return 
	}

	var userRequest StartRequest
	userRequest.Init(&inputInfo)
	for thisFlag, flagValue := range inputInfo.Data{
		switch thisFlag{
			case "h":
				userRequest.FsName["Help"], _=strconv.ParseBool(flagValue)
			default:
				response.Set("404-invalid input","invalid flag of wharf start.")
				io.WriteString(w, response.String())
				return 
		}	
	}	

	thisTask, err := checkTaskName(&w, r, userRequest.Args)
	if err != nil{
		return 
	}
	if thisTask.Status==RUNNING{
		response.Set(util.OK, "The task is already started.")
		io.WriteString(w, response.String())
		return 
	}

	//start each ct accoring to the task
	var isStarted bool
	isStarted = true
	var outputData StartOutput
	for i := range thisTask.Set {
		unit := thisTask.Set[i]		
		start2delErr := startContainerWithIP(&(thisTask.Set[i]))//hostip , contaienr id, container ip
		if start2delErr != nil{
			isStarted = false	
			outputData.Append(unit.ContainerDesc.Id, unit.HostIp, start2delErr.Error())
		}
	}

	if isStarted==true{
		thisTask.Status = RUNNING 
		delete(Tasks, userRequest.Args[0])
		Tasks[userRequest.Args[0]]=thisTask

		outputData.Warning="task "+userRequest.Args[0]+" is started."
		response.Set(util.OK, outputData.String())
		io.WriteString(w, response.String())
	}else{
		outputData.Warning="Failed to start "+ userRequest.Args[0]
		response.Set(util.SERVER_ERROR, outputData.String())
		io.WriteString(w, response.String())
	}	
	return 
}

/*
function:
	Rm some tasks.
param:
	flags , task1, task2...
*/
func RmTaskHandler( w http.ResponseWriter, r *http.Request){
	var response util.HttpResponse
	inputInfo , err := GetInfoFromRequest(&w, r)
	if err != nil{
		return 
	}

	//flag parse
	var userRequest RmRequest
	userRequest.Init(&inputInfo)
	for thisFlag, flagValue := range inputInfo.Data{
		switch thisFlag{
			case "h":
				userRequest.FsName["Help"], _=strconv.ParseBool(flagValue)
			default:
				response.Set(util.INVALID_INPUT, "invalid flag of wharf stop.")
				io.WriteString(w, response.String())
				return 
		}	
	}	

	thisTask, err := checkTaskName(&w, r, userRequest.Args)
	if err != nil{
		return 
	}

	if thisTask.Status!=DOWN{
		response.Set(util.INVALID_INPUT, "The task is not stoped yet.")
		io.WriteString(w, response.String())
		return 
	}

	var isRemoved bool
	isRemoved = true
	var outputData StopOutput
	for i := range thisTask.Set {
		unit := thisTask.Set[i]		
		rmErr := removeContainerOnHost(unit.ContainerDesc.Id, unit.HostIp)
		if rmErr != nil{
			isRemoved = false 
			outputData.Append(unit.ContainerDesc.Id, unit.HostIp, rmErr.Error())
		}
		clearTaskInGres(unit)
	}

	if isRemoved ==true{
		delete(Tasks, userRequest.Args[0])

		outputData.Warning="task "+userRequest.Args[0]+" is removed."
		response.Set(util.OK, outputData.String())
		io.WriteString(w, response.String())
	}else{
		outputData.Warning="Failed to remove"+ userRequest.Args[0]
		response.Set(util.SERVER_ERROR, outputData.String())
		io.WriteString(w, response.String())
	}
	return
}

/*function:
clear the resource occuppation in Gres according to the task
return :
	nil,  if no err
*/
func clearTaskInGres( unit CalUnit){
	thisRes , exists := Gres[unit.HostIp]	
	if !exists {
		return 
	}
	delete(Gres, unit.HostIp)	

	length := len(thisRes.Docker_nr)
	for i:=0 ; i< length; i++{
		if unit.Docker_nr[i]==1 {
			thisRes.Docker_nr[i]++	
		}	
	}
	Gres[unit.HostIp]=thisRes
}

/*
function:
	check if the user provides valid taskname.
return:
	
*/
func checkTaskName(w *http.ResponseWriter, r *http.Request, Args []string)(thisTask Task, err error){
	var response util.HttpResponse
	if len(Args)<1{
		response.Set(util.INVALID_INPUT, "You must provide one taskname to the command.")
		io.WriteString(*w, response.String())
		return	thisTask, errors.New("Taskname is not provided.") 
	}

	thistask, exists := Tasks[Args[0]]	
	if !exists{
		response.Set(util.INVALID_INPUT, "There is no task with the name of "+`'`+Args[0]+`'`)	
		io.WriteString(*w, response.String())
		return	thistask, errors.New("Taskname is invalid.") 
	}
	return thistask, nil
}
/*
function:
	stop some tasks.
param:
	flags , task1, task2...
return:
			
*/
func StopTaskHandler( w http.ResponseWriter, r *http.Request){
	var response util.HttpResponse
	inputInfo , err := GetInfoFromRequest(&w, r)
	if err != nil{
		return 
	}

	var userRequest StopRequest
	userRequest.Init(&inputInfo)
	for thisFlag, flagValue := range inputInfo.Data{
		switch thisFlag{
			case "h":
				userRequest.FsName["Help"], _=strconv.ParseBool(flagValue)
			default:
				response.Set(util.INVALID_INPUT, "invalid flag of wharf stop.")
				io.WriteString(w, response.String())
				return 
		}	
	}	

	thisTask, err := checkTaskName(&w, r, userRequest.Args)
	if err != nil{
		return 
	}

	if thisTask.Status==DOWN{
		response.Set(util.OK, "The task is already stoped.")
		io.WriteString(w, response.String())
		return 
	}

	var isStoped bool
	isStoped = true
	var outputData StopOutput
	for i := range thisTask.Set {
		unit := thisTask.Set[i]		
		stop2delErr := StopContainer2deleteDev(unit)
		if stop2delErr != nil{
			isStoped=false	
			outputData.Append(unit.ContainerDesc.Id, unit.HostIp, stop2delErr.Error())
		}
	}

	if isStoped==true{
		thisTask.Status = DOWN
		delete(Tasks, userRequest.Args[0])
		Tasks[userRequest.Args[0]]=thisTask

		outputData.Warning="task "+userRequest.Args[0]+" is stopped."
		response.Set(util.OK, outputData.String())
		io.WriteString(w, response.String())
	}else{
		outputData.Warning="Failed to stop "+ userRequest.Args[0]
		response.Set(util.SERVER_ERROR, outputData.String())
		io.WriteString(w, response.String())
	}
	return
}

/*
function:stop container on some host, and delete the device about the container.
	This is two http requesta.
return value:the error

Note: we do no need delete the device, for it will be deleted after the container stoped
*/
func StopContainer2deleteDev(unit CalUnit)error{
	stoperr := StopContainerOnHost(unit.ContainerDesc.Id, unit.HostIp)
	if stoperr != nil{
		return stoperr
	}
	return stoperr
}
/*insect the detail informatin about only one task.*/
func InspectTaskHandler( w http.ResponseWriter, r *http.Request){
	var response util.HttpResponse
	inputInfo , err := GetInfoFromRequest(&w, r)
	if err != nil{
		return 
	}

	var userRequest InspectReqest
	userRequest.Init()
	for thisFlag, flagValue := range inputInfo.Data{
		switch thisFlag{
			case "f":
				userRequest.Field=flagValue 
			default:
				response.Set(util.INVALID_INPUT, "invalid flag of wharf inspect.")
				io.WriteString(w, response.String())
				return 
		}	
	}	
	//must contain one taskname 
	if len(inputInfo.Args) != 1{
		response.Set(util.INVALID_INPUT, "You must provide one and only  taskname to insepect.")
		io.WriteString(w, response.String())
		return 
	}
	thistask, exists := Tasks[inputInfo.Args[0]]	
	if !exists{
		response.Set(util.INVALID_INPUT, "There is no task with the name of "+`'`+inputInfo.Args[0]+`'`)	
		io.WriteString(w, response.String())
		return 
	}
	
	var outputData InspectOutput 		
	outputData.GetData(thistask)
	if userRequest.Field==""{
		var outputByte []byte
		outputByte ,err = json.Marshal(outputData)	
		if err!=nil{
			response.Set(util.SERVER_ERROR, err.Error())
		}else{
			response.Set(util.OK, string(outputByte))
		}
	}else{
		immutable := reflect.ValueOf(outputData)                                                                                                                                                        
		fieldValue := userRequest.Field

		if fieldValue[0:1]==`.`{
			fieldValue = fieldValue[1:]
		}   
			
		itermsStr := strings.Split(fieldValue,".")

		for i:=range itermsStr{
			immutable = immutable.FieldByName(itermsStr[i])
		}   

		response.Set(util.OK,immutable.String())

	}
	io.WriteString(w,response.String())
	return 
}


/*List the information about one or more tasks.
The information is not in detai */
func ListTaskHandler(w http.ResponseWriter, r *http.Request){
	//the return value :response
	var response util.HttpResponse
	inputInfo , err := GetInfoFromRequest(&w, r)
	if err != nil{
		return 
	}
	var userRequest PsRequest
	userRequest.Init()
	for thisFlag, flagValue := range inputInfo.Data{
		switch thisFlag{
			case "a":
				userRequest.All, _= strconv.ParseBool(flagValue)
			case "l":
				userRequest.Latest,_=strconv.ParseBool(flagValue)
			case "n":
				userRequest.Name=flagValue
			case "i":
				userRequest.Image=flagValue
			case "t":
				userRequest.Type=flagValue
			default:
				response.Set(util.INVALID_INPUT, "invalid flag of wharf ps.")
				io.WriteString(w, response.String())
				return 
		}	
	}
	//flags conflict
	if userRequest.All && userRequest.Latest ||
		userRequest.All && userRequest.Name != "" ||
		userRequest.Latest && userRequest.Name != ""{
			response.Set(util.INVALID_INPUT, "flags are conflicting with each other.")
			io.WriteString(w, response.String())
			return 
	}
	
	var psIterm PsOutput
	//task name is set || latest is set
	if userRequest.Name != "" || userRequest.Latest{
		if userRequest.Latest{
			userRequest.Name=LatestTask
		}
		thisTask, exists := Tasks[userRequest.Name]	
		if !exists{
			response.Set(util.INVALID_INPUT, `No such task with name of "`+userRequest.Name+`"`)	
			io.WriteString(w, response.String())
			return 
		}else{
			fillPSOutpusWithTask(&psIterm, &thisTask)	
			data , _ := json.Marshal(psIterm)
			response.Set(util.OK, string(data))
			io.WriteString(w, response.String())
			return 
		}
	}
	//traverse all the task in Tasks
	for _,  thisTask := range Tasks{
		response.Status=util.OK
		if  (userRequest.All==true || thisTask.Status == RUNNING)&&
			(userRequest.Image=="" || thisTask.Cmd.ImageName == userRequest.Image)&&
			(userRequest.Type=="" || thisTask.Cmd.TypeName == userRequest.Type){
			fillPSOutpusWithTask(&psIterm, &thisTask)	
			data , _ := json.Marshal(psIterm)
			response.Append(string(data))
		}
		io.WriteString(w, response.String())
	}
	return 
}

/*Function:
get the information in http.Request to 'inputInfo'.
return value:
	1)succeed: the data was stored 'inputInfo'
	2)Fail:the error information is write into 'w'
*/
func GetInfoFromRequest(w *http.ResponseWriter, r *http.Request)(inputInfo util.SendCmd,err error){
	var response util.HttpResponse

	var input []byte
	input,err = util.ReadContentFromHttpRequest(r)
	if err != nil{
		response.Set(util.SERVER_ERROR, err.Error()+"in server Handler()")	
		io.WriteString(*w, response.String())
		return inputInfo, err
	}

	err = json.Unmarshal(input, &inputInfo)
	if err != nil{
		response.Set(util.SERVER_ERROR, err.Error()+"in Server TaskHandler()")	
		io.WriteString(*w, response.String())
		return inputInfo, err
	}
	return inputInfo, nil
}

/**/
func fillPSOutpusWithTask(psIterm *PsOutput, thisTask *Task){
		psIterm.TaskName = thisTask.Cmd.TaskName
		if thisTask.Status == DOWN{
			psIterm.Status = "DOWN" 
		}else{
			psIterm.Status = "RUNNING for "	+ time.Since(thisTask.CreatedTime).String()
		}
		psIterm.Type = thisTask.Cmd.TypeName
		psIterm.Image = thisTask.Cmd.ImageName
		psIterm.Size = len(thisTask.Set)
		psIterm.Cpus = thisTask.Cmd.TotalCpuNum
}

/*function:create task from the user command. 
	we will create the task and store it in Tasks
param:
	r: r.Body include the string of the user command
	w:	w.Body include the result of the create.response will be marshal to w.
*/
func CreateTaskHandler( w http.ResponseWriter, r *http.Request){
	var response util.HttpResponse 
	var thisTask Task
	response.Status= util.INVALID_INPUT

	var contents []byte
	contents = make([]byte, 1000)
	length, err := r.Body.Read(contents)
	if err != nil && err != io.EOF{
		fmt.Fprintf(os.Stderr, "CreateHandler: can not read from http resposeWriter\n")	
		fmt.Fprintf(os.Stderr, "%s", err)
		response.Warnings = []string{"CreateHandler: can not read from http resposeWriter\n" + err.Error()}
		io.WriteString(w, response.String())
		return 
	}
	var res util.SendCmd
	if *FlagDebug {
		fmt.Println("contents:", string(contents))
	}
	//make sure the char in contents should not include '\0'
	contentsRight := contents[:length]
	errunmarshal := json.Unmarshal(contentsRight, &res) 
	if errunmarshal != nil{
		fmt.Fprintf(os.Stderr, "CreateHandler: Unmarshal failed for contents: ")
		fmt.Fprintf(os.Stderr, "%s", errunmarshal)
		response.Warnings = []string{"CreateHandler: Unmarshal failed for contents: " + errunmarshal.Error()}
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
						response.Warnings =[]string{`the type of the task is not supported yet. Only "single" and "mpi" is supported.` }
						io.WriteString(w,response.String())
						return 
					}
				case strings.EqualFold(thisflag,"n") :	
					userRequest.TaskName = flagvalue
				case strings.EqualFold(thisflag,"s") :	
					userRequest.Stratergy=flagvalue
					if !strings.EqualFold(thisflag, "MEM") && !strings.EqualFold(thisflag, "COM"){
						response.Warnings = []string{`Only MEM and COM are valid for -s flag`}
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
						response.Warnings = []string{"CreateHandler" + openerr.Error()}
						io.WriteString(w, response.String())
						return 
					}	
					unmarshalerr := util.UnmarshalReader(readerme, &(userRequest.ResNode))
					if unmarshalerr != nil{
						response.Warnings = []string{unmarshalerr.Error()}	
						io.WriteString(w, response.String())
					}
			default:
				fmt.Fprintf(os.Stderr, "CreateHandler: %s flag invalid", thisflag)
				response.Warnings = []string{"CreateHandler: " + thisflag + "flag invalid"}
				io.WriteString(w, response.String())
				return 
			}
		}

		var err error
		endpoint := []string{"http://" + MasterConfig.EtcdNode.Ip +":" + MasterConfig.EtcdNode.Port}
		err = UpdateEtcdForUse(endpoint, MasterConfig.EtcdNode.Key, true, true)
		if err!= nil{
			if *FlagDebug {
				util.PrintErr("Failded to Etcd data!")
			}
			response.Warnings = []string{err.Error()}
			io.WriteString(w, response.String())
			return 
		}
		//Debug
		if *FlagDebug {
			util.PrintErr("Etcd data has been updated!")
		}

		err = UpdateGres()
		if err!= nil{
			response.Warnings =[]string{ err.Error()}
			io.WriteString(w, response.String())
			return 
		}
		//Debug
		if *FlagDebug {
			util.PrintErr("Gres has been updated!With ", len(Gres), " terms left.")
			if len(Gres)==0{
				util.PrintErr("Error: ",  "0 node can be used in Gres")
			}	
		}


		err = Filter(userRequest)
		//Debug
		if *FlagDebug {
			util.PrintErr("Rres has been Filtered!, with ", len(Rres), " terms left")
		}

		err = Allocate( userRequest )
		if err!= nil{
			io.WriteString(w, err.Error())	
			io.WriteString(w, response.String())
			return 
		}
		//Debug
		if *FlagDebug {
			util.PrintErr("Allocate complished for the create task! ", len(Ares), " containers will be created!")
		}

		thisTask.Init(userRequest,len(Ares) )

		err = ImageTransport(&thisTask)
		if err!= nil{
			io.WriteString(w, err.Error())	
			io.WriteString(w, response.String())
			return 
		}

		//create container,start it,  bind ip
		err = CreateContainer2Ares(&thisTask )
		if err!= nil{
				//Debug
				if *FlagDebug {
					util.PrintErr("CreateContainer2Ares failed:",  err.Error())
				}
			response.Set(util.SERVER_ERROR, err.Error())
			io.WriteString(w, response.String())
			return 
		}
		thisTask.CreatedTime = time.Now()
		thisTask.Status = RUNNING 
		Tasks[userRequest.TaskName]=thisTask
		LatestTask = userRequest.TaskName
		response.Set(util.OK,thisTask.Cmd.TaskName )
		io.WriteString(w, response.String())
		
	}
}

/*
Function:ImageTransport accoring to Ares
Pro:
	1.test the image exists,from all the Gres, get imageNodes and emptyNodes
	2.Partion -- from the imageNodes and emptyNodes, get array of imageTransportHead 
		
	3.Transport -- (parallel)transport for each imageTransportHead, post the result to array res
		3.1 save 
		3.2 transport 
		3.3 load	:parallel
		3.4 delete	:parallel
	4.return nil if all the res is true; else return err 
*/

func ImageTransport( thistask *Task)error{
	imageName := thistask.Cmd.ImageName
	if imageName==""{
		util.PrintErr("imageName should not be null")	
		os.Exit(1)
	}
	if *FlagDebug{
		util.PrintErr("[ ImageTransport ]")
	}
	var imageNodes []string	
	var emptyNodes []string
	
	exits,_:= searchImageOnHost(imageName, MasterConfig.Server.Ip)
	if exits{
		imageNodes=append(imageNodes,MasterConfig.Server.Ip)	
	}	

	for ip,_:= range Gres{
		exits,_:= searchImageOnHost(imageName, ip)
		if exits{
			imageNodes=append(imageNodes,ip)	
		}	
	}

	if len(imageNodes)==0{
		return errors.New("You do not have the image in the system")	
	}

	for ip, _:= range Ares{
		if *FlagDebug{
			util.PrintErr("search imageNodes on ip(imageName,ip):",imageName, ip)
		}
		exits,_:= searchImageOnHost(imageName, ip)
		if !exits{//no such image
			emptyNodes=append(emptyNodes,ip)	
		}	
	}
	
	transportHeads := partition(imageNodes, emptyNodes, imageName)
//transport
	var chs	[]chan error 
	setnum := len(transportHeads)
	
	if *FlagDebug{
		data ,_ := json.Marshal(transportHeads)
		util.PrintErr(setnum , "transportHead:\n ",string(data))
	}
	chs = make([]chan error, setnum)
	for i:=0;i<setnum;i++{
		chs[i]=make(chan error)
		go saveTransLoadDel(imageName, transportHeads[i], chs[i])
	}
	for i:=0;i<setnum;i++{
		value := <-chs[i]	
		if value!=nil{
			return value
		}
	}
	if *FlagDebug{
		util.PrintErr("[ ImageTransport end!]")
	}
	return nil
}

func saveTransLoadDel(imageName string, transportHead util.ImageTransportHeadAPI, ch chan error)error{
	if *FlagDebug{
		util.PrintErr("[ saveTransLoadDel ]")
	}
	err := SaveImageOnHost(imageName, transportHead.Server)
	if err != nil{
		ch <- err 
		return err	
	}
	transerr := TransportImageWithHead(transportHead)	
	if transerr != nil{
		ch <- transerr 
		return transerr	
	}
	load2delerr := Load2DelwithHead(transportHead)
	if load2delerr!=nil{
		ch <- load2delerr
		return load2delerr
	}
	ch <- nil 
	return nil
}

/*
carefull: in this function, image name >> tarfilename
*/
func partition(imageNodes[]string, emptyNodes []string, imageName string)[]util.ImageTransportHeadAPI{
	if *FlagDebug{
		util.PrintErr("[partition]")	
	}
	var transportHeads []util.ImageTransportHeadAPI
	var thisHead util.ImageTransportHeadAPI
	m:=len(imageNodes)
	n:=len(emptyNodes)
	fmt.Println("imagenodes, emptyNodes:",m, n)
	num:= n/m
	remain:=n%m
	var i int
	var emptyIndex int

	netIp := MasterConfig.Network.Net
	thisHead.Net = util.GetNetOfBIp(netIp.String())
	thisHead.DataIndex = 0
	thisHead.FileName= imageName+".tar"//make sure this filename

	for i=0;i<remain;i++{
		thisHead.Server = imageNodes[i]
		thisHead.Nodes = make([]string, num+1)
		for j:=0;j<num+1;j++{
			thisHead.Nodes[j] = util.GetHostOfBIp(emptyNodes[emptyIndex])
			emptyIndex++
		}
		transportHeads= append(transportHeads, thisHead)
	}

	for ;i<m&&num!=0;i++{
		thisHead.Server = imageNodes[i]
		thisHead.Nodes = make([]string, num)
		for j:=0;j<num;j++{
			thisHead.Nodes[j] = util.GetHostOfBIp(emptyNodes[emptyIndex])
			emptyIndex++
		}
		transportHeads= append(transportHeads, thisHead)
	}
	return  transportHeads
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
2. If the machine is in Gres, and the status is DOWN or alive, delete it from Gres.
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
		if *(FlagDebug){
			fmt.Println("GetMachineResource(): What we get in etcd is:")	
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
			temp.Docker_nr= make([]int, machine_info.CpuInfo.Num)
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
	if *(FlagDebug){
		fmt.Println("↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓	UpdateEtcdForUse(): What we get in etcd is:↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓" )	
	}
	for i:= 0; i<ip_nr; i++{
		//skip the "/" to get ip
		ip := res.Node.Nodes[i].Key[1:]
		machineValue := res.Node.Nodes[i].Value
		if *(FlagDebug){
			fmt.Println( ip, " : ", machineValue)	
		}

		var machine_info util.Machine 
		json.Unmarshal([]byte(machineValue), &machine_info)
		if machine_info.Status == util.DOWN{
			client.Delete(key, false)	
		}else{//not down
			getResourceUrl := `http://` + ip + ":"+ MasterConfig.Resource.Port+`/` +`get_resource`
			resp, err := http.Get(getResourceUrl)	
			
			var getSucceed bool//the default is false
			getSucceed = false
			if err == nil {
				defer resp.Body.Close()	
				body, _:= ioutil.ReadAll(resp.Body)
				var resp util.HttpResponse
				jsonerr := json.Unmarshal(body, &resp)
				if jsonerr != nil{
					if *FlagDebug {
						util.PrintErr("Function-UpdateEtcdForUse: json err")
						util.PrintErr("try to unmarshal data failed:", string(body))
					}
					return jsonerr	
				}
				if strings.HasPrefix(resp.Status, "200"){
					getSucceed =true 
				}else{
					getSucceed =false
				}
				if *(FlagDebug) && getSucceed{
					fmt.Println("UpdateEtcdForUse(): now the above info is updated")	
				}
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

	if *(FlagDebug){
		fmt.Println("↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑ UpdateEtcdForUse(): etcd data end. ↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑" )	
	}
	return nil
}

