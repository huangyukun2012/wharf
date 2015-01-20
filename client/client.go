//package get_machine_info 
/******************************************
1.get info of machine
2.process info
3.send info to etcd
*******************************************/

package main 

import (
	"fmt"
	"os"
	"net"
	"strings"
	"encoding/json"
	"bufio"
	"io"
	"strconv"
	"time"
	"wharf/utils"			
	"github.com/coreos/go-etcd/etcd"

)
type Config struct{
	MasterIp	string
	EtcdNode 	utils.Etcd
}

var ClientConfig Config
var cache []utils.MachineRaw
var index int
var res utils.MachineRaw

const (
	size = 31	
)

func Init(){

	err := GetClientConfig()
	if err != nil{
		fmt.Println(err)
	}
	 index = 0
	cache = make([]utils.MachineRaw, size)
	res, _= GetMachineInfo()
	for i:=0;i<size;i++{
		cache[i]= res
	}
	return 
}

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

func SendInfo2Etcd(){
	// send info to master for timeinterval 1, 5, 15 
	var last [3]int
	var send utils.Machine 

		send.MemInfo = res.MemInfo
		for i:=0 ; i< utils.TimeIntervalNum; i++{
			last[i] = index - utils.TimeInterval[i]*2;	
			if last[i]<0 {
				last[i]= last[i]+size
			}
		}		
		history := []utils.CpuRaw{cache[last[0]].CpuInfo, cache[last[1]].CpuInfo, cache[last[2]].CpuInfo}
		send.CpuInfo = utils.HandleCPU(res.CpuInfo, history)
			if value, err := json.Marshal(&send) ; err != nil{
				panic(err)
			}else{
				machines := []string{"http://" + ClientConfig.EtcdNode.Ip +":" + ClientConfig.EtcdNode.Port}
				Store(machines, res.Ip, string(value), uint64(600*time.Second) )		
				fmt.Println(res.Ip)
				fmt.Println(string(value))
			}
}

func Store(endpoint []string, key string, value string, ttl uint64) (error){
	client := etcd.NewClient(endpoint)
	_, err := client.Set(key, value, ttl)
	if err != nil{
		fmt.Println("Hearbeat failed -- key: ", key,", value:",  value, "-- ", err)
		return err
	}
	return nil
}
/***
Get raw machine info 
***/
func GetMachineInfo( ) (utils.MachineRaw, error){
	var res utils.MachineRaw
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
func GetCpuInfo() (res utils.CpuRaw, err error) {
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
	master := ClientConfig.MasterIp;
	conn, err := net.Dial("udp", master+":80")
    if err != nil {
        fmt.Println(err.Error())
        return "Error"
    }
    defer conn.Close()
    return strings.Split(conn.LocalAddr().String(), ":")[0]
}

/***
get mem info from /proc/meminfo
***/
func GetMem() (res utils.Mem, reserr error){//get mem_total, mem_free
	mem_info_file := "/proc/meminfo"		
	fin, err := os.Open(mem_info_file)
	defer fin.Close()
	if err != nil{
		fmt.Println(mem_info_file, err)	
		reserr = err
		return res, reserr
	}
	lines := bufio.NewReader(fin)
	for i:=0; i<2; i++{
		line, err := lines.ReadString('\n')
		if err != nil || err == io.EOF {
			fmt.Println("Error: invalid content in /proc/meminfo")	
			return res, &utils.ContentError{mem_info_file, err}
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
		fmt.Println(cpu_usage_file, err)
		return 0, res	,err  
	}
	content := bufio.NewReader(fin)
	for{//every line
		line,err := content.ReadString('\n')
		if err != nil || io.EOF==err {
			fmt.Println("err: invalid content in /proc/stat")
			return 0, res, &utils.ContentError{cpu_usage_file, err}
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
		fmt.Println(cpu_loadavg_file, err)
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



func GetClientConfig() ( error ){
	filename := "/etc/wharf/config"
	reader , err := os.Open(filename)	
	if err != nil{
		fmt.Println(filename, err)	
		return err
	}
	ClientConfig , err = UnmarshalConfig(reader)	
	if err != nil {
		fmt.Println(err)	
		return err
	}
	return nil
}

//unmarshal config
func UnmarshalConfig(reader io.Reader )( Config, error){
	decoder := json.NewDecoder(reader)
	var res Config					
	err := decoder.Decode(&res)
	return res, err
}
