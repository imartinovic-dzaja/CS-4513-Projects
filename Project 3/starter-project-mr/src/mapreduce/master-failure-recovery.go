package mapreduce

import (
	"fmt"
	"os"
	"bufio"
	"errors"
	"strconv"
)

/*removes the specified string from the masters notCompletedInexes field*/
/*if the string is not in the list simply returns nil*/
func (mr *Master) removeFileFromNotCompleted (task string) error {
	if !mr.hasAborted {
		//we first need to find the index of the task	
		i:=0
		list := mr.notCompletedIndexes	
		for ; i<len(list); i++ {
			if list[i] == task {
				break
			}
		}
		if i == len(list) {
			return errors.New("Task not found in notCompleted!")
		} else {
			list[len(list)-1], list[i] = list[i], list[len(list)-1]
    			mr.notCompletedIndexes = list[:len(list)-1]
	
			return nil	
		}
	}
	return nil
}

/*sets indexes in the notCompletedIndexes field of the master based on the phase*/
/*if it is the map phase then the indexss range from 0 to the number of input files*/
/*if it is the reduce phase then the indexes range from 0 to the number of reducers */
/*if it is the merge phase then the field is set to nil*/
func (mr *Master) setupNotCompletedIndexes(phase jobPhase) {
	if !mr.hasAborted {
		switch phase {
			case mapPhase:
				for i:=0; i<len(mr.files); i++ {
					mr.notCompletedIndexes = append(mr.notCompletedIndexes, strconv.Itoa(i))	
				}
			case reducePhase:	
				for i:=0; i<mr.nReduce; i++ {
					mr.notCompletedIndexes = append(mr.notCompletedIndexes, strconv.Itoa(i))
				}
		
			default:
				mr.notCompletedIndexes = nil
		}
	}
}

/*recovers state information from a failed master based on the master_task_log.log and master_worker_log.log files*/
func (mr *Master) recoverMaster() bool {
	if !mr.hasAborted {
		fmt.Printf("recoverMaster: Attempting to recover master!\n")
		/*open the master logfile for recovery*/
		masterLogFile, err := os.OpenFile("master_task_log.log", os.O_CREATE|os.O_RDONLY, 0755)
		checkError(err)
		
		
		// returns a Scanner type; read from file
		// scans lines
		input := bufio.NewScanner(masterLogFile)
		
		//the first line identifies the jobname, delete all contents of the logfile if it doesn't match the masters jobname	
		input.Scan()
		jobName := input.Text()
		if (jobName != mr.jobName) {
			//close masterLogFile first
			err := masterLogFile.Close()
			checkError(err)
			os.Remove("master_log.log")
			fmt.Printf("recoverMaster: Failed to recover master! Read: %s, Expected %s\n", jobName, mr.jobName)
			mr.overwriteTaskLog(mapPhase)
			mr.overwriteWorkerLog()
			mr.setupNotCompletedIndexes(mapPhase)
			return false
		} else {
			//if the job name does match proceed to reading the phase 
			fmt.Printf("recoverMaster: jobName matches!\n")
			input.Scan()
			/*read the job phase*/
			mr.phase = jobPhase(input.Text())
			
			/*initial not completed tasks will be different depending on the phase*/
			mr.setupNotCompletedIndexes(mr.phase)
			fmt.Printf("recoverMaster: Phase is %s!\n", string(mr.phase))
	
			//now read the lines of the file which represent completed tasks for the corresponding phase
			for input.Scan() {
	
				intermediateFile := input.Text()
				mr.removeFileFromNotCompleted(intermediateFile)
				fmt.Printf("recoverMaster: Found completed task %s!\n", string(intermediateFile))
			}
			
			//close masterLogFile 
			err := masterLogFile.Close()
			checkError(err)
	
			//recover the workers as well
 			mr.recoverWorkers()
	
			fmt.Printf("recoverMaster: Successfully recovered master!\n")
			return true
		}
	
	}
	return false
}


/*overwrites the master-task-log.log file based on the jobPhase argument*/
/*in short it deletes the existing file and creates a new file with the job name on the first line
and the phase name on the second line
*/
func (mr *Master) overwriteTaskLog(phase jobPhase) {
	if !mr.hasAborted {

		
		//write to file, but do not append, simply wipe all data 
		mr.deleteTaskLog()
		masterLogFile, err := os.OpenFile("master_task_log.log", os.O_WRONLY|os.O_CREATE, 0600)
		checkError(err)
		// close masterLogFile on exit and check for its returned error
		 defer func() {
			 err := masterLogFile.Close()
			 checkError(err)
		}()
		mr.phase = phase
		masterLogFile.WriteString(mr.jobName + "\n")
		masterLogFile.WriteString(string(mr.phase) + "\n")
		fmt.Printf("overwriteTaskLog: New jobName %s \t new phase %s!\n", mr.jobName, mr.phase)
		
		mr.notCompletedIndexes = nil
	}
}

