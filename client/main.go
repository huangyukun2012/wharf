package main

import(
	"fmt"
	"log"
	"net/http"
)

func SetIpHandler(w http.ResponseWriter, r *http.Request){

}

func ShutDownHandler(w http.ResponseWriter, r *http.Request){
	log.Fatal("The client will be shut down!")
	fmt.Println("The client will be shut down!")
	return 
}

func GetResourceHandler(w http.ResponseWriter, r *http.Request){
	SendInfo2Etcd()
}

func main(){
	Init()
	//go routine
	CollectLoop()

	http.HandleFunc("/get_resource", GetResourceHandler)
	http.HandleFunc("/shut_down", ShutDownHandler)
	http.HandleFunc("/set_ip", SetIpHandler)

	err := http.ListenAndServe(":8080", nil)
	if err != nil{
		log.Fatal("ListenAndServe", err.Error())	
	}
}
