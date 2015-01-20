#/bin/bash
#para: hostIP, container_name, 
if [ $# != 2 ];then
	echo -e "error: invalid parameters"
	exit 1
fi

container_netmask=16
container_gw=10.18.111.1000

container_name = $1
bridge_if= veth_`echo ${container_name} | cut -c 1-10`
container_ip = $2/${container_netmask}

container_id=`docker ps | grep $1 | awk '{print \$1}'`

mkdir -p /var/run/netns
pid=11153
ln -s /proc
