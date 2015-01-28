package master

import(
	"log"
)
type WharfCli struct{
	
}
func (cli *WharfCli ) CmdHelp(args ...string) error {
	if len(args) > 0 {
		method, exists := cli.GetMethod(args[0])
		if !exists{
			fmt.Fprintf(cli.err, "Error: Command not found: %s\n", args[0])
		}else {
			method("--help")
			return nil
		}
	}

	help := fmt.Sprintf("Usage: wharf [OPITIONS] COMMAND [arg...]\n")
	for _,command := range [][]string{
		{"image", "List images"}
		{"task", "List tasks"}
		{"kill", "Kill tasks"}
		{"create", "Create tasks"}
		{"version", "Show the Wharf version information"}
		{"remove", "Remove one or more tasks"}
		{"removei", "Remove one or more images"}
	}{
		help += fmt.Sprintf("	%-10.10s%s\n", command[0], command[1])
	}
	fmt.Fprintf(cli.err, "%s\n", help)
	return nil
}

func main()(){
	if reexec.Init(){
		return 	
	}

	flag.Parse()

	if *flVersion{
		showVersion()	
		return 
	}

	if *flDebug{
		os.Setenv("DEBUG", "1")	
	}
}

func Usage(){
	
}

func parse(){

}