/*updates the master-task-log.log file by appending the given string to the file*/
/*the string should represent the completed task index*/
func (mr *Master) updateTaskLog(filename string) {
	mr.Lock()
	if !mr.hasAborted {

		//write to file, appending the filename
		masterLogFile, err := os.OpenFile("master_task_log.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		checkError(err)
		// close masterLogFile on exit and check for its returned error
		 defer func() {
			 err := masterLogFile.Close()
			 checkError(err)
		}()
	
		masterLogFile.WriteString(filename + "\n")
		fmt.Printf("updateTaskLog: Adding %s to log file!\n", filename)
		fmt.Printf("updateTaskLog: Updated log file!\n")
	}
	mr.Unlock()
}

/*deletes master-task-log.log*/
func (mr *Master) deleteTaskLog() {
	if !mr.hasAborted {
		os.Remove("master_task_log.log")
	}
}

/*recovers registered workers for a master based on master_worker_log.log files*/
func (mr *Master) recoverWorkers() bool{
	if !mr.hasAborted {
		fmt.Printf("recoverWorkers: Attempting to recover workers!\n")
		/*open the master logfile for recovery*/
		masterLogFile, err := os.OpenFile("master_worker_log.log", os.O_CREATE|os.O_RDONLY, 0755)
		checkError(err)
		
		
		// returns a Scanner type; read from file
		// scans lines
		input := bufio.NewScanner(masterLogFile)
		
		//the first line identifies the jobname, delete all contents of the logfile if it doesn't match the masters jobname
		input.Scan()
		jobName := input.Text()
		if (jobName != mr.jobName) {
			//close masterLogFile first
			err := masterLogFile.Close()
			checkError(err)
			fmt.Printf("recoverWorkers: Failed to recover workers! Read: %s, Expected %s\n", jobName, mr.jobName)
			mr.overwriteWorkerLog()
			return false
		} else {
			fmt.Printf("recoverWorkers: jobName matches!\n")
			//now recover workers
			//create a temporary list of workers
			var workers []string
			for input.Scan() {
				workerName := input.Text()
				workers = append(workers, workerName)
				fmt.Printf("recoverWorkers: Found worker %s!\n", workerName)
			}
			
			//close masterLogFile 
			err := masterLogFile.Close()
			checkError(err)
			
			//now overwrite the log file, so we don't have copies of same workers
			mr.overwriteWorkerLog()
			
			/*now register these workers to our new master*/
			var args ReRegisterArgs
			args.Master = mr.address
			for _, workerName := range workers {
				//send an RPC to all workers so they register with this new master
				call(workerName, "Worker.ReRegister", &args, new(struct{}))
			}
	
			fmt.Printf("recoverWorkers: Successfully recovered workers!\n")
			return true
		}
	}	
	return false

}

/*overwrites the master-worker-log.log file*/
/*in short it deletes the existing file and creates a new file with the job name on the first line*/
func (mr *Master) overwriteWorkerLog() {
	if !mr.hasAborted {
		
		//write to file, but do not append, simply wipe all data 
		mr.deleteWorkerLog()
		masterLogFile, err := os.OpenFile("master_worker_log.log", os.O_WRONLY|os.O_CREATE, 0600)
		checkError(err)
		// close masterLogFile on exit and check for its returned error
		 defer func() {
			 err := masterLogFile.Close()
			 checkError(err)
		}()
		masterLogFile.WriteString(mr.jobName + "\n")
		fmt.Printf("overwriteWorkerLog: New jobName %s \n", mr.jobName)
		fmt.Printf("overwriteWorkerLog: Overwritten log file!\n")
	}
}

/*updates the master-worker-log.log file by appending the given string to the file*/
/*the string should represent the registered worker address*/
func (mr *Master) updateWorkerLog(workerName string) {
	mr.Lock()
	if !mr.hasAborted {
		//write to file, appending the filename
		masterLogFile, err := os.OpenFile("master_worker_log.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		checkError(err)
		// close masterLogFile on exit and check for its returned error
		 defer func() {
			 err := masterLogFile.Close()
			 checkError(err)
		}()
	
		masterLogFile.WriteString(workerName + "\n")
		fmt.Printf("updateWorkerLog: Adding %s to log file!\n", workerName)
	
	}	
	mr.Unlock()
}

/*deletes the master-worker-log.log*/
func (mr *Master) deleteWorkerLog() {
	if !mr.hasAborted {
		os.Remove("master_worker_log.log")
	}
}

