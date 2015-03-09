#/bin/bash
#para:  container_ip, container_id
if [ $# != 2 ];then
	echo -e "error: invalid parameters"
	exit 1
fi

container_netmask=16
container_ip=$1
container_id=$2
Bridge=docker0
container_gw=`LANG=C ifconfig $Bridge | awk '/inet addr:/{ print $2 }' | awk -F: '{print $2 }'`
eth_name=eth0

#get pid
pid=`docker inspect -f '{{.State.Pid}}' $container_id`
if [ -z $pid ];then
	exit 1
fi

#build namespace for pid
if [ ! -d /var/run/netns ];then
	mkdir -p /var/run/netns
fi

if [ ! -f /var/run/netns/$pid ];then
	ln -s /proc/$pid/ns/net /var/run/netns/$pid
fi

#create a virtual network device
random_num=$RANDOM
vnet_dev=veth${random_num}
ifconfig | grep $vnet_dev > /dev/null
res=$?
while [ $res -eq 0 ]
do
	random_num=$RANDOM
	vnet_dev=veth${random_num}
	ifconfig | grep $vnet_dev > /dev/null
	res=$?
done

#now ge get and valid random_num

#create a new peer2peer device
Indev=In${random_num}
ip link add $vnet_dev type veth peer name $Indev

#bind device to docker bridge
brctl addif $Bridge $vnet_dev
if [ $? -eq 1 ];then
	echo "Fail to add interface $vnet_dev to bridge $Bridge "
	exit 1
fi	

ip link set $vnet_dev up
if [ $? -eq 1 ];then
	echo "Fail to set $vnet_dev up"
	exit 1
fi	

ip link set $Indev netns $pid
if [ $? -eq 1 ];then
	echo "Fail to bind $vnet_dev with pid $pid"
	exit 1
fi	

ip netns exec $pid ip link set dev $Indev name $eth_name
if [ $? -eq 1 ];then
	echo "Fail to rename $Indev to $eth_name in pid $pid namespace"
	exit 1
fi	

ip netns exec $pid ip link set $eth_name up
if [ $? -eq 1 ];then
	echo "Fail to set $eth_name up"
	exit 1
fi	

ip netns exec $pid ip addr add $container_ip/16 dev $eth_name
if [ $? -eq 1 ];then
	echo "Fail to bind $container_ip with $eth_name "
	exit 1
fi	

ip netns exec $pid ip route add default via $container_gw > /dev/null
if [ $? -eq 1 ];then
	echo "Fail to set route info for pid $pid"
	exit 1
fi	

exit 0
