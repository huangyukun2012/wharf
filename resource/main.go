package main

import(
	"bufio"
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
	var buffer []byte
	var result  util.BindResult
	buffer = make([]byte, 100)//we only store id and ip
	n, err := r.Body.Read(buffer)
	if err != nil{
		result.Succeed = false
		result.Warning = err.Error()
		data ,_:= json.Marshal(result)
		io.WriteString(w, string(data) )
		return 
	}

	content := buffer[:n]
	var containerAndIp util.Container2Ip
	jsonerr := json.Unmarshal(content, &containerAndIp)
	if jsonerr != nil{
		result.Succeed = false
		result.Warning = jsonerr.Error()
		data ,_:= json.Marshal(result)
		io.WriteString(w, string(data) )
		return 
	}
	
	binderr := BindIp2Container(containerAndIp.Id, containerAndIp.Ip)
	if binderr != nil{
		result.Succeed = false
		result.Warning = binderr.Error()
		data,_ := json.Marshal(result)
		io.WriteString(w, string(data))	
		return 
	}
	
	result.Succeed=true
	data,_ := json.Marshal(result)
	io.WriteString(w, string(data))
	return 
}

func BindIp2Container(ip , containerId string)(error ){
	_, err0 := exec.LookPath("bindip.sh")
	if err0 != nil{
		errors.New(`You have not install 'bindip.sh' in your computer! Please install it first.`)
	}
	cmd := exec.Command("bindip.sh", ip, containerId )
	err := cmd.Run()
	var res error
	if err != nil{
		res=errors.New("Fail to bind ip " + ip +" to container " + containerId)	
	}
	return res
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
	var restart bool
	restart = false
	reader := bufio.NewReader(os.Stdin)

	if len(isDockerStarted)>0{
		for{
			fmt.Println("Docker deamon is running.Do you want to restart it?(y/n)")		
			input, _ := reader.ReadBytes('\n')
			if input[0]!='y'{
				restart=true
				break
			}else if input[0]!='n'{
				restart=false	
				break
			}
		}
	}

	if !restart{
		return nil	
	}
	stopDockerCmd := exec.Command("pkill", "docker")
	stoperr := stopDockerCmd.Run()
	if stoperr!=nil{
		return stoperr
	}

	cmd := exec.Command("docker", "-b=br0",  "-d" , "-H unix:////var/run/docker.sock" , "-H 0.0.0.0:4243")
	err := cmd.Run()
	var res error
	if err != nil{
		res= errors.New("Fail: can not start docker,"+err.Error())	
	}
	fmt.Println("docker has started with bridge br0.")
	return res
}

func ShutDownHandler(w http.ResponseWriter, r *http.Request){
	log.Fatal("The client will be shut down!")
	util.PrintErr("The client will be shut down!")
	return 
}

func GetResourceHandler(w http.ResponseWriter, r *http.Request){
	var res util.HttpResponse
	senderr := SendInfo2Etcd(util.UP)
	if senderr != nil{
		res.Set(false, senderr.Error())
	}else{
		res.Set(true, "nil")
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

	err := http.ListenAndServe(ClientConfig.Resource.Port, nil)
	if err != nil{
		log.Fatal("ListenAndServe", err.Error())	
	}

}
