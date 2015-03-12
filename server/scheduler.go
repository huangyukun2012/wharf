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
		tasknamep.Set[index].HostMachine = resData.Node 
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
		}
		starterr := StartContainerOnHost(thisContainer.Id, ip)
		if starterr != nil{
			return starterr	
		}

		binderr:= BindIpWithContainerOnHost(thisContainerIp.String(), thisContainer.Id, ip)
		if binderr != nil{
			return binderr	
		}
		
		index++
	}
	return nil
}

/*function:
	start a container with id of "id" ont host "hostIP".
	This will give out a http request to the docker deamon on "HostIP".
 return value:
 	nil, when no error
	*/
func StartContainerOnHost( id , hostIp string)error{
	endpoint := "http://" + hostIp 
	path := `/containers/` + id +`/start`

	res, err := http.Post(endpoint+path ,util.POSTTYPE, strings.NewReader("") )

	if err != nil{
		return err
	}
	//err == nil
	if strings.HasPrefix(res.Status, "200"){
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
	endpoint := "http://" + hostIp 
	path := `/containers/` + id +`/stop?t=1`

	res, err := http.Post(endpoint+path ,util.POSTTYPE, strings.NewReader(""))

	if err != nil{
		return err
	}
	//err == nil
	if strings.HasPrefix(res.Status, "200"){
		return nil	
	}else{
		return errors.New(res.Status)	
	}
}
