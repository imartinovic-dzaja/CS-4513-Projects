package mapreduce

import (
	"strconv"
	"fmt"
)

// schedule starts and waits for all tasks in the given phase (Map or Reduce).
func (mr *Master) schedule(phase jobPhase) {
	var ntasks int
	var nios int // number of inputs (for reduce) or outputs (for map)
	switch phase {
	case mapPhase:
		ntasks = len(mr.files)
		nios = mr.nReduce
	case reducePhase:
		ntasks = mr.nReduce
		nios = len(mr.files)
	}

	debug("Schedule: %v %v tasks (%d I/Os)\n", ntasks, phase, nios)

	for _, file := range mr.notCompletedIndexes {
		fmt.Printf("Schedule: NotCompleted: %s\n", file)
	}
	
	// All ntasks tasks have to be scheduled on workers, and only once all of
	// them have been completed successfully should the function return.
	// Remember that workers may fail, and that any given worker may finish
	// multiple tasks.
	//
	// TODO:
	//this channel blocks until it is ready to proceed to next phase
	proceed := make (chan bool)
	

	completed :=  ntasks - len(mr.notCompletedIndexes)

	if completed == ntasks {
		proceed<-true
	}

	go func() {
		for i := 0; i < len(mr.notCompletedIndexes) && !mr.hasAborted; i++ {	//i counts number of tasks assigned so far
			var args DoTaskArgs
			index, _ := strconv.Atoi(mr.notCompletedIndexes[i])
			args.JobName = mr.jobName
			args.File = mr.files[index] // input file, only used in map tasks
			args.Phase = phase 		// are we in mapPhase or reducePhase?
			args.TaskNumber = index		// this task's index in the current phase

			// NumOtherPhase is the total number of tasks in other phase; mappers
			// need this to compute the number of output bins, and reducers needs
			// this to know how many input files to collect.
			args.NumOtherPhase = nios
			//this nesteded goroutine sends an rpc to a worker
			if (mr.hasAborted) {
				return
			}	

			go func () {
				if (mr.hasAborted) {
					return
				}
				workerName := <- mr.registerChannel //only schedule tasks when a worker is free
				if (mr.hasAborted) {
					return	
				}

				for call(workerName, "Worker.DoTask", args, new(struct{})) == false {
					//if the worker fails assign the same task to another free worker
					workerName = <- mr.registerChannel
				}
					
				if (mr.hasAborted) {
					return	
				}

				//once the worker completes update the number of completed tasks
				switch phase {
					case mapPhase:			
						mr.updateTaskLog(strconv.Itoa(index))	//log that the job is finished
					case reducePhase:
						mr.updateTaskLog(strconv.Itoa(index)) //log that the job is finished	
					
				}

				if (mr.hasAborted) {
					return	
				}

				mr.Lock()
				completed++
				//once all tasks complete, we proceed
				if (completed == ntasks) {
					proceed<-true
				}
				
				mr.Unlock()

				if (mr.hasAborted) {
					return	
				}
				mr.registerChannel <- workerName	//signal the worker is free again
				
			} ()		
		} //closes for loop
	}() //closes nested goroutine	


	if (mr.hasAborted) {
		return
	}		
	
	//proceed to next phase only when appropriate
	<-proceed
	debug("Schedule: %v phase done\n", phase)
}
