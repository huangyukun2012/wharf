The dir client will be compiled into a execute file `resource`.

It will run as an deamon,which will serve as a http server.This process will provide functions:
0)Listen on one port to act as a http server.
1)set ip to container on This host.
2)Update etcd server.
3)Shut Down this Node.

Before Install the resource , make sure you have config the "resource.conf" file first.

Make sure to run the process with "sudo"
