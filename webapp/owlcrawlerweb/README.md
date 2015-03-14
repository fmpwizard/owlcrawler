go-examples
===========

Simple application using Go and Twitter Flight
==============================================

# How to use

1. `git clone https://github.com/fmpwizard/go-examples.git`
2. `cd go-examples`
2. `git checkout flightjs-comet`
3. `bower install` //This will download all the js dependencies
4. `npm install` //This will install more dependencies related to packaging the js code, etc
4. When using Go, you need to set the `GOPATH` to your current project's filder, so:  `export GOPATH=/home/diego/work/golang/delete/go-examples`
5. `cd src/github.com/fmpwizard/chat`
6. `go get` //this will download, build and install the dependencies based on the import statements on the `chat.go` file
7. Now go back to the root of the project, where the `run.sh` file is and run:
7. `grunt requirejs` //This step will run js lint and concatenate all your js files into one, and it will minify it too
7. `grunt` //This step will create the main.min.css, as well as the js files for the app to work.
8. Finally, run `./bin/chat --root-dir= <path to current dir>` and it will start a web server at `http://127.0.0.1:7070`


## Playing around with it

If you would like to make changes to the js code and see the changes take effect, you can run `grunt` in `watch` mode, so that the changes will become live.

Simply run `grunt watch` from the root of the project (where the Gruntfile.js file is)

And when you make changes to the go code, you will have to stop and start the app again. There is a `./run.sh` file you can use to start the app (you will need to edit it to fit your needs)
