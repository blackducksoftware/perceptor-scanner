## Perceptor-scanner

A pod for scanning images by combining the powers of the Hub and Perceptor!

## Structure

This repo includes the standard image facade and scanning implementations.

Both or either may be swapped out by alternate implementations, so long as the REST APIs are maintained.

## Intended use

 Perceptor-scanner pod consisting of 2 containers:
 - perceptor-imagefacade: makes tar files of docker images available for scanning
 - perceptor-scanner: downloads a scan client from the hub upon startup, and uses the scan client to scan docker images pulled by the imagefacade
 
## Testing

 - PifTester -- ties together real perceivers and the image facade, for image facade testing
 - ScannerTester -- uses a mock image facade to test the scanner

## Getting involved

To get involved, create an issue, submit a PR, or directly ping the developers in your issue.

We welcome new contributors!

## See also

Companion repositories: 

 - [perceptor](https://github.com/blackducksoftware/perceptor)
 - [perceivers](https://github.com/blackducksoftware/perceivers)
 - [perceptor-protoform](https://github.com/blackducksoftware/perceptor-protoform)
 - [skyfire](https://github.com/blackducksoftware/perceptor-skyfire)

