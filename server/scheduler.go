package server 

import(
	"fmt"
	"encoding/json"
	"net/http"
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

func ListContainers()( []APIContainers, error ){
	path := `http://127.0.0.1:4243/containers/json?` 
	c := &http.Client{}
	body, _, err := c.Do("GET", path, nil)
	if err != nil{
		return nil, err
	}
	var containers []APIContainers
	err = json.Unmarshal(body, &containers)
	if err != nil{
		return nil, err
	}
	return containers, nil
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
	data := json.Marshal(opts)
	res, err := http.Post(url, contentType, strings.newReader(data))
	
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

/*function: create container according to Ares
 fill the task struct for the taskname
 and bind ip to containers
 param:
 	*Ares:global param
	tasknamep: the task to be filled--containerDesc.Id, hostIp, HostMachine 
 return value:
	nil, if succeed; error , if fail.
*/
func CreateContainer2Ares( tasknamep *Task) error{
	var index int32
	for ip, resData := range Ares{
		*tasknamep.Set[index].HostIp = ip
		*tasknamep.Set[index].Hostmachine = resData.Node 
		thisContainerIp, getiperr :=  GetFreeIP()
		if getiperr != nil{
				return getiperr
		}
		*tasknamep.Set[index].ContainerIp = thisContainerIp.String() 
		var opts CreateContainerOptions
		opts.Hostname = thisContainerIp//we set the container hostname as its ip 
		//modifiy: if Docker_nr all = 0, the node will be filter out
		//undefined	
		opts.AttachStdin = true
		opts.AttachStdout = true
		opts.AttachStderr = true
		opts.Tty = true
		opts.OpenStdin =true
		opts.StdinOnce = false
		opts.Cpuset =  utils.DockerNr2String( resData.Docker_nr )
		opts.Image = tasknamep.TaskName
		opts.NetworkDisabled = true

		endpoint := ip + ":" + MasterConfig.DockerService.Port 	
		thisContainer, createerr := CreateContainer(endpoint, opts)
		if createerr != nil{
			return createerr	
		}else{
			*tasknamep.Set[index].ContainerDescp.Id = thisContainer.Id	
		}
		binderr:= BindIpWithContainerOnHost(thisContainerIp, thisContainer.Id, ip)
		if binderr != nil{
			return binderr	
		}
		index++
	}
	return nil
}

func main(){
	res, err := ListContainers()	
	fmt.Println(res[0].ID)
}
