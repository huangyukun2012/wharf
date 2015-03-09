package main

import(
	"log"
	"net/http"
	"wharf/utils"
	"encoding/json"
	"errors"
//	"github.com/VividCortex/godaemon"
)

func BindIp2containerHandler(w http.ResponseWriter, r *http.Request){
	var buffer []byte
	var result  utils.BindResult
	buffer = make([]byte, 100)//we only store id and ip
	n, err := r.Body.Read(buffer)
	if err != nil{
		result.Succeed = false
		result.Warning = err.Error()
		data := json.Marshal(result)
		ioutil.WriteString(w, string(data) )
		return 
	}

	content := buffer[:n]
	var containerAndIp utils.Container2Ip
	jsonerr := json.Unmarshal(content, &containerAndIp)
	if jsonerr != nil{
		result.Succeed = false
		result.Warning = jsonerr.Error()
		data := json.Marshal(result)
		ioutil.WriteString(w, string(data) )
		return 
	}
	
	binderr := BindIp2Container(containerAndIp.Id, containerAndIp.Ip)
	if binderr != nil{
		result.Succeed = false
		result.Warning = binderr.Error()
		data := json.Marshal(result)
		ioutil.WriteString(w, string(data))	
		return 
	}
	
	result.Succeed=true
	data := json.Marshal(result)
	ioutil.WriteString(w, string(data))
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

func ShutDownHandler(w http.ResponseWriter, r *http.Request){
	log.Fatal("The client will be shut down!")
	errPrintln("The client will be shut down!")
	return 
}

func GetResourceHandler(w http.ResponseWriter, r *http.Request){
	senderr := SendInfo2Etcd(utils.UP)
	var content string
	if senderr != nil{
		content = "fail"	
	}else{
		content = "succeed"
	}
	io.WriteString(w, content)
}

func main(){
//	godaemon.MakeDaemon(&godaemon.DaemonAttr{})
	utils.Daemon(0,1)

	Init()
	//send one for service discovery
	SendInfo2Etcd(utils.ALIVE)

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
