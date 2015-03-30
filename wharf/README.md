wharf.go contains Wharf's main function

This file provides first line CLI argument parsing and envirionment variable setting

## wharf

Usage of wharf:
  -d, --daemon=false                     Enable daemon mode
  -D, --debug=false                      Enable debug mode
  -r, --restart=false					 Restart previously running wharf 
  -v, --version=false                    Print version information and quit

  

Usage: wharf <command>

where <command> is one of:
 create          Create task
 images          List images
 inspect		 List the details of a task
 kill            Kill tasks
 ps 			 List tasks
 rm				 Remove one or more tasks
 rmi			 Remove one or more images
 version         Show the Wharf version information

wharf <command> -h for subcommand help


###1.wharf version

Usage: wharf version [options]

Show version of wharf.

options:
	-h=false	Show help information and quit.

### 2.wharf create

Usage: whaft create [options] 

Create a task from the user.

options:

  -t=typename				The type of the task. Now it supports 'single' and 'mpi' and the default value is 'mpi'.

  -n=taskname				The name of the task. If the user do not give an taskname, it will provide a random one.

  -s=strategy				The stategy of how we allocate the resource we need. Now it supports 'COM' and 'MEM' and the default value is 'COM'.

  -c=num					The cpu we need for our task.This option is a *must* one.

  -l=floatnum				The biggest allowed average overload in the past 1 minute of one node. Any node who has a bigger average overload than floatnum will be filtered out. If this option is not
  							provided, no nodes will be filtered out.

  -i=imagename				The name of the image, with which we will create our task. This option is also a *must* one.

  -C=num					The container num that we can bind on one cpu.That is to say, the task will share the cpu with other container on the cpu. The default value is 1, which means one cpu for one
  							container.

  -f=filename				The node set which we create our task in. In the filename, we have each ip in one line. The default set is all the nodes with a resource daemon.


### 3.wharf ps 

  
Usage: wharf ps [options]

List tasks.

options:

  -a, --all=false      		Show all tasks. Only running tasks are shown by default.

  -l, --latest=false    	Show only the latest created task, include non-running ones.

  -n=TASKNAME				Show n last created task, include non-running ones.

  -i=IMAGENAME			    Show the task with the image name.

  -t=TYPENAME    			Show the task with the type name.


### 4.wharf inspect 

Usage: wharf inspect [OPTIONS] TASKNAME 

Return low-level information on a task 

  -f, --format=""    Format the output using the given go template.

### 5.wharf stop

Usage: wharf stop [OPTIONS] TASKNAME [TASKNAME...]


Stop a running task---stop all the container related to this task.The Ip and the virtual network device will not be given out by this command.

  --help=false       Print usage

### 6.wharf start 

Usage: wharf start [OPTIONS] TASKNAME [TASKNAME...]

Restart a stopped 	task 

  --help=false       Print usage

### 7.wharf  rm 		

Usage: wharf rm [OPTIONS]  TASKNAME [TASKNAME...]

Remove one or more  tasks. 

  -f, --force=false      Force removal of running  task: stop the task before remove it. 

  -v, --volumes=false    Remove the volumes associated with the containers in the task.

### wharf images          

Usage: docker images [OPTIONS] [TASKNAME]

List images

  -a, --all=false      Show all images (by default filter out the intermediate image layers)

  -f, --filter=[]      Provide filter values (i.e. 'dangling=true')

  -no-trunc=false     Don't truncate output

  -q, --quiet=false    Only show numeric IDs

  -n, --nodes=true     Show the all the nodes who have this image. 


### wharf rmi         

Usage: wharf rmi IMAGE [IMAGE...]

Remove one or more images

  -a, --all=true	  	 	Remove all the images with the name IMAGE in the cluster except for the image on the server.
  -f, --force=false    		Force removal of the image.
  -no-prune=false      		Do not delete untagged parents.
  -nodes=NODE1,NODE2... 	Only remove the imags form the given nodes.

