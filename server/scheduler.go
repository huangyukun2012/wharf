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
	path := "/containers/create?"
	url := "http://" + endpoint + path
	contentType := "application/json"
	data, _:= json.Marshal(opts)
	res, err := http.Post(url, contentType, strings.NewReader(string(data)))
	
    if err != nil{
		return nil, err
    }else{
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
