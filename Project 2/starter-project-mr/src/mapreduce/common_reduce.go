package mapreduce


import (
	"os"
	"encoding/json"
)

// doReduce does the job of a reduce worker: it reads the intermediate
// key/value pairs (produced by the map phase) for this task, sorts the
// intermediate key/value pairs by key, calls the user-defined reduce function
// (reduceF) for each key, and writes the output to disk.
func doReduce(
	jobName string, // the name of the whole MapReduce job
	reduceTaskNumber int, // which reduce task this is
	nMap int, // the number of map tasks that were run ("M" in the paper)
	reduceF func(key string, values []string) string,
) {
	// TODO:
	// You will need to write this function.
	// You can find the intermediate file for this reduce task from map task number
	// m using reduceName(jobName, m, reduceTaskNumber).
	// Remember that you've encoded the values in the intermediate files, so you
	// will need to decode them. If you chose to use JSON, you can read out
	// multiple decoded values by creating a decoder, and then repeatedly calling
	// .Decode() on it until Decode() returns an error.
	//
	// You should write the reduced output in as JSON encoded KeyValue
	// objects to a file named mergeName(jobName, reduceTaskNumber). We require
	// you to use JSON here because that is what the merger than combines the
	// output from all the reduce tasks expects. There is nothing "special" about
	// JSON -- it is just the marshalling format we chose to use. It will look
	// something like this:
	//
	// enc := json.NewEncoder(mergeFile)
	// for key in ... {
	// 	enc.Encode(KeyValue{key, reduceF(...)})
	// }
	// file.Close()
	//
	// Use checkError to handle errors.
	
	//create a map which contains all of the keys and a slice for their values
	KVs := make(map[string] []string)	
	
	//read the key-value pairs in a file
	for i := 0; i < nMap; i++ {
		//figure out the filename
		filename := reduceName(jobName, i, reduceTaskNumber)

		//open the file to read the jsons
		fi, err := os.OpenFile(filename, os.O_RDONLY, 0755)
		checkError(err)

		//read the jsons
		dec := json.NewDecoder(fi)


		// while the decoder is encountering json objects
		for dec.More() {
			// decode the object
			var kv KeyValue	
			err = dec.Decode(&kv)
			checkError(err)

			//append the newly read value for its corresponding key to the appropriate slice inside the map
		 	KVs[kv.Key] = append(KVs[kv.Key], kv.Value)
		}
		
		
		fi.Close() //close the file

	}
	
	//create filename to which to write the output to
	mergeFileName := mergeName(jobName, reduceTaskNumber)
	//delete previously existing output file
	os.Remove(mergeFileName) //ignore error if file does not exist	


	//open the file to which to write the output to 
	fo, err := os.OpenFile(mergeFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	checkError(err)
	
	//create a json encoder to encode the results
	enc := json.NewEncoder(fo)	

	//for each key in our map, call the reduceF function and store its results in our output file
	for key, values := range KVs {
		var kv KeyValue
		kv.Key = key
		kv.Value = reduceF(key, values)
		err = enc.Encode(&kv)
		checkError(err)

	}
	
	//close the output file when done
	fo.Close()
	

}
