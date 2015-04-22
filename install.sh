#!/bin/bash
#make====================
cd server;go build
cd ../wharf;go build
cd ../resource;go build
cd ../image;go build
cd ..

#move bindip.sh to $PATH
if [ -f /usr/local/bin/bindip.sh];then
	cp tools/bindip.sh /usr/local/bin/bindip.sh
fi

#cp etcd to $PATH
if [ -f /usr/local/bin/etcd ];then
	cp tools/etcd /usr/local/bin/etcd
fi


#cp etcd to $PATH
if [ -f /usr/local/bin/etcd ];then
	cp tools/etcd /usr/local/bin/etcd
fi

#cp etcd to $PATH
if [ -f /usr/local/bin/etcd ];then
	cp tools/etcd /usr/local/bin/etcd
fi
