package mapreduce

import (
	"hash/fnv"
	"os"
	"bufio"
	"encoding/json"
	
)

// doMap does the job of a map worker: it reads one of the input files
// (inFile), calls the user-defined map function (mapF) for that file's
// contents, and partitions the output into nReduce intermediate files.
func doMap(
	jobName string, // the name of the MapReduce job
	mapTaskNumber int, // which map task this is
	inFile string,
	nReduce int, // the number of reduce task that will be run ("R" in the paper)
	mapF func(file string, contents string) []KeyValue,
) {
	// TODO:
	// You will need to write this function.
	// You can find the filename for this map task's input to reduce task number
	// r using reduceName(jobName, mapTaskNumber, r). The ihash function (given
	// below doMap) should be used to decide which file a given key belongs into.
	//
	// The intermediate output of a map task is stored in the file
	// system as multiple files whose name indicates which map task produced
	// them, as well as which reduce task they are for. Coming up with a
	// scheme for how to store the key/value pairs on disk can be tricky,
	// especially when taking into account that both keys and values could
	// contain newlines, quotes, and any other character you can think of.
	//
	// One format often used for serializing data to a byte stream that the
	// other end can correctly reconstruct is JSON. You are not required to
	// use JSON, but as the output of the reduce tasks *must* be JSON,
	// familiarizing yourself with it here may prove useful. You can write
	// out a data structure as a JSON string to a file using the commented
	// code below. The corresponding decoding functions can be found in
	// common_reduce.go.
	//
	//   enc := json.NewEncoder(file)
	//   for _, kv := ... {
	//     err := enc.Encode(&kv)
	//
	// Remember to close the file after you have written all the values!
	// Use checkError to handle errors.
//opening the input file and getting its file descriptor
	fi, err := os.OpenFile(inFile, os.O_RDONLY, 0755)
	checkError(err)
	
	// close f on exit and check for its returned error
	 defer func() {
		 err := fi.Close()
		 checkError(err)
	}()	
	
		
	// returns a Scanner type; read from file
	// scans lines
	input := bufio.NewScanner(fi)
	var contents string		//container for storing contents of file so it can be passed to mapF
	for input.Scan() {
		contents += input.Text()
		contents += "\n"	
	}
	
	//call mapF on the file, provide the filename and contents
	mapFOutput := mapF(inFile, contents)
	
	//delete previously existing intermediate files and create new ones
	inputFiles := make([]*os.File, nReduce) 
	for i := 0 ; i < nReduce; i++ {
		intermediateFilename := reduceName(jobName, mapTaskNumber, i)
		os.Remove(intermediateFilename) //ignore error if file does not exist
		
		inputFiles[i], _ = os.OpenFile(intermediateFilename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)

		defer inputFiles[i].Close();
	}
	
	//write the key-value pairs as jsons to appropriate intermediate file
	for _, kv := range mapFOutput {

		fo := inputFiles[int(ihash(kv.Key))%nReduce]	//use hashing and modular arithmetic to decide filename
		
		enc := json.NewEncoder(fo)
		err = enc.Encode(&kv)

		checkError(err)

	}
}

func ihash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}
