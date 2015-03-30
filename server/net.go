//In the function Update, we should use go routine
package server 

import(
	"net"
	"fmt"
	"os"
	"os/exec"
	"wharf/util"
)

var  NetworkSize int32
var  freeBit []byte//
var  startIp  net.IP
var  endIp net.IP
var  Current int32
var  Updated bool
type IpError struct{
	Name string
}

func (e *IpError)Error() string{
	return e.Name
}
type networkConfig struct{
	Busy []net.IP
}

func initNetwork( ){
	// for a net of B type , there well be 2^16 ips. Among all the ips, x.x.0.0 and x.x.255.255 will be left out
	// we map it by bit, and 2^13B will be needed to store the ips
	NetworkSize =  1<< 14
	freeBit = make([]byte, NetworkSize)//1 means can be use, and 0 means can not be use
	var i int32
	for i=0 ;i<NetworkSize; i++{
		freeBit[i]=0xff
	}
	// At the begin, all the ips can not be used
	Current = int32 (MasterConfig.Network.Start[14] )*256 + int32 ( MasterConfig.Network.Start[15])
	Updated = false
	Update()
}

/*the first time, we will update all the bits of freeBit.
After that, we call this function because some container or host shut down without their ips collected. 
So we just test these ips not available.
*/
func Update()error{
	fmt.Println("Ip pool begin to update..." ) 

	var data networkConfig
	filename := `/etc/wharf/network.conf`	
	reader , err := os.Open(filename)
	if err!=nil{
		return err	
	}
	
	errjson := util.UnmarshalReader(reader, &data)
	if errjson != nil{
		return errjson	
	}
	busynum := len(data.Busy)
	for i:=0;i<busynum;i++{
		thisip := data.Busy[i]
		SetStateByIp(thisip, 0)	
	}
	Updated = true
	fmt.Println("OK:	Ip pool update!" ) 
	return nil
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
		freeBit[byteIndex] |= unit 
	}else{
		freeBit[byteIndex] ^= unit	
	}
	return 
}

//Test if the ip is invalid accoring to freeBit
func TestStateByIP( thisIp net.IP ) (value byte) {
	node := thisIp[14:]
	var bitIndex int32
	bitIndex = int32(node[0]) * 256 + int32(node[1])
	value = TestStateByIndex(bitIndex)
	return value	
}


//Test if the IndexTH ip is invalid accoring to freeBit
func TestStateByIndex(index int32) (value byte){
	bitIndex := index
	byteIndex := bitIndex/8	
	bitIndex=bitIndex%8

	var unit byte
	unit = 1>> uint32(bitIndex) 
	value = freeBit[byteIndex] & unit
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
	var i, index, start, end int32
	start = 256 * (int32 (MasterConfig.Network.Start[14])) + int32(MasterConfig.Network.Start[15])
	end = 256 * (int32 (MasterConfig.Network.End[14])) + int32(MasterConfig.Network.End[15])
	index= Current+1
	for i=start;i<=end;i++{
		if index>end{
			index=start	
		}
		value := TestStateByIndex(index)	
		if value==1 {
			res, err := Index2Ip(index)	
			SetStateByIndex(index, 0)
			Current=index
			return res , err
		}
		index++
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

