#/bin/bash
#para: HostIp, container_name,  container_ip
if [ $# != 3 ];then
	echo -e "error: invalid parameters"
	exit 1
fi

container_netmask=16
container_gw=$1
container_name=$2
container_ip=$3
mac='12:34:56:78:9a:bc'
container_id=`docker ps | grep $container_name | awk '{print \$1}'`

pid=`docker inspect -f '{{.State.Pid}}' $container_id`
if [ ! -d /var/run/netns ];then
	mkdir -p /var/run/netns
fi

if [ ! -f /var/run/netns/$pid ];then
	ln -s /proc/$pid/ns/net /var/run/netns/$pid
fi

vnet_dev=veth$RANDOM

ip link add $vnet_dev type veth peer name B
brctl addif docker0 $vnet_dev
ip link set $vnet_dev up

ip link set B netns $pid
ip netns exec $pid ip link set dev B name eth0
ip netns exec $pid ip link set eth0 address $mac
ip netns exec $pid ip link set eth0 up
ip netns exec $pid ip addr add $container_ip/16 dev eth0
ip netns exec $pid ip route add default via $docker0
exit 0

