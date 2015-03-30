//keep in mind that, if we do not execute the program in -d mode, this is an user mode process.
//So if we want to communicate with the wharf daemon, we neeed http request.

package main 

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"strconv"
	"io/ioutil"
	"errors"
	"encoding/json"
	"net/http"
	"wharf/util"
	"wharf/version"
	"wharf/server"
)

const(
	PROGRAM = "wharf"
)

const(
	Create = iota
	Images
	Inspect
	Stop
	Start
	Ps
	Rm
	Rmi
	Version
)
// A map of all of the registered sub-commands.
var cmds map[string]*cmdCont = make(map[string]*cmdCont)

//the flags to be send to server
var sendflags map[string]string = make(map[string]string)

// Matching subcommand.
var matchingCmd *cmdCont

// Arguments to call subcommand's runnable.
var args []string

// Flag to determine whether help is
// asked for subcommand or not
var flagHelp *bool
var flagDaemon *bool
var FlagDebug *bool
var flagRestart *bool
var flagVersion *bool

// Cmd represents a sub command, allowing to define subcommand
// flags and runnable to run once arguments match the subcommand
// requirements.
type Cmd interface {
	//set flags
	Flags(*flag.FlagSet) *flag.FlagSet
	//run cmd with flags and args
	Run(args []string)([]byte)
}

type Subcommand struct{
	fs *flag.FlagSet
	name string
}

//set flags for Subcommand
func (e *Subcommand) Flags(inputfs *flag.FlagSet) *flag.FlagSet{
	e.fs = inputfs
	return inputfs
}

func (e *Subcommand) PrintDefaults() {
	fmt.Println(e.fs.Usage)
	return 
}
func  storeFlag(fl *flag.Flag){
	sendflags[fl.Name]=fl.Value.String()
}

/*give subcommad to httpserver, this acted as a http request
return value:
	error occur:Fail to post, or the server do no answer.
	error nil;the data from the server
*/
func (e *Subcommand ) Run( arg []string) ([]byte, error){

	var contents []byte
	fs := e.fs
		
	if *FlagDebug {
		fmt.Fprintf(os.Stderr, "subcommand %s will be executed\n", e.name)
	}

	fs.Visit(storeFlag)
	var tobesend util.SendCmd 
	tobesend.Data = sendflags
	tobesend.Args = arg
	value, err := json.Marshal(&tobesend)
	if	err != nil{
		panic(err)	
		return contents, err
	}else{
		var url string
		url = "http://" + server.MasterConfig.Server.Ip + ":" + server.MasterConfig.Server.Port +"/" + e.name

		if *FlagDebug {
			fmt.Fprintf(os.Stderr, "the value of post data is %s\n", string(value))	
		}

		res, err2 := http.Post(url, `appliction/x-www-form-urlencoded`, strings.NewReader(string(value)) )	
		if err2 !=nil{
			fmt.Fprintf(os.Stderr, "Error: Fail to Post: %s", err2 )
			return contents, err2
		}else if !strings.HasPrefix(res.Status, "200"){
			util.PrintErr("Error: Server Fail to handle this request, "+res.Status)	
			os.Exit(1)	
			return contents,errors.New("Error: Server Fail to handle this request, "+res.Status) 
		}else{
			defer res.Body.Close()
			var err3 error
			contents = make([]byte, 1000)
			contents, err3 = ioutil.ReadAll(res.Body)
			if err3 != nil{
				fmt.Fprintf(os.Stderr,"%s", err3)	
				return contents, err3
			}else{
				if *FlagDebug {
					fmt.Fprintf(os.Stderr, "the return data of post is %s\n", string(contents))	
				}
				return  contents, nil
			}
		}
	}
	return contents, nil
}

type  flagDec struct {
	name string
	usage string
	value string 
}

type cmdCont struct {//sub command 
	name string
	desc string
	command Subcommand 
	requiredFlags []string
	validFlags []flagDec
}

// Registers a Cmd for the provided sub-command name. E.g. name is the
// `status` in `git status`.
func On(name, description string, requiredFlags []string, validFlags []flagDec) {
	cmds[name] = &cmdCont{
		name: name,
		desc: description,
		requiredFlags: requiredFlags,
		validFlags: validFlags,
	}
	cmds[name].command.name = name
}

func DefUsage(){
	fmt.Fprintf(os.Stderr, "\nA management software for docker\n")
	return 
}


