//The implement of Docker API on Client and other http request.
package server 

import(
	"errors"
	"strings"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"wharf/util"
)

/*=========== containers================*/

type APIContainer struct{
	Id string
	Image string
	Command string
	Created int64
	Status	string
	Ports []APIPort
	SizeRW	int64
	SizeRootFs	int64
	Names	[]string
}

type APIPort struct{
	PrivatePort	int64
	PublicPort	int64
	Type		string
	IP			string
}

type CreateContainerOptions struct{
	Hostname string
	Domainname string
	User string
	Memory int32
	CpuShares int32
	Cpuset string
	AttachStdin bool 
	AttachStdout bool 
	AttachStderr bool 
	Tty bool
	OpenStdin bool
	StdinOnce bool
	Cmd []string
	Image string
	WorkingDir string
	NetworkDisabled bool
}


func (opts *CreateContainerOptions)Init(){
		opts.AttachStdin = true
		opts.AttachStdout = true
		opts.AttachStderr = true
		opts.Tty = true
		opts.OpenStdin =true
		opts.StdinOnce = false
		opts.NetworkDisabled = true

}

type CreateContainerReturn struct{
	Id string
	Warnings []string
}



/*This func will create container in endpoint according to opts, and return value of Container
para:	
	endpoint:	ip:port
*/
func CreateContainer(endpoint string, opts CreateContainerOptions)( *CreateContainerReturn , error){
	if *FlagDebug{
		util.PrintErr("[ CreateContainer ]")
	}
	path := "/containers/create?"
	url := "http://" + endpoint + path
	contentType := "application/json"
	data, _:= json.Marshal(opts)
	res, err := http.Post(url, contentType, strings.NewReader(string(data)))
	
    if err != nil{
		return nil, err
    }else if!strings.HasPrefix(res.Status,"201") {
		return nil, errors.New(res.Status)
	}else {
        defer res.Body.Close()
        contents , _:= ioutil.ReadAll(res.Body)
		var returnContainer CreateContainerReturn
		unmarshalerr := json.Unmarshal(contents, &returnContainer)
		if unmarshalerr != nil{
			return nil, unmarshalerr	
		}else{
			return &returnContainer, nil
		}
    }   
}

/*This func will inspect container in endpoint according to opts, and return value of Container
para:	
	endpoint:	ip:port
*/
func InspectContainer(endpoint string, opts InspectReq)( ct Container, err error){
	if *FlagDebug{
		util.PrintErr("[ InspectContainer ]")
	}

	path := "/containers/"+opts.Id+"/json"
	url := "http://" + endpoint + path
	res, err := http.Get(url)
	
    if err != nil{
		return ct, err
    }else{
		if strings.HasPrefix(res.Status, "404"){
			return ct, errors.New(res.Status)	
		}else if strings.HasPrefix(res.Status, "200"){
			defer res.Body.Close()

			contents , _:= ioutil.ReadAll(res.Body)
			unmarshalerr := json.Unmarshal(contents, &ct)
			if unmarshalerr != nil{
				return ct, unmarshalerr	
			}else{
				return ct, nil
			}
			
		}else{
			return ct, errors.New(res.Status)	
		}
    }   
}

/*function: create container according to Ares
 fill the task struct for the taskname
 and bind ip to containers, and start them
 param:
 	*Ares:global param
	tasknamep: the task to be filled--containerDesc.Id, hostIp, HostMachine 
 return value:
	nil, if succeed; error , if fail.
*/
func CreateContainer2Ares( tasknamep *Task) error{
	if *FlagDebug{
		util.PrintErr("[ CreateContainer2Ares ]")
	}
	var index int
	for ip, resData := range Ares{
		tasknamep.Set[index].HostIp = ip
		/* tasknamep.Set[index].HostMachine = resData.Node */ 
		thisContainerIp, getiperr :=  GetFreeIP()
		if getiperr != nil{
				return getiperr
		}
		tasknamep.Set[index].ContainerIp = thisContainerIp.String() 
		var opts CreateContainerOptions
		opts.Init()
		opts.Hostname = thisContainerIp.String()//we set the container hostname as its ip 
		//modifiy: if Docker_nr all = 0, the node will be filter out
		opts.Cpuset =  util.GetNozeroIndex( resData.Docker_nr )
		opts.Image = tasknamep.Cmd.ImageName

		endpoint := ip + ":" + MasterConfig.Docker.Port 	
		thisContainer, createerr := CreateContainer(endpoint, opts)
		if createerr != nil{
			return createerr	
		}else{
			tasknamep.Set[index].ContainerDesc.Id = thisContainer.Id	
			tasknamep.Set[index].Docker_nr = resData.Docker_nr
		}
		starterr := StartContainerOnHost(thisContainer.Id, ip)
		if starterr != nil{
			return starterr	
		}
		if *FlagDebug{
			util.PrintErr("container ", thisContainer.Id, "is started on host", ip, "\n")	
		}

		networkName, binderr:= BindIpWithContainerOnHost(thisContainerIp.String(), thisContainer.Id, ip)
		if binderr != nil{
			return binderr	
		}
		tasknamep.Set[index].Nic = networkName		
		index++
	}
	return nil
}

