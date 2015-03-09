
package util

import(
	"net"
    "log"
    "os"
    "runtime"
    "syscall"

)

const(
	DOWN=-1
	ALIVE=0
	UP=1
	MaxFailTime=3
	POSTURL="application/x-www-form-urlencoded"
)

type Etcd struct{
	Ip string
	Port string
	Key string
}

type IpPool struct{
	Net  	net.IP	
	IPMask 	net.IP 
	Start  	net.IP	
	End	   	net.IP 
}

/*===machine====*/
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
	Status  int32
	FailTime int32
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


type SendCmd struct{
	Data map[string]string
}

/*===net ===*/
type Container2IP struct{
	Id string
	Ip	string
}

type BindResult struct{
	Succeed bool
	Warning string
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


//daemon(0,1) 
func Daemon(nochdir, noclose int) int {

    var ret, ret2 uintptr
    var err syscall.Errno
 
    darwin := runtime.GOOS == "darwin"
 
    // already a daemon
    if syscall.Getppid() == 1 {
        return 0
    }
 
    // fork off the parent process
    ret, ret2, err = syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)
    if err != 0 {
        return -1
    }
 
    // failure
    if ret2 < 0 {
        os.Exit(-1)
    }
 
    // handle exception for darwin
    if darwin && ret2 == 1 {
        ret = 0
    }
 
    // if we got a good PID, then we call exit the parent process.
    if ret > 0 {
        os.Exit(0)
    }
 
    /* Change the file mode mask */
    _ = syscall.Umask(0)
 
    // create a new SID for the child process
    s_ret, s_errno := syscall.Setsid()
    if s_errno != nil {
        log.Printf("Error: syscall.Setsid errno: %d", s_errno)
    }
    if s_ret < 0 {
        return -1
    }
 
    if nochdir == 0 {
        os.Chdir("/")
    }
 
    if noclose == 0 {
        f, e := os.OpenFile("/dev/null", os.O_RDWR, 0)
        if e == nil {
            fd := f.Fd()
            syscall.Dup2(int(fd), int(os.Stdin.Fd()))
            syscall.Dup2(int(fd), int(os.Stdout.Fd()))
            syscall.Dup2(int(fd), int(os.Stderr.Fd()))
        }
    }
 
    return 0
}

//To do:
//In func Daemon: the os.Stderr should be redirected to log file
