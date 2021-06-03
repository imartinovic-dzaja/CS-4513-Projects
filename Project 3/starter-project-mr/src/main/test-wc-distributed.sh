# usage:
# $cd main/
# ./test_wc_distributed.sh
# known issue: if the script appears to hang, you can try to manually delete all the connections under e.g., 
#!/bin/bash

# get the UID, and current process ID to create uniq Unix socket names
u=$UID
p=$BASHPID
echo $u, $p

# create the tmp directory if it is not already exisiting
# note, Ubuntu will throw error (address binding error) if the unix socket is the current directory
# if you encounter issues, you can try to manually create the /var/tmp/824-$u-$p by substituting the specific values
mkdir -p /var/tmp/824-$u
chmod 777 /var/tmp/824-$u




# submit the wordcount job to the MapReduce library (distriubted mode)
go run wc.go master /var/tmp/824-$u/mr$p-master pg-*.txt &
MASTERPID=$!

# start 3 workers
go run wc.go worker /var/tmp/824-$u/mr$p-master /var/tmp/824-$u/mr$p-worker0 &
go run wc.go worker /var/tmp/824-$u/mr$p-master /var/tmp/824-$u/mr$p-worker1 &
go run wc.go worker /var/tmp/824-$u/mr$p-master /var/tmp/824-$u/mr$p-worker2 &




# wait until master is done, before comparing the output
wait $MASTERPID

# compare the execution result: mrtmp.wcseq with the ground truth: mr-testout.txt
sort -n -k2 mrtmp.wcseq | tail -10 | diff - mr-testout.txt > diff.out
if [ -s diff.out ]
then
echo "Failed test. Output should be as in mr-testout.txt. Your output differs as follows (from diff.out):" > /dev/stderr
  cat diff.out
else
  echo -e "\e[32mpassed test case! \e[0m" > /dev/stderr
fi

# clean up
# you could uncomment the first two lines, if you wish to examine the intermediate outputs
echo "cleaning up..."
rm mrtmp.*
rm diff.out
rm /var/tmp/824-$u/*
killall wc
