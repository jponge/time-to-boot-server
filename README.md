# `time-to-boot-server` ~ Measure the time to boot a server and make a first connection

This is a simple tool to measure how long it takes for a server application to boot and succeed in receiving a first network request.

## Usage

You can run it as in:

    time-to-boot-server --target http://localhost:8080/ --executable python -- -m SimpleHTTPServer 8080

Run with `--help` to get a list of all arguments.

There are 2 connection modes:

* `http-get`: succeeds on the first HTTP GET request with a 200 status code, and consumes all the body
* `tcp-connect`: succeeds on the first established TCP connection, and does not consuje anything.

`tcp-connect` is the fastest time to connect to the server, but for any framework with lazy initialization some components may only be initialized upon the first request so `http-get` is more accurate _in general_.

## Building and running

This is a Go program so...

    go get github.com/jponge/time-to-boot-server

If you don't know how to build Go code:

    mkdir ~/Code/go-workspace
    export GOPATH=~/Code/go-workspace
    export PATH=$GOPATH/bin:$PATH
    
    go get github.com/jponge/time-to-boot-server

## License

MIT, see [LICENSE](LICENSE)