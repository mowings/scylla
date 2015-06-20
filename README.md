# Scylla - centralized, modern multi-host cron
Scylla is a job management system that allows cron-like functionality centralized on a single host, using  ssh instead of remote agents to run jubs remotely. Scylla was inspired by Yelp's excellent [tron](https://github.com/Yelp/Tron) package, but offers a number of improvements.

## Features
* No dependencies, small footprint. Scylla is distributed as a pair of executables. Binaries are available as tar files and .deb files (see below).
* ssh-based. No remote agents required. Offers both connect and read-timeouts to detect hung jobs
* Run jobs on single hosts, or pools of hosts. Jobs run across pools can run round-robin (1 host chosen per job) or in parallel
* Dead simple configuration
* Alert on failures
* Built in web server
* Full API, including calls to run jobs and update host pools

## Installing it
### From a package
If you install scylla via the debian or rpm files, you are done.

### From the archive
You can also install from the tar file. Once you have extracted the files from the archive, copy the `scyd/` directory to either `/opt` or `/usr/local`. Copy the scyctl utility to anywhere on your path. 

If you want to run as a non-privileged user (recomended), you will need to manually create `/var/run/scylla` and `/var/lib/scylla`, create the user you wish to run as and change ownership of those directories to that of the scylla user. You should also copy the sample `scylla.conf` to /etc/scylla.

## Running it
* Change to the `scyd/` directory
* As root, run either `./scyd` or (better) `sudo -u <non_privileged_user> ./scyd`

Scyd logs to stdout. Note that if you run as a non-privileged user, you need to be sure `/var/run/scylla` and `/var/lib/scylla` exist and are owned by that user.

## Building it
Install go. I'd suggest not using a package manager.
```
(be sure you have set $GOPATH)
$ mkdir -p $GOPATH/src/github.com/mowings
$ cd $GOPATH/src/github.com/mowings
$ git clone https://github.com/mowings/scylla.git
$ cd scylla/scyd
$ go build
$ cd ../scyctl
$ go build
````
This will leave the binaries `scyctl` and `scyd` in their respective source directories

## Getting Started

