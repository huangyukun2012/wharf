#/bin/bash
#install the resource process on client.This shell script must be executed in this directory.
echo "Before you run the install.sh, make sure the resrouce.conf file in directory config has been configed. 
mv setip.sh /usr/local/bin
if [ -d /etc/wharf/ ];then
	mkdir /etc/wharf
fi
cp ../config/config /etc/wharf/
