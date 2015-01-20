//package get_machine_info 
/******************************************
1.get info of machine
2.process info
3.send info to etcd
*******************************************/

package utils 

import(
	"net"
)

type Etcd struct{
	Ip string
	Port string
	Key string
}

type IpPool struct{
	Net 	net.IP
	IPMask net.IPMask
	Start   net.IP
	End	    net.IP
}

type Mem struct {
	Free int
	Total int
}

type CpuRaw struct {
	Num	int
	Loadavg []float32// 1, 5, 15
	Ticks [][]int64// every cpu in every type
}

type Cpu struct{
	Num int
	Loadavg []float32// for 1, 5, 15
	Usage [][]float32// for 1, 5, 15 and for different cpu
}

type Machine struct{
	MemInfo Mem
	CpuInfo Cpu
}

type MachineRaw struct{
	Ip string
	MemInfo Mem
	CpuInfo CpuRaw
}

type ContentError struct{
	Name string
	Err error
}

var TimeInterval = []int{1, 5, 15}
var TimeIntervalNum = 3
var TicksType = 4//usr, nice, sys, idle

func (e *ContentError ) Error() string{
	return "Error: content is not right in " + e.Name + e.Err.Error()
}

func (e *IpPool)Error( ) string{
	str := "no invalid ip is available in the ip pool"
	return str
}
func HandleCPU( now CpuRaw, last []CpuRaw) (res Cpu) {
	//timeIntervalNum == len(last)
	res.Num = now.Num
	res.Loadavg = now.Loadavg
	TimeIntervalNum := len(TimeInterval)
	res.Usage = make([][]float32, TimeIntervalNum)	

	for i:=0;i<TimeIntervalNum;i++{
		res.Usage[i] = make([]float32, res.Num)
		//the columns are : usr, nice, sys, idle
		var sum int64
		for k:=0;k<res.Num; k++{
			var temp int64
			sum = 0
			for j:=0;j<TicksType;j++{
				temp = now.Ticks[k][j] - last[i].Ticks[k][j]	
				sum += temp
			}
			if sum == 0{
				res.Usage[i][k] = 0	
			}else{
				res.Usage[i][k] = float32(temp)/float32(sum ) 
			}
		}
	}//for loop :i
	
	return res
}
