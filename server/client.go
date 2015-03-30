/*
This file include some struct about the user request:

Keep in mind: client.go is for 'wharf' and container.go is for 'docker'
*/

package server

import(

	"time"
	"encoding/json"
	"wharf/util"
)

/*======ps request and response====*/
type PsRequest struct{
	All bool
	Latest bool
	Name string 
	Image string 
	Type string
}

func (p *PsRequest)Init() {
	p.All = false
	p.Latest = false
	p.Name = ""
	p.Image = ""
	p.Type = ""
	return 
}

type PsOutput struct{
	TaskName string
	Status 	string
	Type	string
	Image	string
	Size	int
	Cpus	int
}
/*======inspect request and response====*/
/* In this secition ,we will get all the data , and 
then we will handle this data in the client--not in 
the server*/
type InspectReqest struct{
	Field string
}

func (ins *InspectReqest)Init(){
	ins.Field=""
}

type InspectOutput struct{
	Cmd  CreateRequest
	Status	string//how long has it ran
	Set	[]CalUnitStr// we must allocate first
}

func (in *InspectOutput )GetData(thisTask Task){
	in.Cmd=thisTask.Cmd
	in.Set = make([]CalUnitStr, len(thisTask.Set))
	for i :=0;i< len(thisTask.Set);i++ {
		in.Set[i].GetData( thisTask.Set[i])		
	}

	if thisTask.Status == DOWN{
		in.Status = "DOWN" 
	}else{
		in.Status = "RUNNING for " + time.Since(thisTask.CreatedTime).String()
	}   
}

type  CalUnitStr struct{
	ContainerIp string
	ContainerId string
	HostIp string
	Status	string
}

/* get information of every calUnit:
	ContainerIp, ContainerId, HostIp, Status
*/
func ( c *CalUnitStr )GetData( input CalUnit){
	c.ContainerIp = input.ContainerIp
	c.HostIp = input.HostIp
	c.ContainerId = input.ContainerDesc.Id

	endpoint := c.HostIp+`:`+MasterConfig.Docker.Port
	var opts =InspectReq{Id:c.ContainerId}
	res, err := InspectContainer(endpoint,opts )	
	if err != nil{
		c.Status=err.Error()	
	}else{
		if res.State.Running{
			c.Status="Running"	
		}else{
			c.Status="Down"	
		}
	}
}

/*================stop/start==============*/
type StopOutput struct{
	Warning string		
	FailNodes []StopFailNode
} 

type StopFailNode struct{
	ContainerId string
	HostIp string
	ErrInfo	string	
}

func (s *StopOutput)String() string{
	if s==nil{
		return "nil"	
	}	
	res ,_ := json.Marshal(*s)
	return string(res)
}

func (s *StopOutput)Append(id, ip, err string) {
	s.FailNodes = append(s.FailNodes, StopFailNode{ContainerId:id, HostIp:ip, ErrInfo:err})
	return 
}

type StopRequest struct{
	Args []string
	FsName map[string]bool
}

func (s *StopRequest)Init( input *util.SendCmd){
	s.Args = input.Args
	s.FsName = make(map[string]bool, 1)
}

//start
type StartOutput StopOutput

type StartRequest StopRequest

func (s *StartRequest)Init( input *util.SendCmd){
	s.Args = input.Args
	s.FsName = make(map[string]bool, 1)
}

func (s *StartOutput)String() string{
	if s==nil{
		return "nil"	
	}	
	res ,_ := json.Marshal(*s)
	return string(res)
}

func (s *StartOutput)Append(id, ip, err string) {
	s.FailNodes = append(s.FailNodes, StopFailNode{ContainerId:id, HostIp:ip, ErrInfo:err})
	return 
}

//rm
type RmOutput StopOutput

type RmRequest StopRequest

func (s *RmRequest)Init( input *util.SendCmd){
	s.Args = input.Args
	s.FsName = make(map[string]bool, 1)
}

func (s *RmOutput)String() string{
	if s==nil{
		return "nil"	
	}	
	res ,_ := json.Marshal(*s)
	return string(res)
}

func (s *RmOutput)Append(id, ip, err string) {
	s.FailNodes = append(s.FailNodes, StopFailNode{ContainerId:id, HostIp:ip, ErrInfo:err})
	return 
}

