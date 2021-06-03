CS4513: Final Project  The MapReduce Library : Handling Master Failure
=======================================

Team members
-----------------

1. Ivan Martinovic (imartinovic@wpi.edu)

Description:
-------------
This project extends upon my MapReduce Library implementation by adding to it the ability to tolerance failure of a master thread. All work has been done inside the mapreduce folder. Most of the newly written code can be found inside master-failure-recovery.go. I also edited some existing files such as the master.go, schedule.go and worker.go (I will get into details shorty). Testing proved to be difficult. I first wanted to use a bash script which would terminate the master process, however running kill masterID did not seem to kill the master's schedule thread, which lead to improper execution. I was therefore forced to use go for testing, however this was also not optimal as I will explain shortly.

Changes to Master structure
----------------------------
To implement the feature which handles failure of the master thread, I first added a couple fields to the Master structure inside master.go. Namely I added:

phase jobPhase
notCompletedIndexes []string 
abortChannel chan bool
hasAborted bool

jobPhase - the current phase for the job the master is executing

notCompletedIndexes - contains the list of indexes to all files in the current phase of execution which have not yet been completed. For map phase it contains the indexes of files inside the mr.files array which have not yet been completed. For reduce phase, it contains the indexes of the reduce tasks which have not yet been completed. 

abortChannel - a channel used to signal to the master thread (more specifically the run function) to abort execution. This is used primarily for testing purposes, as I found that due to time constraints, to be the only method with which I can reliably shut down the master thread along its goroutines. During testing, at specific time intervals this channel is signalled which causes the master to abort.

hasAborted - holds information whether the master is suppossed to be aborted. This variable is also used primarily for testing purposes. I found it to be the only way to stop goroutines from executing (such as the schedule routine), or at least make them return early. Therefore throughout the master-failure-recovery.go, schedule.go and master.go (for the master, specifically the run function), you will see checks whether this boolean variable is true or false. It's function is to serve as SIGINT and For the real implementation it can be completely removed along its corresponding if statements.

Workflow overview
------------------
The workflow of the master has now changed.
The master now keeps two log files, namely: "master_task_log.log" and "master_worker_log.log"
master_task_log.log contains the following information:
	-name of the job that is executing on the first line
	-current phase of the current job on the second line
	-list of completed tasks for the current phase of the current job on all the subsequent lines

master_worker_log.log contains the following information:
	-name of the job that is executing on the first line
	-list of all workers (i.e. their addresses) who registered to the master on all subsequent lines

Most of our attention will be focused inside the run comman inside master.go
When run starts, it starts a go routine which does the basic execution of mapreduce(on which we will focus on in the next paragraph). Apart from this (at the bottom) it also listens to two channels: the abort channel and the done channel. When doneChannel is singalled, run finishes executing properly. When abortChannel is signalled, run finishes executing by simulating a master failure.

Now back to the goroutine. It calls recoverMaster() defined in master-failure-recovery.go, which reads master_task_log.log
If the file does not exist, or the file contains the a job name not equal to the job name of the master, the master will replace it with a new file containing the proper jobName and the jobPhase set to mapPhase. The master will then also replace the master_worker_log.log with a file containing only the appropriate job name. It then sets up the list of notCompletedIndexes such that it contains the indexes from 0 to the number of files in the files field (because the phase is mapPhase)

If the file does exist and the job name matches that of the master, the master will take this as an indication a master has failed and that there is information to recover from the log. The master then proceeds to read the job phase from the file and updates his own jobPhase field. Next, based on the phase, the master sets up the initial list of indexes i.e. notCompletedIndexes. Then, the master proceeds reading the completed task indexes from the log file. For each task index it enconters, it removes it from the list of notCompletedIndexes. Eventually once all tasks are read, notCompletedIndexes only contains indexes of tasks which have not been completed. Once the master recovers files, it proceeds to recover the workers.
	
	The master reads the "master_worker_log.log" file. If it finds that such a file does not exist, or that its job name is mismatched it will create a new file with the appropriate job name. If it finds that the file exists and that it has the appropriate job name, it proceeds reading the addresses of workers from the subsequent lines. It adds each worker address it encounters to a temporary array. Once done, the master would posses the addresses of workers previously registered to the failed master. The next step would be to ask the workers to re-register to this new master. But before that is done, the master-worker-log.log file is replaced by a file containing only the appropriate job name (to avoid duplicating the workers which we just read -- this will be clearer in a moment).  Next for each of the workers in the temporary array, we do an RPC call to ReRegister defined in worker.go. This simply makes the worker register to a new address using the Master.Register RPC call. Notice now the Master.Register, along with adding the worker address to its mr.workers field, also adds the workers to the master-worker-log.log file. Once all workers are registered, we have completed the recovery of the master. 

Now based on the jobPhase we recovered (jobPhase is mapPhase if we didn't recover anything), we continue execution from that point (implemented in the switch statement). Notice that after each call to schedule we overwrite our task log and list of not completed task indexes according to the next phase. In other words if we just finished our map phase, we wipe all data from the task log and update it to contain the same job name and now the reduce phase. Similarly, our list of notCompleted tasks after finishing map phase will be empty (since all tasks are completed), and therefore we need to update it as well before proceeding to the next schedule task.

Next lets dive into the schedule function since it has also seen some changes. 
Please ignore the code snippets such as:
	if (mr.hasAborted) {
		return
	}
As they are only there to make testing possible. 
Now we keep track of the initial number of completed tasks (line 37) as the difference between the total number of tasks and the number of task indexes stored in notCompletedIndexes. Next, inside the big for loop (line 44), we only assign a number of tasks equal to the number of not completed task indexes. Then inside the loop we only assign task indexes of tasks which have not yet been completed (lines 48 and 49). Then a nested goroutine (line 61) is used to send an RPC to the workers in order to do the specific task. The only difference inside this nested goroutine now is that we are logging the index of the task which completes to our master-task-log.log file.


Testing
--------
I've provided 2 test cases inside test_test.go. Namely TestSingleMasterFailure and TestManyMasterFailures.
TestSingleMasterFailure tests how mapreduce hanles the master thread failing only once after singleFail (specified on line 21) seconds and recovering after and additional singleFailRecovery (specified on line 22)
TestManyMasterFailures tests how mapreduce handles the master thread failing repeatedly every multipleFailsInterval (specified on line 5) seconds. 

The grader should feel free to play with these values, as well as nNumber, nMap and nReduce. The ones provided by me worked best for my laptop, however other values may be more optimal to showcase desired behaviour on laptops with different performances.

I have purposefully left some prinlines to make the behavior more apparent to the grader. 

The output I've obtained when running TestSingleMasterFailure is provided in ivan_test_single_master.txt
The output I've obtained when running TestManyMasterFailures is provided in ivan_test_many_master.txt

Both of these are inside the mapreduce folder

Errata
------

Describe any known errors, bugs, or deviations from the requirements.




---
