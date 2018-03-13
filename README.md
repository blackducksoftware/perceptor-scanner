# Pardon our dust, this repo is still heavily under construction / migration.

## Perceptor-scanner

A pod for scanning images by combining the powers of the Hub and Perceptor!

## Structure

The perceptor-scanner abstraction includes the image facade and the scanning implementation.  

Both or either are replacable, so long as the API calls and responses are effectively the same.

## Repository structure

 Perceptor-scanner pod:  Consisting of 2 conatiners:
 - perceptor-scanner container ~ downloads a scanclient from the hub upon startup
 - perceptor-imagefacade container ~ handles image data, metadata.
   
## Getting involved

These repositories represent refactoring of previous work in blackduck's ose-scanner, and the blackduck perceptor prototype release which is in alpha now.  To get involved, create an issue, submit a PR, or directly ping the developers in your issue).

We welcome new contributors !

## See also

Companion repositories: perceptor, perceivers, perceptor-protoform.  

