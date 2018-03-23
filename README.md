## Perceptor-scanner

A pod for scanning images by combining the powers of the Hub and Perceptor!

## Structure

The perceptor-scanner abstraction includes the image facade and the scanning implementation.  

Both or either are replacable, so long as the API calls and responses are effectively the same.

## Intended use

 Perceptor-scanner pod consisting of 2 containers:
 - perceptor-imagefacade: makes tar files of docker images available for scanning
 - perceptor-scanner: downloads a scanclient from the hub upon startup, and uses the scan client to scan docker images pulled by the imagefacade
 
## Testing

 - PifTester -- ties together real perceivers and the image facade, for image facade testing
 - ScannerTester -- uses a mock image facade to test the scanner

## Getting involved

These repositories represent refactoring of previous work in blackduck's ose-scanner, and the blackduck perceptor.

To get involved, create an issue, submit a PR, or directly ping the developers in your issue.

We welcome new contributors!

## See also

Companion repositories: perceptor, perceivers, perceptor-protoform.  

