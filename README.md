# Scylla - centralized, modern multi-host cron
Scylla is a job management system that allows cron-like functionality centralized on a single host, using  ssh instead of remote agents to run jubs remotely. Scylla was inspired by Yelp's excellent [tron](https://github.com/Yelp/Tron) package, but offers a number of improvements.

## Features
* No dependencies, small footprint. Scylla is distributed as a pair of executables and a few supporting files. Binaries are available as tar files and .deb files for 64 and 32-bit versions of linux. You can target other systems if you build from source.
* ssh-based. No remote agents required. Offers both connect and read-timeouts to detect hung jobs
* Run jobs on single hosts, or pools of hosts. Jobs run across pools can run round-robin (1 host chosen per job) or in parallel
* Dead simple configuration
* Simplified support for sudoed jobs
* Support for file uploads
* Alert on failures
* Built in web server
* Full API, including calls to run jobs and update/create host pools

## Installing it
Binary packages are stored in github with each release. Click `releases` to see the available releases.

### From a package
If you install scylla via the debian files, you are done. 

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
Scylla is configured via an ini-formatted  config file, `/etc/scylla.conf` You should have a bare-bones version of this file installed already. If not, go ahead and set that up now:

### Setting up the web listener and defaults
```
[web]
listen = "0.0.0.0:8080"

[defaults]
# keyfile="keys/secret"
connect-timeout=10
read-timeout=0 # Default, 1 day
sudo-command = "sudo -i /bin/bash -c"
user=scylla
```
The `listen` directive tells us where to listen for the web UI and API calls. You may wish to restrict the address to localhost, as there is no security on either interface built in. You can use any proxy server (nginx works well) to add basic authentication and restrict api access as required.

You will need to set at least a single private ssh key file in the `[defaults]` section that can be used to log in to remote hosts. To add more key files, add more `keyfile` entries here. Note that scylla does not support password authentication at all, and ssh-agent support is not built in (although it is planned). You should also set a default `user` to login to remote hosts. Note that keys and user names can be overridden easily in individual jobs, but defaults are a good idea.

### Adding a simple job
Let's add a simple job that runs `uptime` on a single remote host. To add a new job, create a new `job` section in `scylla.conf` with the job name, host and command:

```
[job "test"]
description = Scylla test job
host = scylla@foo.example.com:22
command = uptime
```
These directives should be self-explanatory, with the exception of `host`. A scylla host is specified as 

    [user@]host[:port]
    
Of course, we specified a default user in our `defaults` section and, sensibly, the destination port defaults to 22, so we can really write this job section as:

```
[job "test"]
description = Scylla test job
host = foo.example.com
command = uptime
```
But it's good to know we can change the user and port as needed. Note that you can have multiple command attributes for a job; each command will be run in sequence.

Once the job has been added, tell scyd to update itself with the reload command:

    scyctl reload

As written, `test` is a usable job -- but without a schedule it can only be run manually via either the API or via scyctl: `scyctl run test`. So let's add a schedule:

```
[job "test"]
description = Scylla test job
host = foo.example.com
command = uptime
schedule = cron */5 * * * *
```

Now our job will run at 5-minute intervals. Run `scyctl reload` to reload the configuration

The cron schedule type  use a standard (numeric only) cron format, where each section represents:
* minute (0-59)
* hour (0-23
* month day (1-31)
* month(1-12)
* day of week(0-6), where 0 is Sunday

Each section can include a range(`-`), interval (`*/n`) and multiple comma-separated values. This is a very basic cron schedule, although enhancements are in the works to support non-numeric days of the week, last-day of the month and nth dow of the month (second wed, last sun, etc).

We can also define pools of hosts to run a single job on:

```
[pool "webservers"] # Use user "foo" and port 2222
host = foo@webserver-1.foo.bar:2222
host = foo@@webserver-2.foo.bar:2222
host = foo@@webserver-2.foo.bar:2222

[job "test"]
description = Scylla test job
pool = webservers
command = uptime
schedule = cron */5 * * * *
```

Each time the test job runs, it will choose a different host from the pool to run on (in a round-robin fashion). If you want the job to run on all hosts in the pool at the same time, change the job's pool attribute from `pool=webservers` to `pool = webservers parallel`

If you add `dynamic = yes` to any pool definition, the pool hosts can be updated via the api. You can also update pool hosts via scyctl by piping a list of hosts (one-per-line) into the update_pool command. So for example:

    <some_command> | scyctl update_pool webservers

We can specifiy that we would like to be notified when a job fails. First set up a notification in the config file:

```
[notifier "slack"]
path=./notifiers/slack # executable file that sends a message to our team via slack
arg=team
arg=token
arg=channel

# tell our job to use the slack notifier
[job "test"]
description = Scylla test job
host = foo.example.com
command = uptime
schedule = cron */5 * * * *
notifier = slack
```
Now, whenever our job fails, we will get a notification via slack. By default, the notifier is triggered every time a job fails, and then once after the first sucess. You can also specifiy `edge-trigger = true` to only alert on the first failure, and to then stay silent until the first success. Finally, set 'always' to true if you wish to be alerted on all runs.

The `path` attribute in a notifier section refers to any executable file. When the notification fires, this file will be run with at least three parameters (in this order):

* run status (as a string) ie, `succeeded` `failed`
* The job name
* The integer run id

Additionally, any `args` specified in the notifier section will be passed in the sequence specified.

## More Documentation?
Not yet. Please have a look at test.conf in scyd/config for more info on config file format. More documentation should be coming soon.