// Prints the usage.
func Usage() {
	//program := os.Args[0]
	program := "wharf"
	if len(cmds) == 0 {
		// no subcommands
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] COMMAND [arg...]\n", program)
		DefUsage()
		return
	}
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] <command> [arg...]\n\n", program)
	fmt.Fprintf(os.Stderr, "where <command> is one of:\n")
	for name, cont := range cmds {
		fmt.Fprintf(os.Stderr, " %-15s %s\n", name, cont.desc)
	}
	if numOfGlobalFlags() > 0 {
		fmt.Fprintf(os.Stderr, "\nGlobal OPTIONS:\n")
		flag.PrintDefaults()
	}
	fmt.Fprintf(os.Stderr, "\n%s <command> -h for subcommand help\n", program)
}

/* function :Print the subcommand usage*/
func subcommandUsage(cont *cmdCont) {
	program := PROGRAM
	switch  cont.name {
		case "version":
			fmt.Fprintf(os.Stderr, "Usage of %s %s:\n", program , cont.name)
			fmt.Fprintf(os.Stderr, `              Just use "wharf version"`+"\n")
		case "create":
			fmt.Fprintf(os.Stderr, "Usage: %s %s [OPTIONS]\n", program , cont.name)
			fmt.Fprintf(os.Stderr, `Create a task from user command`+"\n")
		case "ps":
			fmt.Fprintf(os.Stderr, "Usage: %s %s [OPTIONS]\n", program , cont.name)
			fmt.Fprintf(os.Stderr, `List one or more tasks`+"\n")
		case "inspect":
			fmt.Fprintf(os.Stderr, "Usage: %s %s [OPTIONS] TASKNAME\n", program , cont.name)
			fmt.Fprintf(os.Stderr, `Return low-level information on a task`+"\n")
		case "stop":
			fmt.Fprintf(os.Stderr, "Usage: %s %s [OPTIONS] TASKNAME [TASKNAME...]\n", program , cont.name)
			fmt.Fprintf(os.Stderr, `Stop a running task:stop all the container related to this task.The Ip and the virtual network device will not be given out by this command.`+"\n")
		case "start":
			fmt.Fprintf(os.Stderr, "Usage: %s %s [OPTIONS] TASKNAME [TASKNAME...]\n", program , cont.name)
			fmt.Fprintf(os.Stderr, `Restart a stopped  task.`+"\n")
		case "rm":
			fmt.Fprintf(os.Stderr, "Usage: %s %s [OPTIONS] TASKNAME [TASKNAME...]\n", program , cont.name)
			fmt.Fprintf(os.Stderr, `Restart a stopped  task.`+"\n")
		default:
	}
	// should only output sub command flags, ignore h flag.
	fs := matchingCmd.command.Flags(flag.NewFlagSet(cont.name, flag.ContinueOnError))
	fs.PrintDefaults()

	util.PrintErr("\n"+cont.desc+"\n") 
	linewidth := 160	
	for index := range cont.validFlags{
		each := cont.validFlags[index]
		if len(each.usage)<linewidth{
			util.PrintErr("-"+each.name+"    "+each.usage)	
		}else{
			util.PrintErr("-"+each.name+"    "+each.usage[:linewidth])	
			util.PrintErr("      "+each.usage[linewidth:])	
		}
		/* util.PrintErr("-", eflag) */	
	}

	if len(cont.requiredFlags) > 0 {
		fmt.Fprintf(os.Stderr, "\nrequired flags:\n")
		fmt.Fprintf(os.Stderr, " %s\n\n", strings.Join(cont.requiredFlags, ", "))
	}
}

