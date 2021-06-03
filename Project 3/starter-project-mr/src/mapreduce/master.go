package mapreduce

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Master holds all the state that the master needs to keep track of. Of
// particular importance is registerChannel, the channel that notifies the
// master of workers that have gone idle and are in need of new work.
type Master struct {
	sync.Mutex

	address         string
	registerChannel chan string
	doneChannel     chan bool
	workers         []string // protected by the mutex


	// Per-task information
	jobName string   // Name of currently executing job
	files   []string // Input files
	nReduce int      // Number of reduce partitions

	shutdown chan struct{}
	l        net.Listener
	stats    []int

	phase		jobPhase
	notCompletedIndexes []string // protected by the mutex (contains list of non-completed files for recovery purposes)
	abortChannel chan bool
	hasAborted bool
}

// Register is an RPC method that is called by workers after they have started
// up to report that they are ready to receive tasks.
func (mr *Master) Register(args *RegisterArgs, _ *struct{}) error {
	mr.Lock()
	
	debug("Register: worker %s\n", args.Worker)
	mr.workers = append(mr.workers, args.Worker)
	//make sure to write the name down

	
	go func() {
		mr.registerChannel <- args.Worker
	}()

	mr.Unlock()	
	mr.updateWorkerLog(args.Worker)	
	return nil
}

// newMaster initializes a new Map/Reduce Master
func newMaster(master string) (mr *Master) {
	mr = new(Master)
	mr.address = master
	mr.shutdown = make(chan struct{})
	mr.registerChannel = make(chan string)
	mr.doneChannel = make(chan bool)
	mr.phase = mapPhase	//initial phase is the mapPhase
	mr.abortChannel = make(chan bool)
	mr.hasAborted = false
	return
}

// Sequential runs map and reduce tasks sequentially, waiting for each task to
// complete before scheduling the next.
func Sequential(jobName string, files []string, nreduce int,
	mapF func(string, string) []KeyValue,
	reduceF func(string, []string) string,
) (mr *Master) {
	mr = newMaster("master")
	go mr.run(jobName, files, nreduce, func(phase jobPhase) {
		switch phase {
		case mapPhase:
			for i, f := range mr.files {
				doMap(mr.jobName, i, f, mr.nReduce, mapF)
			}
		case reducePhase:
			for i := 0; i < mr.nReduce; i++ {
				doReduce(mr.jobName, i, len(mr.files), reduceF)
			}
		}
	}, func() {
		mr.stats = []int{len(files) + nreduce}
	})
	return
}

// Distributed schedules map and reduce tasks on workers that register with the
// master over RPC.
func Distributed(jobName string, files []string, nreduce int, master string) (mr *Master) {
	mr = newMaster(master)
	mr.startRPCServer()
	go mr.run(jobName, files, nreduce, mr.schedule, func() {
		fmt.Printf("Master Distributed: killing workers!\n")
		mr.stats = mr.killWorkers()
		mr.stopRPCServer()
	})
	return mr
}



// run executes a mapreduce job on the given number of mappers and reducers.
//
// First, it divides up the input file among the given number of mappers, and
// schedules each task on workers as they become available. Each map task bins
// its output in a number of bins equal to the given number of reduce tasks.
// Once all the mappers have finished, workers are assigned reduce tasks.
//
// When all tasks have been completed, the reducer outputs are merged,
// statistics are collected, and the master is shut down.
//
// Note that this implementation assumes a shared file system.
func (mr *Master) run(jobName string, files []string, nreduce int,
	schedule func(phase jobPhase),
	finish func(),
) {

	go func() {	
	
		startTimeStamp := time.Now()
		mr.jobName = jobName
		mr.files = files
		mr.nReduce = nreduce
		
		if (mr.hasAborted) {
			return	
		}
		
		hasRecovered := mr.recoverMaster()		//recover state if possible
		if (hasRecovered) {fmt.Printf("%s: Continuing Map/Reduce task %s\n", mr.address, mr.jobName)
		}else {fmt.Printf("%s: Starting Map/Reduce task %s\n", mr.address, mr.jobName)}
			

		switch mr.phase {
			case mapPhase:
				schedule(mapPhase)
				mr.overwriteTaskLog(reducePhase)
				mr.setupNotCompletedIndexes(reducePhase)
				schedule(reducePhase)
				mr.overwriteTaskLog(mergePhase)
				mr.setupNotCompletedIndexes(mergePhase)
				if (!mr.hasAborted) {
					finish()
					mr.merge()
				}
			case reducePhase:
				schedule(reducePhase)
				mr.overwriteTaskLog(mergePhase)
				mr.setupNotCompletedIndexes(mergePhase)
				if (!mr.hasAborted) {
					finish()
					mr.merge()
				}
			
			case mergePhase:
				if (!mr.hasAborted) {
					finish()
					mr.merge()
				}
		}

		if (mr.hasAborted) {
			return	
		}
		
		fmt.Printf("%s: Map/Reduce task completed\n", mr.address)
		// can use this to measure how long the MapReduce distributed takes;
		// but given that we are running this on the simple machine, the speedup isn't very obvious...
		secs := time.Since(startTimeStamp).Seconds()
		fmt.Printf("job: %s with [%d] mappers and [%d] reducers completes in %fseconds\n", jobName, len(files), nreduce, 	secs)
			
		mr.doneChannel <- true
		mr.deleteTaskLog()
		mr.deleteWorkerLog()
	
	}()			//closes goroutine


	select {
        	case <-mr.abortChannel:
			mr.hasAborted = true
            		return
        	case <-mr.doneChannel:
			mr.doneChannel <- true
			return
	}
        
}				//closes run
	
// Wait blocks until the currently scheduled work has completed.
// This happens when all tasks have scheduled and completed, the final output
// have been computed, and all workers have been shut down.
func (mr *Master) Wait() {
	<-mr.doneChannel
}

// killWorkers cleans up all workers by sending each one a Shutdown RPC.
// It also collects and returns the number of tasks each worker has performed.
func (mr *Master) killWorkers() []int {
	mr.Lock()
	defer mr.Unlock()
	ntasks := make([]int, 0, len(mr.workers))
	for _, w := range mr.workers {
		debug("Master: shutdown worker %s\n", w)
		var reply ShutdownReply
		ok := call(w, "Worker.Shutdown", new(struct{}), &reply)
		if ok == false {
			fmt.Printf("Master: RPC %s shutdown error\n", w)
		} else {
			ntasks = append(ntasks, reply.Ntasks)
		}
	}
	return ntasks
}