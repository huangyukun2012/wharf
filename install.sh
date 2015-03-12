#!/bin/bash
#move setip.sh to $PATH
if [ -f /usr/local/bin/setip.sh ];then
	cp resource/setip.sh /usr/local/bin/setip.sh
fi
#cp etcd to $PATH
if [ -f /usr/local/bin/etcd ];then
	cp etcd /usr/local/bin/etcd
fi
