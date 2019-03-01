# girun runs file given in first argument, starting up gi gui event loop

import sys
from gi import go

# run filename given in first argument
go.InitRunFileSet(sys.argv[1])
go.Init()