/*register global flags and subcommand*/
func register( ){
	//register for global flags
	flagDaemon = flag.Bool("d",false,"\tEnable daemon mode" )
	FlagDebug = flag.Bool("D",false,"\tEnable debug mode" )
	flagRestart = flag.Bool("r",false,"\tRestart previously running wharf" )
	flagHelp = flag.Bool("h",false,"\tShow help information and quit" )
	flagVersion = flag.Bool("v",false,"\tPrint version information and quit" )//total help

	//register for sub command
	for _,command := range [][]string{
		{"create", "Create tasks"},
		{"images", "List images"},
		{"inspect", "Give details about a task"},
		{"stop", "Stop a running task"},
		{"start", "Start a stoped task."},
		{"rm", "Remove one or more stoped tasks"},
		{"ps", "List tasks"},
		{"rmi", "Remove one or more images"},
		{"version", "Show the Wharf version information"},
	}{
		//all the Cmd are inited as Version, and we will modify it later!
		if strings.EqualFold( command[0], "create"){
			On(command[0], command[1], []string{"c", "i"},
				[]flagDec{
				{"t","The type of a task.Now it support 'single' and 'mpi'", `mpi`},
				{"n","The name of the task. If the user do not give an taskname, it will provide a random one.", "typename"},
				{"s","The stategy of how we allocate the resource we need. Now it supports 'COM' and 'MEM' and the default value is 'COM'.", "COM" },
				{"c","The cpu we need for our task.This option is a must one.","1"},
				{"l","The biggest allowed average overload in the past 1 minute of one node. Any node who has a bigger average overload than floatnum will be filtered out. If this option is not provided, no nodes will be filtered out.","10.00"},
				{"i","The name of the image, with which we will create our task. This option is also a must one.","imageName"},
				{"C","The container num that we can bind on one cpu.That is to say, the task will share the cpu with other container on the cpu. The default value is 1, which means one cpu for one container","1"},
				{"h","Print Usage.","false"},
			})	
		}else if strings.EqualFold( command[0], "images"){
			On(command[0], command[1],  []string{},
				[]flagDec{
			})	
		}else if strings.EqualFold( command[0], "inspect"){
			On(command[0], command[1],  []string{},
				[]flagDec{
					{"f","Format the output using the given go template",""},
					{"h","Print Usage.","false"},
			})	
		}else if strings.EqualFold( command[0], "stop"){
			On(command[0], command[1],  []string{},
				[]flagDec{
					{"h","Print Usage.","false"},
			})	
		}else if strings.EqualFold( command[0], "start"){
			On(command[0], command[1],  []string{},
				[]flagDec{
					{"h","Print Usage.","false"},
			})	
		}else if strings.EqualFold( command[0], "rm"){
			On(command[0], command[1],  []string{},
				[]flagDec{
				{"f","Force removal of running  task: stop the task before remove it.", `false`},
				{"v","Remove the volumes associated with the containers in the task.", `false`},
				{"h","Print Usage.","false"},
			})	
		}else if strings.EqualFold( command[0], "ps"){
			On(command[0], command[1],  []string{},
				[]flagDec{
				{"a","Show all tasks. Only running tasks are shown by defualt.","false" },
				{"l","Show only the latest created task, include non-running ones.", "false"},
				{"n","Show the task with the task name.", ""},
				{"i","Show the task with the image name.", ""},
				{"t","Show the task with the type name.", ""},
				{"h","Print Usage.","false"},
			})	
		}else if strings.EqualFold( command[0], "rmi"){
			On(command[0], command[1],  []string{},
				[]flagDec{
				{"h","Print Usage.","false"},
			})	
		}else if strings.EqualFold( command[0], "version"){
			On(command[0], command[1],  []string{},
				[]flagDec{
				{"h","Print Usage.","false"},
			})	
		}
	}
}
// Parses the flags and leftover arguments to match them with a
// sub-command. Evaluate all of the global flags and register
// sub-command handlers before calling it. Sub-command handler's
// `Run` will be called if there is a match.
// A usage with flag defaults will be printed if provided arguments
// don't match the configuration.
// Global flags are accessible once Parse executes.
func Parse() {
	//command line args parse: if the flags are not defined in advance, parse will stop when come across some unexpected flags.
	flag.Parse()
	// if there are no subcommands registered,
	// return immediately
	if len(cmds) < 1 {
		return
	}
	
	flag.Usage = Usage
	//sub command is not provided 
	if flag.NArg() < 1 {
		//sub command is not set, global flag will be processed
		if *flagHelp {//-h
			flag.Usage()
			os.Exit(1)
		}
		if *flagVersion {//-v
			version.ShowVersion()	
			os.Exit(1)
		}
		if *FlagDebug {//-D
			/* fmt.Fprintf(os.Stderr, "Debug mode means nothing when used alone.\n") */
			//carefull: This branch will not go out of the function
		}
		if *flagRestart{//-r
			fmt.Fprintf(os.Stderr, "Restart mode is not processed yet\n")
			os.Exit(1)
		}
		if *flagDaemon{//-d
			//run as a deamon
		}
		return 
	}

	//sub command is provided
	if *FlagDebug {
		fmt.Fprintf(os.Stderr, "Now sub command will be parsed in Parse of file wharf.go\n")	
	} 
	//command : wharf create -t=typename  -i=imagename
	name := flag.Arg(0)
	if cont, ok := cmds[name]; ok {
		fs := cont.command.Flags(flag.NewFlagSet(name, flag.ExitOnError))
		//define flags for sub command
		defineFlagset(fs, name)

		// leftout the sub command, and the flag para is also in the flag.Args
		//eg: wharf create -t=mpi -c=200, then 'flag.Args()[1:] ' means `create -t=mpi -c=200`
		fs.Parse(flag.Args()[1:])

		//here, args is null for the eg above
		args = fs.Args()

		matchingCmd = cont

		// Check for required flags.
		flagMap := make(map[string]bool)
		for _, flagName := range cont.requiredFlags {
			flagMap[flagName] = true
		}

		fs.Visit(func(f *flag.Flag) {
			delete(flagMap, f.Name)
		})

		if len(flagMap) > 0 {
			subcommandUsage(matchingCmd)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: Command not found ", name)
		flag.Usage()
		os.Exit(1)
	}
}

//define the flags for fs according to subcommandName
func defineFlagset( fs *flag.FlagSet, subcommandName string) error{

	subcommand , ok := cmds[subcommandName]
	if !ok {
		return errors.New("func defineFlagset: sumcommand is not in cmds") 
	}

	flagnum := len(subcommand.validFlags)
	for i:=0;i<flagnum;i++{
		fs.String(subcommand.validFlags[i].name, subcommand.validFlags[i].value, subcommand.validFlags[i].usage)
	}
	return nil
}

// Runs the subcommand's runnable. If there is no subcommand
// registered, it silently returns.
// The params is passed to the subcommand by global var args
func Run() {
	if matchingCmd != nil {
		if *flagHelp {
			subcommandUsage(matchingCmd)
			return
		}
		res, err := matchingCmd.command.Run(args)
		if err != nil{
			util.PrintErr("Server Error!")	
			return 
		}
		subCmdName := matchingCmd.name
		var output util.HttpResponse
		jsonerr := json.Unmarshal(res, &output)
		if jsonerr!= nil{
			util.PrintErr("Error: Run()--json unmarshal err for output")	
			return 
		}

		//the data is return by the server correctly.
		switch subCmdName{
			case "create":
				if strings.HasPrefix(output.Status,"200"){
					fmt.Println(`Task "`, output.Warnings[0],`"`, "is created!" )		
					return 
				}else{
					util.PrintErr(`Error: Task Created failed. `, output.Warnings[0])	
				}	
			case "ps":
				if strings.HasPrefix(output.Status,"200"){
					fmt.Fprintf(os.Stdout, "%-20s%-30s%-20s%-20s%-20s%-20s\n","Taskname", "Status","Image", "Type", "Size", "CPUs")
					result := output.Warnings
					var psIterm server.PsOutput
					for iterm := range result{
						err := json.Unmarshal([]byte(result[iterm]), &psIterm)	
						if err!= nil{
							util.PrintErr("The result can not be parsed!")
						}
						dotindex := strings.Index(psIterm.Status, ".")
						if dotindex != -1{
							psIterm.Status = psIterm.Status[:dotindex]+"s"
						}
						fmt.Fprintf(os.Stdout, "%-20s%-30s%-20s%-20s%-20s%-20s\n",psIterm.TaskName, psIterm.Status, psIterm.Image, psIterm.Type, strconv.Itoa(psIterm.Size), strconv.Itoa(psIterm.Cpus))
					}
				}else{
					util.PrintErr(`Error: Task ps failed. `, output.Warnings[0])	
				}
			case "inspect":
				if strings.HasPrefix(output.Status,"200"){
					var outputByte []byte
					outputByte = []byte(output.Warnings[0])
					_, exists := sendflags["f"] //such as .state.pid	
					if  !exists{//no field is provided
						var outputData server.InspectOutput
						err = json.Unmarshal(outputByte, &outputData)
						if err != nil{
							util.PrintErr("Error: the data from the server can not be resolved.")	
						}
						util.FmtJson(outputByte)
					}else{
						fmt.Fprintf(os.Stdout, string(outputByte))
					}
				}else{
					util.PrintErr(`Error: Task inspect failed. `, output.Warnings[0])	
				}
			case "stop":
				outputByte := []byte(output.Warnings[0])
				var outputData server.StopOutput
				err = json.Unmarshal(outputByte, &outputData)
				if err != nil{
					util.PrintErr(string(outputByte))
				}

				if strings.HasPrefix(output.Status,"200"){
					fmt.Println(outputData.Warning)
				}else{
					fmt.Println(outputData.String())	
				}
			case "start":
				outputByte := []byte(output.Warnings[0])
				var outputData server.StartOutput
				err = json.Unmarshal(outputByte, &outputData)
				if err != nil{
					util.PrintErr(string(outputByte))
				}

				if strings.HasPrefix(output.Status,"200"){
					fmt.Println(outputData.Warning)
				}else{
					fmt.Println(outputData.String())	
				}
			case "rm":
				outputByte := []byte(output.Warnings[0])
				var outputData server.StartOutput
				err = json.Unmarshal(outputByte, &outputData)
				if err != nil{
					util.PrintErr(string(outputByte))
				}

				if strings.HasPrefix(output.Status,"200"){
					fmt.Println(outputData.Warning)
				}else{
					fmt.Println(outputData.String())	
				}

			default:
				util.PrintErr("subcommand", subCmdName, "is not recognized!")
		}

	}
}

// Returns the total number of globally registered flags.
func numOfGlobalFlags() (count int){
	flag.VisitAll(func(flag *flag.Flag) {
		count++
	})
	return
}

func commandRegAndParse(){
	register()
	Parse()
}
