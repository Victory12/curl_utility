# curl

Simple tool to make http requests in parallel and print MD5 hash of the response body

## Description

- The tool accepts websites as inputs and make http requests, if request is successful, prints the
   sitename allong with MD5 hash of the response body.

## Flags
- Flag ```-parallel``` is used to set how many requests can be processed parallely for the given input websites. Default is 1, limit is 10. If the parameter is greater than the number of urls, it will be reduced to their number.

## Usage

- Clone the repository using git clone.
- Build the project using command 
```
      go build
```
- Run the program using command
```
      ./curl -parallel <no of parallel sites to be processed> <list of sites separated by space>
```
- To run, this alternate command also can be used
```
      ./curl -parallel=<no of parallel sites to be processed> <list of sites separated
```
- e.g.
```
      ./curl -parallel 3 google.com www.fb.com http://yahoo.com
```
- To run the test cases, run command- 
```
      go test ./...
```
## Assumptions

* The website are fetched using GET request.
* Http client go througth redirects, so be carefull it's not so safe. If can be future feature to turn off handling redirects.
* If there are several identical urls in the input, they will be combined.
* Timeouts are defined in the code. It's good to add possibility for getting them from input params.
* If any website fetching is failed, the program will return
	* "${host} error:${status}" in case of response with status that not equal to 200
	* "${host} error:timeout" in case of client timeout.
	* "${host} error:timeout" in case of unknown error. Error will be print to Stderr.

