
package util

import(
	"errors"
	"io"
	"io/ioutil"
	"net"
    "log"
    "os"
    "runtime"
	"strings"
	"strconv"
    "syscall"
	"encoding/json"
	"net/http"
	"fmt"
	"time"
	"os/exec"
)

const(
	DOWN=-1
	ALIVE=0
	UP=1
	MaxFailTime=3
	POSTTYPE="application/x-www-form-urlencoded"

	//http response
	OK="200-no error"
	INVALID_INPUT="404-invalid input"
	SERVER_ERROR="500-server error"
)
/*=====================config======================*/

type Config struct{
	EtcdNode 	Etcd
	Network 	Network
	Server   	Serve
	Docker		DockerService
	Resource 	Resource	
	Image		Image
}

func (c *Config)Init()error{
	filename := "/etc/wharf/wharf.conf"
	reader , err := os.Open(filename)	
	if err != nil{
		PrintErr(filename, err)	
		return err
	}
	err = UnmarshalReader(reader, c)	

	return err 
}


type Etcd struct{
	Ip string
	Port string
	Key string
}

type Network struct{
	Net  	net.IP	
	IPMask 	net.IP 
	Start  	net.IP	
	End	   	net.IP 
}

type Serve struct{
	Ip  string 
	Port string
}

type DockerService struct{
	Port	string
	Bridge	string
}

type Resource struct{
	Port	string
}

type Image struct{
	Port	string
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
	Args []string
}

/*===net ===*/
type Container2Ip struct{
	Id string
	Ip	string
}


//=============response
type HttpResponse struct{
	Status	 string
	Warnings	 []string
}
func (h *HttpResponse)Append( iterm string){
	h.Warnings = append(h.Warnings, iterm) 
}
func (h *HttpResponse)Set(status string, warning string){
	h.Status = status 
	h.Warnings = []string{warning}
}
func (h *HttpResponse)String() string{
	if h==nil{
		return "nil"	
	}
	res , _:= json.Marshal(*h)
	return string(res)
}

//======================operation
func UnmarshalReader( reader io.Reader, res interface{})(  error){
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&res)
	return err
}

/*=============Docker_nr=================*/
func IsAllZero(input []int)bool{
	for i:=0;i<len(input);i++{
		if input[i] != 0{
			return false	
		}	
	}
	return true
}

func PositiveNum(input []int)int{
	var ans int
	for i:=0;i<len(input);i++{
		if input[i] > 0{
			ans++
		}	
	}
	return ans 
}

func GetNozeroIndex(data []int)string{
	var res string 
	for i:=0;i<len(data);i++{
		if data[i]>0 {
			res = res+ strconv.Itoa(i)+","	
		}	
	}
	if  !strings.EqualFold(res,""){
		n := len(res)
		res=res[:n-1]//left out the last comma	
	}
	return res
}

var TimeInterval = []int{1, 5, 15}
var TimeIntervalNum = 3
var TicksType = 4//usr, nice, sys, idle



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

func PrintErr( a ...interface{}){
	fmt.Fprintln(os.Stderr, a...)
}

func ReadContentFromHttpRequest( r *http.Request)([]byte, error){
	var contents []byte
	contents = make([]byte, 1000)
	length, err := r.Body.Read(contents)
	if err != nil && err != io.EOF{
		return nil, errors.New("Server Fail read from the http requst in ReadContentFromHttpRequest()") 
	}
	return contents[:length], nil
}

func ReadContentFromHttpResponse( res *http.Response, ans interface{})(err error){
	defer res.Body.Close()
	contents , _:= ioutil.ReadAll(res.Body)
	unmarshalerr := json.Unmarshal(contents, ans)
	if unmarshalerr != nil{
		return unmarshalerr	
	}else{
		return nil
	}
}
//To do:
//In func Daemon: the os.Stderr should be redirected to log file
func FmtJson( input []byte){
	fmt.Println(string(input))
}

/*
function:
	if no data is send to clock: wait for duration, and then return.
	if data is send to clock: then the function will return .
*/
func Timer(duration time.Duration, clock *chan bool)bool{
	var isTimeOut bool

	var timeout chan bool
	timeout = make(chan bool, 1)

	go func(){
		time.Sleep(duration)	
		timeout <- true
	}()

	select {
		case <- (*clock):
			isTimeOut=false
		case <- timeout:	
			isTimeOut=true
	}

	return isTimeOut
}


/*===================Ip address handler====================*/
func GetNetOfBIp(ip string)string{
	domains := strings.Split(ip, ".")
	res := domains[0]+"."+domains[1]
	return res
}

func GetHostOfBIp(ip string)string{
	domains := strings.Split(ip, ".")
	res := domains[2]+"."+domains[3]
	return res

}

/*===ImageTransportHead===*/
type ImageTransportHeadAPI struct{
	Net string	`192.168`
	FileName string	
	DataIndex int
	Nodes []string `1.1`
	Server	string `ip`
}

type Image2TarAPI struct{
	Image string
	TarFileName string
}

/*===============start some progress====================*/
func StartDocker(brName string)error{
	if brName==""{
		brName="br0"
	}

	fmt.Println("Make sure:you have configed the net interface bridge",brName,", or we will start docker using this br0 by default....")
	_, err0 := exec.LookPath("docker")	
	if err0 != nil{
		errors.New(`You have not install 'docker' in your computer! Please install it first.`)
	}

	//test if docker has started yet
	isDockerStartedCmd := exec.Command("pgrep", "docker")
	isDockerStarted,_ := isDockerStartedCmd.Output()

	if len(isDockerStarted)>0{
		//ddocker is stated yet.
			fmt.Println("Docker deamon is already running(not start by wharf).Please check it is running with ",brName)		
			return nil
	}

	//docker is not stated, so we will start it
	cmd := exec.Command("docker", "-b", brName,  "-d" , "-H", "unix:////var/run/docker.sock" , "-H" ,"0.0.0.0:4243")
	err := cmd.Start()
	var res error
	if err != nil{
		res= errors.New("Fail: can not start docker,"+err.Error())	
	}
	fmt.Println("docker is running with command", `docker -b=`,brName,`-d -H unix://var/run/docker.sock -H tcp://0.0.0.0:4243`)
	return res
}

/*================provide a progress=================*/
func  Progress(first, second int64){
	line := make([]byte,19)
	for i:=0;i<19;i++{
		line[i]='\b'
	}
	fmt.Fprintf(os.Stdout,"%s%9d/%-9d",line,first,second)
	os.Stdout.Sync()
}

func  StringFlush(content string){
	length:=len(content)
	line := make([]byte,length)
	for i:=0;i<length;i++{
		line[i]='\b'
	}
	fmt.Fprintf(os.Stdout,"%s%s",line, content)
	os.Stdout.Sync()
}

/*==================sort of res================*/

type Ip_Cpus struct{
	Ip string
	Cpus int
}

func (i *Ip_Cpus)String()string{
	return i.Ip + "-"+strconv.Itoa(i.Cpus)	
}

type Ip_Cpus_List []*Ip_Cpus

func (list Ip_Cpus_List)Len()int{
	return len(list)
}

func (list Ip_Cpus_List)Less(i, j int)bool{
	if list[i].Cpus > list[j].Cpus{
		return true	
	}else if list[i].Cpus < list[j].Cpus{
		return false	
	}else{
		return list[i].Ip > list[j].Ip
	}
}

func (list Ip_Cpus_List)Swap(i, j int){
	var temp *Ip_Cpus = list[i]
	list[i]=list[j]
	list[j]=temp
}
