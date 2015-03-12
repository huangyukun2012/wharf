//In the function Update, we should use go routine
package server 

import(
	"net"
	"fmt"
	"strings"
	"errors"
	"encoding/json"
	"io/ioutil"
	"os/exec"
	"net/http"
	"wharf/util"
)

var  NetworkSize int32
var  usage []byte
var  Current int32
var  Updated bool

type IpError struct{
	Name string
}

func (e *IpError)Error() string{
	return e.Name
}

func initNetwork( ){
	// for a net of B type , there well be 2^16 ips. Among all the ips, x.x.0.0 and x.x.255.255 will be left out
	// we map it by bit, and 2^13B will be needed to store the ips
	NetworkSize =  1<< 14
	usage = make([]byte, NetworkSize)//1 means can be use, and 0 means can not be use
	// At the begin, all the ips can not be used
	Current = 1
	Updated = false
	SetStateByIndex(0, 0)
	Update()
}

/*the first time, we will update all the bits of usage.
After that, we call this function because some container or host shut down without their ips collected. 
So we just test these ips not available
*/
func Update(){
	fmt.Println("Ip pool begin to update..." ) 
	var count int32
	var bytes [4]byte
	bytes[0] =  MasterConfig.Network.Net[12] 
	bytes[1] =  MasterConfig.Network.Net[13] 
	var i, j byte
	for i=0;i<=255;i++ {
		for j=0;j<=255;j++{
			bytes[2]= i
			bytes[3]= j 
			if i==0&&j==0 || i==255&&j==255 || Updated&&TestStateByIndex(int32(i*255 + i +j))==1{
				continue
			}
			thisIp := net.IPv4(bytes[0], bytes[1], bytes[2], bytes[3])	
			if *flagDebug{
				fmt.Println(thisIp.String())
			}
			var reachable bool
			reachable = IsReachable(thisIp)
									
			if  reachable == false{//ip is not reachable 
				SetStateByIp(thisIp, 1)	
				count++
			}else{

			}
		}
	}
	fmt.Println("Ip pool updated: " , count, " ips are avaiable")
	Updated = true
}

//note: net.IP have a len of 16 byte
func SetStateByIp( thisIp net.IP, value byte){
	node := thisIp[14:]
	var bitIndex int32
	bitIndex = int32(node[0]) * 256 + int32(node[1])
	SetStateByIndex(bitIndex, value)
}

//Set the state of ip by index, we will call this function when malloc or free an ip
func SetStateByIndex(index int32, value byte){
	bitIndex := index
	byteIndex := bitIndex/8	
	bitIndex=bitIndex%8

	var unit byte
	unit = 1>> uint32(bitIndex) 
	if value==1{
		usage[byteIndex] |= unit 
	}else{
		usage[byteIndex] ^= unit	
	}
	return 
}

//Test if the ip is invalid accoring to usage
func TestStateByIP( thisIp net.IP ) (value byte) {
	node := thisIp[14:]
	var bitIndex int32
	bitIndex = int32(node[0]) * 256 + int32(node[1])
	value = TestStateByIndex(bitIndex)
	return value	
}


//Test if the IndexTH ip is invalid accoring to usage
func TestStateByIndex(index int32) (value byte){
	bitIndex := index
	byteIndex := bitIndex/8	
	bitIndex=bitIndex%8

	var unit byte
	unit = 1>> uint32(bitIndex) 
	value = usage[byteIndex] & unit
	if value!=0{
		return 1
	}else{
		return 0
	}
}

func CheckError( err error){
	if err!= nil{
		fmt.Println(err)	
	}
}

//Test if and ip is readchable
func IsReachable( thisIP net.IP ) bool{
	ipString := thisIP.String()
	_, err0 := exec.LookPath("ping")
	if err0 != nil{
		fmt.Println("You have not install ping in your computer! Please install it first.")
		return false
	}
	cmd := exec.Command("ping", "-c 1" , "-w 1", ipString)
	err := cmd.Run()//This command will take 1 second
	if err == nil{
		return true
	}else{
		return false
	}
}

//Get free Ip accoring to the IP pool
//make sure that the 0.0.0.0 and 255.255 are marked invalid
func GetFreeIP( ) (net.IP, error){
	var res net.IP
	var i, index int32
	for i=0;i<256*256;i++{
		index = i+ Current
		if index == 0 || index==255*255{
			continue
		}
		if index >= 256*256{
			index %= (256*256)
		}
		value := TestStateByIndex(index)	
		if value==1 {
			Current += (i+1)
			res, err := Index2Ip(index)	
			return res , err
		}
	}			

	res = net.IPv4(0, 0, 0, 0)
	err := &IpError{"No ip is available in the IP pool"} 
	return res, err
}


//Change the index to Ip accroing the the config file to get the Net IP
func Index2Ip( index int32)(net.IP, error){
	var  res net.IP	
	var err error
	var ip [4]byte
	ip[0] = MasterConfig.Network.Net[12]
	ip[1] = MasterConfig.Network.Net[13]
	ip[2] = byte(index/256)
	ip[3] = byte(index%256 )
	res = net.IPv4(ip[0], ip[1], ip[2], ip[3])
	if index >= 256*256 || index < 0{
		err = &IpError{"The index to IP is invalid!"}
		return res,err 
	}
	return res, nil
}

func MallocIp( num int32)( res []net.IP, err error){
	res = make([]net.IP, num)
	var i int32
	for i=0;i<num;i++ {
		res[i], err = GetFreeIP()	
		if err != nil{
			return res, err
		}
	}
	return res,nil 
}

func FreeIp( res []net.IP){
	length := len(res)
	for i:=0; i<length; i++{
		SetStateByIp(res[i], 0)
	}
	return 
}


/*function: bind a ip to a container in a host.
	This is a http request posted to the "docker server". Actrually, it is handler by module of resource.
	param:	
		ip: the ip to be bind.
		id: the id of the container
		hostip:the hostip
	return value:
*/
func BindIpWithContainerOnHost(containerIp string, id string , hostIp string )(error ){
	port := MasterConfig.Resource.Port	
	endpoint := "http://" + hostIp + ":" + port
	var container2IP util.Container2Ip
	container2IP = util.Container2Ip{id, containerIp}
	data, jsonerr := json.Marshal(container2IP)
	if jsonerr != nil{
		return jsonerr
	}
	res, err := http.Post(endpoint, util.POSTTYPE, strings.NewReader(string(data)) )
	if err != nil{
		return err
	}
	//err == nil
	defer res.Body.Close()
	data , _= ioutil.ReadAll(res.Body)
	var result util.BindResult		
	jsonerr = json.Unmarshal(data, &result)
	if jsonerr != nil{
		return jsonerr	
	}else{
		if result.Succeed{
			return nil
		}else{
			return errors.New(result.Warning)	
		}
	}
}
