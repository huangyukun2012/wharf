//package get_machine_info 
/******************************************
1.get info of machine
2.process info
3.send info to etcd
*******************************************/

package main 

import (
	"errors"
	"fmt"
	"os"
	"net"
	"strings"
	"encoding/json"
	"bufio"
	"io"
	"strconv"
	"time"
	"wharf/util"			
	"github.com/coreos/go-etcd/etcd"

)
var flagD *bool


type Service struct{
	Port 	string
}

var ClientConfig util.Config
var cache []util.MachineRaw
var index int
var res util.MachineRaw

const (
	size = 31	
)

func errPrintln(a ...interface{}){
	fmt.Fprintln(os.Stderr, a ...)
}
//get config file, and init the cache data
func Init(){

	err := ClientConfig.Init()
	if err != nil{
		errPrintln("func Init:", err)
	}
	errdocker := startDocker()
	if errdocker != nil{
		util.PrintErr(errdocker.Error())	
	}
	 
	index = 0
	cache = make([]util.MachineRaw, size)
	res, _= GetMachineInfo()
	for i:=0;i<size;i++{
		cache[i]= res
	}
	return 
}


//create an go routine, which will collect the machine info every 30s
func CollectLoop(){

	//This is a while loop forever
	go func(){
		for{
			res, err:= GetMachineInfo()
			if err == nil{
				time.Sleep(30 * time.Second)
			}
			cache[index]= res
			index = (index+1)%size
		}//for loop ever
	}()
	return 
}

/*send the info of this node to etcd server
function: if the etcd server is down, we will try for 10 times. After that the process will exit.

if succeed , returen nil
else: return err
*/
func SendInfo2Etcd(status int32) (error){
	// send info to master for timeinterval 1, 5, 15 
	var last [3]int
	var send util.Machine 
	send.Status = status

	send.MemInfo = res.MemInfo
	for i:=0 ; i< util.TimeIntervalNum; i++{
		last[i] = index - util.TimeInterval[i]*2;	
		if last[i]<0 {
			last[i]= last[i]+size
		}
	}		
	history := []util.CpuRaw{cache[last[0]].CpuInfo, cache[last[1]].CpuInfo, cache[last[2]].CpuInfo}
	send.CpuInfo = util.HandleCPU(res.CpuInfo, history)
	send.FailTime = 0
	if value, err := json.Marshal(&send) ; err != nil{
		panic(err)
	}else{
		machines := []string{"http://" + ClientConfig.EtcdNode.Ip +":" + ClientConfig.EtcdNode.Port}
		if *flagD{
			fmt.Fprintf(os.Stderr, "SendInfo2Etcd: Store data to etcd %s:%s\n", ClientConfig.EtcdNode.Ip,ClientConfig.EtcdNode.Port)	
			fmt.Fprintf(os.Stderr, "ip is %s and data is %s\n", res.Ip, string(value))
		}
		var newerr error
		for i:=0;i<5;i++{
			newerr = Store(machines, res.Ip, string(value), 0 )		
			if newerr!= nil{
				fmt.Fprintf(os.Stderr, "SendInfo2Etcd: Store failed--%s, it will try %d times\n", newerr,4-i)	
			}else{
				break
			}
		}
		return newerr
	}
}


//store the key-value to endpoint
func Store(endpoint []string, key string, value string, ttl uint64) (error){
	client := etcd.NewClient(endpoint)
	_, err := client.Set(key, value, ttl)
	if err != nil{
		if *flagD {
			errPrintln( "Function Store:", "Hearbeat failed -- key: \n", key,", value:",  value, "-- ", err)
			errPrintln("May be the etcd server is down, please check it\n")
		}
	}
	return err 
}

/***
Get raw machine info 
***/
func GetMachineInfo( ) (util.MachineRaw, error){
	var res util.MachineRaw
	var err1 error
	var err2 error
	res.Ip = GetAddr()
	res.MemInfo,err1 = GetMem()
	if err1 != nil{
		return res, err1
	}
	res.CpuInfo,err2 = GetCpuInfo()
	if err2 != nil{
		return res, err2	
	}
	return res, nil
}

