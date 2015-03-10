//allocator, filter,
package server 
import(
	"net"
	"errors"
)
const (
	COM=1
	MEM=2
) 

type CreateRequest struct{

	TaskName string
	TypeName string
	TotalCpuNum int32
	OverLoadMax float32
	ContainerNumMax int32
	ResNode map[string]string
	ImageName string
	Stratergy int32
}

/*
from Gres, we filter out some node and cpu
*/
func Filter( r CreateRequest) error{
	for ip, nodeRes := range Gres {
		if nodeRes.Node.Status  == utils.UP{
			if nodeRes.Node.CpuInfo.Loadavg[1] < r.OverLoadMax{
				filterb, exist := r.ResNode[ip]
				if !exist || strings.EqualFold(filterb,'1'){
					num := len(nodeRes.Docker_nr)
					for i:=0; i<num;i++{
						nodeRes.Docker_nr[i] = ContainerNumMax - nodeRes.Docker_nr[i]	
					}
					RRes[ip] = nodeRes
				}
			}	
		}
	}
	return nil
}

/*
Allocate cpus according to typename and statergy
*/
func Allocate( r CreateRequest)error{

	totalCPuNum := r.TotalCpuNum

	if strings.EqualFold(r.TypeName, "single"){
		for ip, nodeRes := range Rres {
			totalCPuNum := r.TotalCpuNum
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
				break //from for ip
			}
		}//for ip
		if totalCPuNum > 0{
			return  errors.New("We can not allocate so many cpus in a single machine")	
		}else{
			Ares[ip]=nodeRes
			return nil
		}
	}else if strings.EqualFold(r.TypeName, "mpi"){
		var err error
		switch r.Stratergy{

			case COM:
				err = AllocateCom( r)
				return err
			case MEM:
				err = AllocateMem( r)
				return err
			default://random
				for ip, nodeRes := range Rres{
					for i:=0;i<len(nodeRes.Docker_nr);i++{
						if nodeRes.Docker_nr[i]>0{
							totalCPuNum--	
							if totalCPuNum <=0  {
								break// from for i	
							}
							nodeRes.Docker_nr[i]=1
						}else{
							nodeRes.Docker_nr[i] =0
						}	
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
}

func AllocateCom( r CreateRequest){
}

func AllocateMem( r CreateRequest){
}