/*function:
	start a container with id of "id" ont host "hostIP".
	This will give out a http request to the docker deamon on "HostIP".

 return value:
	Docker remote API Status codes:
	204--no error
	304--container already started
	404--no such container
	500--server error

 	nil, when no error
	*/
func StartContainerOnHost( id , hostIp string)error{
	if *FlagDebug{
		util.PrintErr("[ StartContainerOnHost ]")	
	}	
	endpoint := "http://" + hostIp +":" +MasterConfig.Docker.Port 
	path := `/containers/` + id +`/start`

	res, err := http.Post(endpoint+path ,util.POSTTYPE, strings.NewReader("") )

	if err != nil{
		return err
	}
	//err == nil
	if strings.HasPrefix(res.Status, "204"){
		return nil	
	}else{
		return errors.New(res.Status)	
	}
}

/*function:
	stop a container with id of "id" ont host "hostIP".
	This will give out a http request to the docker deamon on "HostIP".
 return value:
 	nil, when no error
	*/
func StopContainerOnHost( id , hostIp string)error{
	if *FlagDebug{
		util.PrintErr("[ StopContainerOnHost]")	
	}	
	endpoint := "http://" + hostIp +":" +MasterConfig.Docker.Port 
	path := `/containers/` + id +`/stop?t=1`

	res, err := http.Post(endpoint+path ,util.POSTTYPE, strings.NewReader(""))

	if err != nil{
		return err
	}
	//err == nil
	if strings.HasPrefix(res.Status, "204"){
		return nil	
	}else{
		return errors.New(res.Status+" from docker daemon")	
	}
}

/*function:
	remove a container with id of "id" ont host "hostIP".
 return value:
 	nil, when no error
*/
func removeContainerOnHost (id, hostIp string )error{
	if *FlagDebug{
		util.PrintErr("[ removeContainerOnHost ]")
	}
	endpoint := "http://" + hostIp +":" +MasterConfig.Docker.Port 
	path := `/containers/` + id +`?v=1`

	request , err := http.NewRequest("DELETE", endpoint+path, strings.NewReader(""))
	if err != nil{
		util.PrintErr("http.NewRequest: ", err.Error())
		return errors.New("http.NewRequest: "+err.Error())
	}

	resp, err := http.DefaultClient.Do(request)
	if resp != nil{
		defer resp.Body.Close()	
	}
	if err != nil{
		util.PrintErr("http.DefaultClient.Do ", err.Error())	
		return errors.New("http.DefaultClient.Do "+ err.Error())
	}

	//err == nil
	if !strings.HasPrefix(resp.Status, "204"){
		util.PrintErr("resp.Status do not has prefix of 200 --", resp.Status)
		return errors.New("resp.Status do not has prefix of 200 --"+resp.Status)
	}else{
		return nil	
	}
}