/***
get raw info of cpu
***/
func GetCpuInfo() (res util.CpuRaw, err error) {
	var err1 error
	var err2 error
	res.Num, res.Ticks , err1= GetCpuTicks()	
	if err1 != nil{
		return res, err1
	}
	res.Loadavg,err2= GetCpuLoadavg()
	if err2 != nil{
		return res, err2	
	}
	return res, nil
}

/***
get ip address of the machine
***/
func GetAddr() string { //Get ip
	master := ClientConfig.Server.Ip;
	conn, err := net.Dial("udp", master+":80")
    if err != nil {
        errPrintln( err.Error())
        return "Error"
    }
    defer conn.Close()
    return strings.Split(conn.LocalAddr().String(), ":")[0]
}

/***
get mem info from /proc/meminfo
***/
func GetMem() (res util.Mem, reserr error){//get mem_total, mem_free
	mem_info_file := "/proc/meminfo"		
	fin, err := os.Open(mem_info_file)
	defer fin.Close()
	if err != nil{
		errPrintln(mem_info_file, err)	
		reserr = err
		return res, reserr
	}
	lines := bufio.NewReader(fin)
	for i:=0; i<2; i++{
		line, err := lines.ReadString('\n')
		if err != nil || err == io.EOF {
			errPrintln("Error: invalid content in /proc/meminfo")	
			return res, errors.New("invalid content in /proc/meminfo")
		}
		columns := strings.Split(line, " ")
		columns_len := len(columns)
		if columns[0]==string("MemTotal:") {
			res.Total, _ = strconv.Atoi(columns[columns_len -2])			
		}else if columns[0] == string("MemFree:") {
			res.Free,_ = strconv.Atoi(columns[columns_len -2])
		}else{
			break
		}
	}
	return res, nil 
}

/***
get cputicks of all the cpus now
***/
func GetCpuTicks() (cpu_num int, ticks [][]int64,err error ){
	res := make([][]int64, 0)
	line_nr := 0
	cpu_usage_file := "/proc/stat"
	fin, err := os.Open(cpu_usage_file)
	defer fin.Close()
	if err != nil{
		errPrintln(cpu_usage_file, err)
		return 0, res	,err  
	}
	content := bufio.NewReader(fin)
	for{//every line
		line,err := content.ReadString('\n')
		if err != nil || io.EOF==err {
			errPrintln("err: invalid content in /proc/stat")
			return 0, res, errors.New("invalid content in /proc/stat")
		}
		if line[0]=='c' && line[1]=='p' && line[2]=='u'{
			if line[3] != ' '{
				res = append(res, make([]int64, 0))
				columns := strings.Split(line, " ")
				//the columns are : usr, nice, sys, idle
				for i:=1; i<5; i++{//every column
					value,_ := strconv.Atoi(columns[i])	
				//	fmt.Printf("%d*", value)
				  	res[line_nr] = append(res[line_nr], int64(value))
				}
				line_nr += 1
			}
		}else{
			break
		}
	}
	cpu_num = line_nr 
	return cpu_num, res, nil
}
/***
get cpu loadavg from /proc/loadavg of the last 1, 5, 15 minutes
***/
func GetCpuLoadavg( )( []float32, error) {
	res := make([]float32, 3)
	cpu_loadavg_file := "/proc/loadavg"
	fin, err := os.Open(cpu_loadavg_file)
	defer fin.Close()
	if err != nil{
		errPrintln(cpu_loadavg_file, err)
		return res, err
	}
	content := bufio.NewReader(fin)
	line, err := content.ReadString('\n')
	columns := strings.Split(line, " ")
	// we need columns【0：3】		
	for i:=0 ; i<3; i++ {
 		temp, err := strconv.ParseFloat(columns[i], 32) 
		res[i] = float32(temp)
		if err != nil {
			return res, err
		}
	}
	return res, nil
}


//get client config 
func GetClientConfig() ( error ){
	filename := "/etc/wharf/wharf.conf"
	reader , err := os.Open(filename)	
	if err != nil{
		errPrintln(os.Stderr, filename, err)	
		return err
	}
	ClientConfig , err = UnmarshalConfig(reader)	
	if err != nil {
		errPrintln(err)	
		return err
	}
	return nil
}

//unmarshal config
func UnmarshalConfig(reader io.Reader )( util.Config, error){
	decoder := json.NewDecoder(reader)
	var res util.Config					
	err := decoder.Decode(&res)
	return res, err
}
