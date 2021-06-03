#!/bin/bash

# submit the wordcount job to the MapReduce library (Sequential mode)
go run wc.go master sequential pg-*.txt

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
# you could uncomment the following two lines, if you wish to examine the intermediate outputs
rm mrtmp.*
rm diff.out

