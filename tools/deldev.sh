#/bin/bash                                                                                                                                        

if [ $# != 1 ];then
   echo "parameter num should be only one." 
   exit 1
fi

ip link set $1 down
if [ $? -eq 1 ];then
	echo $res
    echo "Error: we can not set $1 down"
    exit 2
fi
ip link delete  $1  
if [ $? -eq 1 ];then
    echo "Error: we can not delete $1."
    exit 3
fi
exit 0