func deleteDevice(hostIp, nicName string)(error){
	if *FlagDebug{
		util.PrintErr("[ deleteDevice ]")
	}
	var response util.HttpResponse
	endpoint := "http://" + hostIp +":" +MasterConfig.Resource.Port 
	path := `/del_dev`

	res , err := http.Post(endpoint+path, util.POSTTYPE, strings.NewReader(nicName))

	if err!= nil{
		return err	
	}

	if strings.HasPrefix(res.Status, "200"){
		err = util.ReadContentFromHttpResponse(res, &response)	
		if err != nil{
			return err	
		}
		if strings.HasPrefix(response.Status, "200"){
			return nil	
		}else{
			return errors.New(response.Status)
		}
	}else{
		return errors.New(res.Status)	
	}
	
}

/*
function:start a container according to a CalUnit.
*/
func startContainerWithIP(unit *CalUnit)(error){
//hostip , contaienr id, container ip
	if *FlagDebug{
		util.PrintErr("[ startContainerWithIP ]")
	}
	containerId := unit.ContainerDesc.Id
	containerIp := unit.ContainerIp
	hostIp:= unit.HostIp

	starterr := StartContainerOnHost(containerId, hostIp)
	if starterr != nil{
		return starterr	
	}else{
		devname , binderr := BindIpWithContainerOnHost(containerIp, containerId, hostIp)
		if binderr == nil{
			unit.Nic = devname	
		}
		return binderr
	}
}



/*function: bind a ip to a container in a host.
	This is a http request posted to the "docker server". Actrually, it is handler by module of resource.
	param:	
		ip: the ip to be bind.
		id: the id of the container
		hostip:the hostip
	return value:
		networkName :the virtual device of the eth
*/
func BindIpWithContainerOnHost(containerIp string, id string , hostIp string )(string, error ){
	if *FlagDebug{
		util.PrintErr("[ BindIpWithContainerOnHost ]")
	}
	var networkName string
	port := MasterConfig.Resource.Port	
	endpoint := "http://" + hostIp + ":" + port
	url := endpoint+`/bindip2container`

	var ctn_ip util.Container2Ip
	ctn_ip = util.Container2Ip{id, containerIp}
	data, jsonerr := json.Marshal(ctn_ip)
	if jsonerr != nil{
		return networkName, jsonerr
	}
	res, err := http.Post(url, util.POSTTYPE, strings.NewReader(string(data)) )
	if err != nil{
		return networkName, err
	}
	//err == nil
	defer res.Body.Close()
	data , _= ioutil.ReadAll(res.Body)
	var result util.HttpResponse
	jsonerr = json.Unmarshal(data, &result)
	if jsonerr != nil{
		return networkName, jsonerr	
	}else{
		var err error
		if strings.HasPrefix(result.Status, "200"){
			networkName = result.Warnings[0]
			err = nil
		}else{
			err = errors.New(result.Warnings[0])	
		}
		return networkName,err
	}
}

/*===========================image=======================*/
/*search if IMAGE is available on hostIP*/
func  searchImageOnHost(nameOrId, hostIp string)(bool, error ){
	port := MasterConfig.Docker.Port
	if port==string(""){
		port="4243"	
	}
	endpoint := hostIp+":"+port
	path := `/images`
	url := `http://`+endpoint+path+`/`+nameOrId+`/`+`json`

	if *FlagDebug{
		util.PrintErr("[ searchImageOnHost ]")
		util.PrintErr("  ",url)
	}
	resp,err := http.Get(url) 
	if err!=nil{
		return false,err
	}else if !strings.HasPrefix(resp.Status, "200"){
		return false, errors.New(resp.Status)	
	}else{
		return true,nil
	}
}


