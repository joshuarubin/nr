# nr

```
Usage of ./nr:

	./nr file1.txt file2.txt ...
	cat file1.txt | ./nr

	A filename argument of '-' indicates that stdin should be read.
	If no filenames are given, input is assumed to come from stdin.

flags:
  -encoding string
    	file encoding of all files, including stdin, valid values defined at https://www.w3.org/TR/encoding/
  -n int
    	only show the top n sequences with the highest frequency count (default 100)
  -sequence-size int
    	number of words per sequence (default 3)
```
