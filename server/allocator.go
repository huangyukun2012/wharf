//allocator, filter,
package server 
import(
	/* "net" */
	"os"
	"strings"
	"fmt"
	"errors"
	"strconv"
	"wharf/util"
)
const (
	COM=1
	MEM=2
) 

type CreateRequest struct{

	TaskName string
	TypeName string
	TotalCpuNum int
	OverLoadMax float32
	ContainerNumMax int
	ResNode map[string]string
	ImageName string
	Stratergy string
}

func (c *CreateRequest )Init(){
	var num int
	for {
		c.TaskName =  strconv.Itoa(int(TasksIndex))
		_, exist := Tasks[c.TaskName]
		if exist {
			TasksIndex ++	
		}else{
			break	
		}
		num++
		if num > 1000{
			fmt.Fprintf(os.Stderr, "There are two many tasks now, please delete some to create new  one")	
			os.Exit(1)
		}
	}

	TasksIndex = (TasksIndex +1)/TaskNumMax
	c.TypeName = "mpi" 
	c.TotalCpuNum = 0 
	c.OverLoadMax = 100 
	c.ContainerNumMax = 1 
	c.ResNode = make(map[string]string , 1)
	c.ImageName ="" 
	c.Stratergy = "COM"
}

/*
from Gres, we filter out some node and cpu.
Only such node is valid:
1)status is UP(not alive or down)
2)Loadavg and  filter file statisfied.
3)Mention: the Docker_nr <0 is ok for the result of Filter
*/
func Filter( r CreateRequest) error{
	for ip, nodeRes := range Gres {
		if nodeRes.Node.Status  == util.UP{
			if nodeRes.Node.CpuInfo.Loadavg[1] < r.OverLoadMax{
				filterb, exist := r.ResNode[ip]
				if !exist || strings.EqualFold(filterb,"1"){
					num := len(nodeRes.Docker_nr)
					for i:=0; i<num;i++{
						nodeRes.Docker_nr[i] = r.ContainerNumMax - nodeRes.Docker_nr[i]	
					}
					Rres[ip] = nodeRes
				}
			}	
		}
	}
	return nil
}

/*
Function:
	Allocate cpus according to typename and statergy
Param:
	r: r.TotalCpuNum, r.Stratergy
*/
func Allocate( r CreateRequest)error{

	totalCPuNum := r.TotalCpuNum

	if strings.EqualFold(r.TypeName, "single"){
		var ip string
		var nodeRes Res
		for ip, nodeRes = range Rres {
			totalCPuNum := r.TotalCpuNum
			fmt.Println("docker_nr is:", nodeRes.Docker_nr)
			for i:=0;i<len(nodeRes.Docker_nr);i++ {
				if	nodeRes.Docker_nr[i] > 0{
					totalCPuNum--	
				}
			}
			if totalCPuNum <= 0 {
				totalCPuNum = r.TotalCpuNum
				for i:=0;i<len(nodeRes.Docker_nr);i++{
					if	nodeRes.Docker_nr[i] > 0{
						nodeRes.Docker_nr[i]=1
						totalCPuNum--
					}else{
						nodeRes.Docker_nr[i]=0
					}
					if totalCPuNum <=0{
						break //from for i	
					}
				}
				Ares[ip]=nodeRes
				return nil
				/* break //from for ip */
			}
		}//for ip
		fmt.Println("totalCPuNum:", totalCPuNum)
		if totalCPuNum > 0{
			return  errors.New("We can not allocate so many cpus in a single machine")	
		}else{
			Ares[ip]=nodeRes
			return nil
		}
	}else if strings.EqualFold(r.TypeName, "mpi"){
		var err error
		switch r.Stratergy{

			case "COM":
				err = AllocateCom( r)
				return err
			case "MEM":
				err = AllocateMem( r)
				return err
			default://random
				for ip, nodeRes := range Rres{
					var HasNozero bool
					for i:=0;i<len(nodeRes.Docker_nr);i++{
						HasNozero = false
						if nodeRes.Docker_nr[i]>0{
							HasNozero = true 
							totalCPuNum--	
							if totalCPuNum <=0  {
								break// from for i	
							}
							nodeRes.Docker_nr[i]=1
						}else{
							nodeRes.Docker_nr[i] =0
						}	
					}
					if !HasNozero {
						continue
					}
					Ares[ip] = nodeRes
					if totalCPuNum <=0 {
						break //from for ip
					}	
				}
				if totalCPuNum > 0{
					return errors.New("Not so many cpus for use----allocate failed.")
				}else{
					return nil
				}
		}		
	}
	return nil
}

func AllocateCom( r CreateRequest) error{
	return nil
}

func AllocateMem( r CreateRequest) error{
	return nil
}

