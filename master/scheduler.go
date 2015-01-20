package master

import(
	"fmt"
	"encoding/json"
	"net/http"
)
type APIContainers struct{
	ID string
	Image string
	Command string
	Created int64
	Status	string
	Ports []APIPort
	SizeRW	int64
	SizeRootFs	int64
	Names	[]string
}

type APIPort struct{
	PrivatePort	int64
	PublicPort	int64
	Type		string
	IP			string
}

func ListContainers()( []APIContainers, error ){
	path := `http://127.0.0.1:4243/containers/json?` 
	c := &http.Client{}
	body, _, err := c.Do("GET", path, nil)
	if err != nil{
		return nil, err
	}
	var containers []APIContainers
	err = json.Unmarshal(body, &containers)
	if err != nil{
		return nil, err
	}
	return containers, nil
}

func main(){
	res, err := ListContainers()	
	fmt.Println(res[0].ID)
}
