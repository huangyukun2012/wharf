package main

import(
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"wharf/util"
	"os/exec"
)

func BindIp2containerHandler(w http.ResponseWriter, r *http.Request){
	var result  util.HttpResponse
	if *flagD{
		util.PrintErr("BindIp2containerHandler():\n")	
	}
	var buffer []byte
	buffer = make([]byte, 100)//we only store id and ip
	n, err := r.Body.Read(buffer)
	if err != nil && err != io.EOF{
		result.Set(util.SERVER_ERROR, err.Error())
		io.WriteString(w, result.String())
		return 
	}

	content := buffer[:n]
	var containerAndIp util.Container2Ip
	jsonerr := json.Unmarshal(content, &containerAndIp)
	if jsonerr != nil{
		result.Set(util.SERVER_ERROR, jsonerr.Error())
		io.WriteString(w, result.String())
		return 
	}
	
	networkName, binderr := BindIp2Container(containerAndIp.Ip, containerAndIp.Id)
	if binderr != nil{
		result.Set(util.SERVER_ERROR, binderr.Error())
		io.WriteString(w, result.String())
		return 
	}else{
		result.Set(util.OK, string(networkName))
		io.WriteString(w, result.String())
		return 
	}
}

func BindIp2Container(ip , containerId string)([]byte,  error ){
	var networkName []byte 
	shortid := containerId[:12]
	if *flagD{
		util.PrintErr("BindIp2Container():ip-id", ip,"-----",  shortid, "\n")	
	}
	_, err0 := exec.LookPath("bindip.sh")
	if err0 != nil{
		errors.New(`You have not install 'bindip.sh' in your computer! Please install it first.`)
		return networkName, err0
	}
	cmd := exec.Command("/bin/bash","-c", " bindip.sh "+ ip+" "+shortid)
	if cmd==nil{
		return networkName, errors.New("Could not create command bindip.sh")
	}
	output , err := cmd.CombinedOutput()
	var res error
	if err != nil{
		res=errors.New("Fail to bind ip " + ip +" to container " + shortid+ ": "  + string(output))	
		return networkName, res
	}else{
		networkName = output
		return networkName, nil	
	}
}


//remember how the devname is conveyed.
func delDevHandler(w http.ResponseWriter, r *http.Request){
	var response util.HttpResponse	
	_, err := exec.LookPath("deldev.sh")
	if err != nil{
		util.PrintErr("Please check that you have installed deldev.sh in PATH")	
	}

	var devname string
	contentByte , err0 := util.ReadContentFromHttpRequest( r)
	if err0 != nil{
		response.Set(util.SERVER_ERROR, err0.Error())
		io.WriteString(w, response.String())
		return 
	}

	devname = string(contentByte)
	cmd := exec.Command("/bin/bash", "-c", "deldev.sh "+devname)
	res, err2 := cmd.CombinedOutput()
	if err2 != nil{
		response.Set( util.SERVER_ERROR, err2.Error())
			if *flagD {
				util.PrintErr(string(res))
				util.PrintErr(err2.Error())	
			}
	}else{
		response.Set( util.OK, "")
	}
	io.WriteString(w, response.String())
	return 
}

func startDocker()error{

	fmt.Println("Make sure:you have configed the net interface bridge br0, and we will start docker using this br0....")
	_, err0 := exec.LookPath("docker")	
	if err0 != nil{
		errors.New(`You have not install 'docker' in your computer! Please install it first.`)
	}

	//test if docker has started yet
	isDockerStartedCmd := exec.Command("pgrep", "docker")
	isDockerStarted,_ := isDockerStartedCmd.Output()

	if len(isDockerStarted)>0{
		//ddocker is stated yet.
			fmt.Println("Docker deamon is running.Please check it is running with br0.")		
			return nil
	}

	//docker is not stated, so we will start it
	cmd := exec.Command("docker", "-b", "br0",  "-d" , "-H", "unix:////var/run/docker.sock" , "-H" ,"0.0.0.0:4243")
	err := cmd.Start()
	var res error
	if err != nil{
		res= errors.New("Fail: can not start docker,"+err.Error())	
	}
	fmt.Println("docker is running with command", `docker -b=br0 -d -H unit///var/run/docker.sock -H 0.0.0.0:4243`)
	return res
}

func ShutDownHandler(w http.ResponseWriter, r *http.Request){
	log.Fatal("The client will be shut down!")
	util.PrintErr("The client will be shut down!")
	os.Exit(1)
	return 
}


func GetResourceHandler(w http.ResponseWriter, r *http.Request){
	if *flagD {
		fmt.Println("GetResourceHandler(): this function is called!")	
	}
	var res util.HttpResponse
	senderr := SendInfo2Etcd(util.UP)
	if senderr != nil{
		if *flagD {
			fmt.Println("GetResourceHandler(): SendInfo2Etcd() failed!")	
		}
		res.Set(util.SERVER_ERROR, senderr.Error())
	}else{
		if *flagD {
			fmt.Println("GetResourceHandler(): SendInfo2Etcd() succeed!")	
		}
		res.Set(util.OK, "nil")
	}
	io.WriteString(w, res.String())
}

func main(){
	flagd := flag.Bool("d", false, "run the resource as a daemon")	
	flagD = flag.Bool("D", false, "output the debuf info")	
	flag.Parse()	

	if *flagd {
		util.Daemon(0,1)
	}

	Init()
	//send one for service discovery
	senderr := SendInfo2Etcd(util.ALIVE)
	if senderr!= nil{
		return 
	}

	//go routine
	CollectLoop()

	http.HandleFunc("/get_resource", GetResourceHandler)
	http.HandleFunc("/shut_down", ShutDownHandler)
	http.HandleFunc("/bindip2container", BindIp2containerHandler)
	http.HandleFunc("/del_dev", delDevHandler)

	err := http.ListenAndServe(`:`+ClientConfig.Resource.Port, nil)
	if *flagD{
		util.PrintErr("Begin to provide service. . .")	
	}
	if err != nil{
		log.Fatal("ListenAndServe", err.Error())	
	}
}
