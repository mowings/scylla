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


