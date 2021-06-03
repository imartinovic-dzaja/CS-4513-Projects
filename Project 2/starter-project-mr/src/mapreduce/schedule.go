package mapreduce

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

	// All ntasks tasks have to be scheduled on workers, and only once all of
	// them have been completed successfully should the function return.
	// Remember that workers may fail, and that any given worker may finish
	// multiple tasks.
	//
	// TODO:
	//this channel blocks until it is ready to proceed to next phase
	proceed := make (chan bool)
	completed := 0		//counts number of completed tasks in the current phase, protected by mutex
	
	//this goRoutine assigns ntasks to workers
	go func() {
		for i := 0; i < ntasks; i++ {
			var args DoTaskArgs
			args.JobName = mr.jobName
			args.File = mr.files[i] // input file, only used in map tasks
			args.Phase = phase 	// are we in mapPhase or reducePhase?
			args.TaskNumber = i	// this task's index in the current phase

			// NumOtherPhase is the total number of tasks in other phase; mappers
			// need this to compute the number of output bins, and reducers needs
			// this to know how many input files to collect.
			args.NumOtherPhase = nios
			
			//this nesteded goroutine sends an rpc to a worker
			go func () {
				workerName := <- mr.registerChannel //get the name of the worker that is free
				for call(workerName, "Worker.DoTask", args, new(struct{})) == false {
					//if the worker fails assign the same task to another free worker
					workerName = <- mr.registerChannel
				}	
				//once the worker completes update the number of completed tasks
				mr.Lock()
				completed++
				
				//if we have completed all the tasks, we may proceed
				if completed == ntasks {
					proceed <- true
				}
				mr.Unlock()

				mr.registerChannel <- workerName	//signal the worker is free again
			}() //closes nested goroutine	
		} //closes for loop
	
	}() //closes "big" goroutine			
	
	//proceed to next phase only when appropriate
	<-proceed

	debug("Schedule: %v phase done\n", phase)
}
