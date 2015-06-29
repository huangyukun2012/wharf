
This file is about how to build,install and run wharf on a cluster.

# pre-install

Before you install wharf on your cluster, you should have `go` and `docker` installed on your computer.

The `wharf` system consist of the following components:`docker`,`etcd`,`resource`,`image`,`wharf` and some tools. The system will have one server(ectd server ,wharf server, image server on the same node.) and some clients.Some components should be installed on the server: `wharf`,`docker`,`image`. Some componets should be installed on the clients:`docker`,`resource`,`image` and the bash program in the directory of `Tools`.

Before you run your system , you should modify the network on your computer. All the docker daemon should run with the bridge `br0`, and you shoud bind your `eth0` to the `br0` so that the container can have an access to the Internet.This mean that you should config your network bridge.

# Build

## wharf(on server node)

1.get the code

git clone https://github.com/huangyukun2012/wharf.git wharf

2.install 

cd utils;go build;cd ..
cd server;go build;cd ..
cd wharf;go build; go install

## image(on every node)

1.get the code 

get clone https://github.com/huangyukun2012/image.git image

2.install 

cd image ; go build ; go install 

## resource( on client nodes)

1.get the code

git clone https://github.com/huangyukun2012/image.git resource

2.install 

cd resource ; go build ;go install

## tools(on client nodes)

INSTALL: copy the files in the directory of `tools` to your $PATH

#config

copy the config files in the directory of wharf/config to /etc/wharf/config/. Then modify the files according to your system.

# Run

Firstly you should start the server , and then the client. When it comes to the command , you can see the file `readme.md` in the directory of `wharf/wharf`.




