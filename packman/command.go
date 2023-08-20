package packman

// Command is a command that can be used for installing and updating a package
type Command struct {
	Name string
	Args []string
}

// Commands contains a set of commands for each operating system
type Commands map[string][]*Command
