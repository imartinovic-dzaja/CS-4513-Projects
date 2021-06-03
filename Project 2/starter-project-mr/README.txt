CS4513: Project 3 The MapReduce Library
=======================================

Note, this document includes a number of design questions that can help your implementation. We highly recommend that you answer each design question **before** attempting the corresponding implementation.
These questions will help you design and plan your implementation and guide you towards the resources you need.
Finally, if you are unsure how to start the project, we recommend you visit office hours for some guidance on these questions before attempting to implement this project.


Team members
-----------------

1. Ivan Martinovic (imartinovic@wpi.edu)

Design Questions
------------------

(2 point) 1. If there are n input files, and nReduce number of reduce tasks , how does the the MapReduce Library uniquely name the intermediate files?

f0-0, ... f0-[nReduce-1]
f1-0, ... f1-[nReduce-1]
...
f[n-1]-0 ... f[n-1][nReduce-1]

During the map phase the library creates nReduce intermediate files for each n input files. It runs doMap at least once on every single input file and it hence names the intermediate files "mrtmp.jobName-mapTask-reduceTask", where the mapTask is the currently assigned map task which is producing the intermediate file, i.e. the current input file for which we are producing intermediate files (specified in the arguments of doMap), and reduceTask is the identifier which specifies which reduce worker is supposed to process the intermediate file (which we simply assign through a for loop from 0 to nReduce). This way we uniquely assign nReduce intermediate files for each n input files.

number specifying

(1 point) 2. Following the previous question, for the reduce task r, what are the names of files will it work on?

f0-r,
f1-r,
...
f[n-1]-r

When we run doReduce, we specify the jobName, the total number of input files and the reduceTask number. doReduce with task number r should run on all intermediate files whose second number in the filename is r. By using a for loop counter i from 0 to number of input files, we may construct the filenames of all the appropriate intermediate files for reduceTask r as the following "mrtmp.jobName-i-r"

(1 point) 3. If the submitted mapreduce job name is "test", what will be the final output file's name?

The merge function defined in master_splitmerge.go creates the final output file. The name of the final output file will be "mrtmp.jobName". And if the job name is "test" then the final output file's name will be mrtmp.test


(2 point) 4. Based on `mapreduce/test_test.go`, when you run the `TestBasic()` function, how many master and workers will be started? And what are their respective addresses and their naming schemes?


1 master, 2 workers.
Addresses should look similar to:

/var/tmp/824-UID/mrPID-master
/var/tmp/824-UID/mrPID-worker0
/var/tmp/824-UID/mrPID-worker1

UID and PID can be any integers.

TestBasic() creates a single master and using a for loop creates 2 workers. Using the port() function the addresses are assigned to the master and workers. The addresses, s, are constructed with the following lines
	s := "/var/tmp/824-"
	s += strconv.Itoa(os.Getuid()) + "/"
	s += "mr"
	s += strconv.Itoa(os.Getpid()) + "-"
	s += suffix

Where suffix is supplied in the arguments to port() and designates the index of the worker

Final concatenated product (from above):
/var/tmp/824-UID/mrPID-master
/var/tmp/824-UID/mrPID-worker0
/var/tmp/824-UID/mrPID-worker1

UID and PID can change based on the user (UID is user ID) and instance of the process (PID is process ID)


(4 point) 5. In real-world deployments, when giving a mapreduce job, we often start master and workers on different machines (physical or virtual). Describe briefly the protocol that allows master and workers be aware of each other's existence, and subsequently start working together on completing the mapreduce job. Your description should be grounded on the RPC communications.


(Most students probably will be able to answer this question correctly---as the process is summarized in the project description. The purpose of this question is to get students to read and understand the protocol.)

Just give submission full points for this question.

When the master begins executing it starts a Remote Procedure Call (RPC) server. Through this server workers may register with the master (i.e. let the master know they exist), by calling register(MasterAddress) (NOTE: master address is supplied as an argument to the workers when calling RunWorker). Before workers register with the master, however, they start their own RPC server, through which they may receive commands from the master once they register. After completing the job, the master then issues a killWorkers command which instructs the workers to return the number of completed tasks, and then the master closes its own RPC server by calling stopRPCServer()

(2 point) 6. The current design and implementation uses a number of RPC methods. Can you find out all the RPCs and list their signatures? Briefly describe the criteria
a method needs to satisfy to be considered a RPC method. (Hint: you can look up at: https://golang.org/pkg/net/rpc/)

rpc.NewServer() *Server 
	- returns a new RPC server

func (server *Server) Register(rcvr interface{}) error
	- this lets the RPC Server know what kind of methos can be called, based on what type rcvr is
	these are the restrictions:
	- exported method of exported type
	- two arguments, both of exported type
	- the second argument is a pointer
	- one return value, of type error

	for example if we run Register(master), where master is of Type Master, then the only method we can run is:
		Register(args *RegisterArgs, _ *struct{})

	similarly if we run Register(worker), where worker is of type Worker, then we would be allowed to run methods:
		Shutdown(_ *struct{}, res *ShutdownReply)
		DoTask(arg *DoTaskArgs, _ *struct{})

func ServeConn(conn io.ReadWriteCloser)
	- ServeConn blocks, serving the connection until the client hangs up.
	- In other words, this function is used in an infinite for loop to wait for a client to finish sending the RPC server a request after a a connection has been accepted 
	- For example after the master accepts a connection from a worker it then wait to serve the connection i.e. waits for the client to send the register message
func Dial(network, address string) (*Client, error) 
	- Dial connects to an RPC server at the specified network address. 

func (client *Client) Call(serviceMethod string, args interface{}, reply interface{}) error
	- Call invokes the named function, waits for it to complete, and returns its error status. 

7. We provide a bash script called `main/test-wc-distributed.sh` that allows testing the distributed MapReduce implementation.
- the bash script passed

Errata
------

Describe any known errors, bugs, or deviations from the requirements.

All test cases have passed, however running the bash scripts seems to take a long time on my Virtual Machine (about 3 minutes or so). 
I couldn't find any issues with the code that would make it run so slowly, so I assume it is because of hardware.


---
