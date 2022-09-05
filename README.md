# Go Rest JSON Client
This small package offers a convenience interface and struct for interacting with JSON based APIs. I've copy/pasted this
code enough between different projects of mine that I finally just extracted it to a package.

It is intended to be used with JSON based APIs where requests and responses are already defined as structs (with appropriate tags). 
I recommend using something like https://mholt.github.io/json-to-go/ to quickly convert JSON examples to Go structs.

## Example Usage