func removeImageOnHost (nameOrId, hostIp string )error{
	if *FlagDebug{
		util.PrintErr("[ removeImageOnHost ]")
	}
	endpoint := "http://" + hostIp +":" +MasterConfig.Docker.Port 
	path := `/images/` + nameOrId 

	request , err := http.NewRequest("DELETE", endpoint+path, strings.NewReader(""))
	if err != nil{
		util.PrintErr("http.NewRequest: ", err.Error())
		return errors.New("http.NewRequest: "+err.Error())
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil{
		util.PrintErr("http.DefaultClient.Do ", err.Error())	
		return errors.New("http.DefaultClient.Do "+ err.Error())
	}
	if resp != nil{
		defer resp.Body.Close()	
	}

	//err == nil
	if !strings.HasPrefix(resp.Status, "200"){
		return errors.New(resp.Status)
	}else{
		return nil	
	}
}

func SaveImageOnHost(nameOrId, hostIp string)(error){
	if *FlagDebug{
		util.PrintErr("[ SaveImageOnHost ]")
		util.PrintErr("		on ", hostIp)
	}

    var info util.Image2TarAPI 
    info = util.Image2TarAPI{Image:nameOrId, TarFileName: nameOrId+`.tar`}
    data , _:= json.Marshal(info)                                                                                                                                                                                   
    var url string
     url = `http://`+hostIp + `:` +MasterConfig.Image.Port+`/save_image` 
    resp, err := http.Post(url, util.POSTTYPE, strings.NewReader(string(data)))
    if err != nil{
		res :=  errors.New("SaveImageOnHost: "+ err.Error())
		util.PrintErr(res)
		return res
	}else if !strings.HasPrefix(resp.Status,"200"){
		res := errors.New("SaveImageOnHost: "+resp.Status)
		util.PrintErr(res)
		return res
    }else{
    	return nil
    }
}

func TransportImageWithHead(info util.ImageTransportHeadAPI)error{
	if *FlagDebug{
		util.PrintErr("[ TransportImagewithHead ]")
	}
                                                                                                                                                                                                                    
    data , _:= json.Marshal(info)
    url := `http://`+info.Server + `:` +MasterConfig.Image.Port+`/transport_image`
    resp, err := http.Post(url, util.POSTTYPE, strings.NewReader(string(data)))
    if err != nil{
		return err
    }else if !strings.HasPrefix(resp.Status, "200"){
		return errors.New("TransportImagewithHead: "+resp.Status)
    }else{
		return nil	
	}
}

func Load2DelwithHead(info util.ImageTransportHeadAPI)error{
	if *FlagDebug{
		util.PrintErr("[ Load2DelwithHead ]")
	}
	tarFileName := info.FileName	
	err := RmTarImageOnHost(tarFileName, info.Server)
	if err != nil{
		return err	
	}
	var chs []chan error
	chs = make([]chan error, len(info.Nodes))
	for index := range info.Nodes{
		chs[index] = make(chan error)
		hostIp := info.Net+"."+info.Nodes[index]
		go load2del(tarFileName, hostIp, chs[index])
	}

	for i:=0;i<len(info.Nodes);i++{
		value := <-chs[i]	
		if value!=nil{
			return value	
		}
	}
	return nil
}

func load2del(tarFileName string , hostIp string, ch chan error){
	if *FlagDebug{
		util.PrintErr("[ load2del ]")
	}
	loaderr := LoadTarOnHost(tarFileName, hostIp)	
	if loaderr!=nil{
		ch <- loaderr
		return 
	}
	delerr := RmTarImageOnHost(tarFileName, hostIp)	
	if delerr!=nil{
		ch <- delerr
		return 
	}
	ch <-nil
	return 
}

func  LoadTarOnHost(tarFileName, hostIp string)error{

	url := `http://`+hostIp+ `:` +MasterConfig.Image.Port+`/load_image`
	if *FlagDebug{
		util.PrintErr("[ LoadTarOnHost ]")
		util.PrintErr(url)
	}

	resp, err := http.Post(url, util.POSTTYPE, strings.NewReader(tarFileName))
	if err != nil{
		res := errors.New("Load tar on host Failed:"+err.Error())
		util.PrintErr(res)
		return res
	}else if !strings.HasPrefix(resp.Status,"200"){
		res := errors.New("Load tar on host Failed:"+resp.Status)
		util.PrintErr(res)
		return res
	}else{
		return  nil
	}
}

func  RmTarImageOnHost(tarFileName, hostIp string)error{
	if *FlagDebug{
		util.PrintErr("[ RmTarImageOnHost ]")
	}

	url := `http://`+hostIp+ `:` +MasterConfig.Image.Port+`/rm_tarfile`
	resp, err := http.Post(url, util.POSTTYPE, strings.NewReader(tarFileName))
	if err != nil{
		return errors.New("Delete tar on host Failed:"+err.Error())
	}else if !strings.HasPrefix(resp.Status,"200"){
		return  errors.New("Delete tar on host Failed:"+resp.Status)
	}else{
		return  nil
	}
}
